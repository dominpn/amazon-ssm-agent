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
	"path/filepath"
	"strconv"
	"sync"

	"github.com/cihub/seelog"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/fileutil"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
)

// rollingLogCollector holds rolling log state of a single namespace
type rollingLogCollector struct {
	// directory path
	dirPath string

	// seelog instnace
	logger seelog.LoggerInterface
}

// namespacedRollingLogCollector holds a rollingLogCollector for each namespace
type namespacedRollingLogCollector struct {
	ctx context.T

	baseDir string
	// maximum number of rolled files
	maxRolls int
	// maximum size of one file
	maxFileSize int64

	// prefix of each log file
	fileNamePrefix string
	mtx            *sync.Mutex
	collectorMap   map[string]*rollingLogCollector
}

// for mocking the write directory since TelemetryDataStorePath is a constant
var getBaseLogStoreDir = func() string {
	return filepath.Join(appconfig.TelemetryDataStorePath, "logs")
}

func newRollingLogCollector(context context.T, maxRolls int, maxFileSize int64, fileNamePrefix string) *namespacedRollingLogCollector {
	return &namespacedRollingLogCollector{
		ctx:            context,
		baseDir:        getBaseLogStoreDir(),
		maxRolls:       maxRolls,
		maxFileSize:    maxFileSize,
		fileNamePrefix: fileNamePrefix,
		mtx:            &sync.Mutex{},
		collectorMap:   make(map[string]*rollingLogCollector),
	}
}

func (c *namespacedRollingLogCollector) CollectLog(namespace string, entry telemetrylog.Entry) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	entryBytes, err := json.Marshal(entry)

	if err != nil {
		return err
	}

	rw, err := c.getLogCollector(namespace)

	if err != nil {
		return err
	}

	err = rw.write(entryBytes)

	return err
}

func (c *namespacedRollingLogCollector) FetchAndDrop(limit int) (telemetrylog.NamespaceLogs, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.flushUnlocked() // finish any pending writes

	log := c.ctx.Log()

	nsMap, err := readAndDeleteRollingLogs(log, c.baseDir, c.fileNamePrefix, limit)

	if err != nil && err != EOF {
		return nil, err
	}

	result := telemetrylog.NamespaceLogs{}

	for ns, lines := range nsMap {
		logs := unmarshalList[telemetrylog.Entry](lines, log)

		result[ns] = logs
	}

	return result, err
}

func (c *namespacedRollingLogCollector) Flush() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.flushUnlocked()
}

func (c *namespacedRollingLogCollector) flushUnlocked() error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(c.collectorMap))

	// flush collectors for each namespace in parallel
	for _, innerCollector := range c.collectorMap {
		wg.Add(1)
		go func(collector *rollingLogCollector) {
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

func (c *namespacedRollingLogCollector) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(c.collectorMap))

	// close collectors for each namespace in parallel
	for _, innerCollector := range c.collectorMap {
		wg.Add(1)
		go func(collector *rollingLogCollector) {
			defer wg.Done()

			errCh <- collector.close()
		}(innerCollector)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errCh)

	for k := range c.collectorMap {
		delete(c.collectorMap, k)
	}

	errs := make([]error, 0)
	for err := range errCh {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *namespacedRollingLogCollector) getLogCollector(namespace string) (*rollingLogCollector, error) {
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

		rw := &rollingLogCollector{
			dirPath: p,
			logger:  seelogger,
		}

		c.collectorMap[namespace] = rw
	}

	return c.collectorMap[namespace], nil
}

func getLoggerConfig(defaultLogDir string, logFile string, maxRolls int, maxFileSize int64) []byte {

	logFilePath := filepath.Join(defaultLogDir, logFile)
	logConfig := `
<seelog type="adaptive" mininterval="2000000" maxinterval="100000000" critmsgcount="500" minlevel="trace">
    <outputs formatid="common"> `
	logConfig += `<rollingfile type="size" filename="` + logFilePath + `" maxsize="` + strconv.FormatInt(maxFileSize, 10) + `" maxrolls="` + strconv.Itoa(maxRolls) + `"/>
    </outputs>
    <formats>
        <format id="common" format="%Msg%n"/>
    </formats>
</seelog>
`
	return []byte(logConfig)
}

func (c *rollingLogCollector) write(bytes []byte) (err error) {
	if c.logger == nil {
		return errors.New("logger is not intialized")
	}

	c.logger.Trace(string(bytes))
	return nil
}

func (rw *rollingLogCollector) flush() error {
	rw.logger.Flush()
	return nil
}

func (rw *rollingLogCollector) close() error {
	rw.logger.Close()

	// remove the namespace direcory if it is empty. ignore errors
	deleteDirectoryIfAllFilesEmpty(rw.dirPath)

	return nil
}
