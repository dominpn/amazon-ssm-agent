// Copyright 2025 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.
package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"

	"github.com/cihub/seelog"
	"golang.org/x/sync/errgroup"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/fileutil"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
)

// rollingDiskMetricCollector holds a [seelog.LoggerInterface] instance
type rollingDiskMetricCollector struct {
	// directory path
	dirPath string

	logger seelog.LoggerInterface
}

// namespacedDiskMetricCollector holds namespace to [rollingDiskMetricCollector] mapping
type namespacedDiskMetricCollector struct {
	ctx context.T

	baseDir string
	// maximum number of rolled files
	maxRolls int
	// maximum size of one file
	maxFileSize int64

	// prefix of each log file
	fileNamePrefix string
	mtx            *sync.Mutex
	collectorMap   map[string]*rollingDiskMetricCollector
}

// for mocking the write directory since TelemetryDataStorePath is a constant
var getBaseMetricsStoreDir = func() string {
	return filepath.Join(appconfig.TelemetryDataStorePath, "metrics")
}

func NewRollingDiskMetricCollector(context context.T, maxRolls int, maxFileSize int64, fileNamePrefix string) *namespacedDiskMetricCollector {
	return &namespacedDiskMetricCollector{
		ctx:            context,
		baseDir:        getBaseMetricsStoreDir(),
		maxRolls:       maxRolls,
		maxFileSize:    maxFileSize,
		fileNamePrefix: fileNamePrefix,
		mtx:            &sync.Mutex{},
		collectorMap:   make(map[string]*rollingDiskMetricCollector),
	}
}

func (c *namespacedDiskMetricCollector) Collect(namespace string, metric metric.Metric[float64]) error {
	metricBytes, err := json.Marshal(metric)

	if err != nil {
		return err
	}

	rw, err := c.getMetricCollector(namespace)

	if err != nil {
		return err
	}

	err = rw.write(metricBytes)

	return err
}

func (c *namespacedDiskMetricCollector) FetchAndDrop(limit int) (metric.NamespaceMetrics[float64], error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.flushUnlocked() // finish any pending writes

	log := c.ctx.Log()

	nsMap, err := readAndDeleteRollingLogs(log, c.baseDir, c.fileNamePrefix, limit)

	if err != nil && err != EOF {
		return nil, err
	}

	result := metric.NamespaceMetrics[float64]{}

	for ns, lines := range nsMap {
		metrics := unmarshalList[metric.Metric[float64]](lines, log)

		result[ns] = metrics
	}

	return result, err
}

func (c *namespacedDiskMetricCollector) Flush() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.flushUnlocked()
}

func (c *namespacedDiskMetricCollector) flushUnlocked() error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(c.collectorMap))

	// flush collectors for each namespace in parallel
	for _, innerCollector := range c.collectorMap {
		wg.Add(1)
		go func(collector *rollingDiskMetricCollector) {
			defer wg.Done()

			errCh <- collector.flush()
		}(innerCollector)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errCh)

	errs := make([]error, 0)
	for err := range errCh {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *namespacedDiskMetricCollector) Clean() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var eg errgroup.Group
	eg.SetLimit(4) // limit to 4 parallel flushes

	// close collectors for each namespace in parallel
	for _, innerCollector := range c.collectorMap {
		func(collector *rollingDiskMetricCollector) {
			eg.Go(func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						c.ctx.Log().Warnf("namespacedDiskMetricCollector Clean panic: %v", r)
						c.ctx.Log().Warnf("Stacktrace:\n%s", debug.Stack())
						err = fmt.Errorf("panic in namespacedDiskMetricCollector Clean %v", r)
					}
				}()

				return collector.flush()
			})
		}(innerCollector)
	}

	// Wait for all goroutines to finish
	err := eg.Wait()
	if err != nil {
		return err
	}

	// clean the directory
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(c.baseDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func (c *namespacedDiskMetricCollector) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var eg errgroup.Group
	eg.SetLimit(4) // limit to 4 parallel closes

	// close collectors for each namespace in parallel
	for _, innerCollector := range c.collectorMap {
		func(collector *rollingDiskMetricCollector) {
			eg.Go(func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						c.ctx.Log().Warnf("namespacedDiskMetricCollector Close panic: %v", r)
						c.ctx.Log().Warnf("Stacktrace:\n%s", debug.Stack())
						err = fmt.Errorf("panic in namespacedDiskMetricCollector Close %v", r)
					}
				}()

				return collector.close()
			})
		}(innerCollector)
	}

	// Wait for all goroutines to finish
	err := eg.Wait()
	if err != nil {
		return err
	}

	for k := range c.collectorMap {
		delete(c.collectorMap, k)
	}
	return nil
}

// getMetricCollector returns a [rollingDiskMetricCollector] for the given namespace
func (c *namespacedDiskMetricCollector) getMetricCollector(namespace string) (*rollingDiskMetricCollector, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	if c.collectorMap[namespace] == nil {
		p := filepath.Join(c.baseDir, namespace)
		if err := fileutil.MakeDirs(p); err != nil {
			return nil, err
		}

		loggerConfig := getLoggerConfig(p, c.fileNamePrefix, c.maxRolls, c.maxFileSize)
		seelogger, err := seelog.LoggerFromConfigAsBytes(loggerConfig)
		if err != nil {
			return nil, err
		}

		rw := &rollingDiskMetricCollector{
			dirPath: p,
			logger:  seelogger,
		}

		c.collectorMap[namespace] = rw
	}

	return c.collectorMap[namespace], nil
}

func (c *rollingDiskMetricCollector) write(bytes []byte) (err error) {
	if c.logger == nil {
		return errors.New("logger is not intialized")
	}

	c.logger.Trace(string(bytes))
	return nil
}

func (rw *rollingDiskMetricCollector) flush() error {
	rw.logger.Flush()
	return nil
}

func (rw *rollingDiskMetricCollector) close() error {
	rw.logger.Close()

	// remove the namespace direcory if it is empty. ignore errors
	deleteDirectoryIfAllFilesEmpty(rw.dirPath)

	return nil
}
