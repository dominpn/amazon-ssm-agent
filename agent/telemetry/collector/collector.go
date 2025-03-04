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
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/telemetry/exporter"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
	"github.com/carlescere/scheduler"
)

// EOF is returned as error in Fetch when no new log entries are present
var EOF = errors.New("EOF")

type Flushable interface {
	Flush() error
}

type LogCollector interface {
	CollectLog(namespace string, log telemetrylog.Entry) error

	FetchAndDrop(limit int) (telemetrylog.NamespaceLogs, error)

	Close() error
}

type MetricsCollector interface {
	Collect(namespace string, metric metric.Metric[float64]) error

	FetchAndDrop(limit int) (metric.NamespaceMetrics[float64], error)

	Clean() error

	Close() error
}

type Collector interface {
	CollectLog(namespace string, log telemetrylog.Entry) error

	Collect(namespace string, metric metric.Metric[float64]) error

	AddExporter(exporter exporter.Exporter)

	RemoveExporter(exporter exporter.Exporter)

	Close() error
}

type collectorT struct {
	aggregationPeriod  time.Duration
	metricCollector    MetricsCollector
	logCollector       LogCollector
	exporterMtx        *sync.Mutex
	exporters          []exporter.Exporter
	exportSchedulerJob *scheduler.Job
}

func NewCollector(context context.T, aggregationPeriod time.Duration, exportPeriod time.Duration) (Collector, error) {
	// TODO : make the parameters configurable. Currently set to 10 max rolling files, 100 KB each
	logCollector := newRollingLogCollector(context, 10, 100*1024, "logs")

	// TODO make these configurable
	metricCollector, err := NewHybridMetricCollector(context, 10, 1024*1024, "metrics", int(aggregationPeriod.Seconds()))

	if err != nil {
		return nil, err
	}

	c := &collectorT{
		aggregationPeriod: aggregationPeriod,
		metricCollector:   metricCollector,
		logCollector:      logCollector,
		exporterMtx:       &sync.Mutex{},
	}

	exportPeriodSeconds := int(exportPeriod.Seconds())

	if exportPeriodSeconds <= 0 {
		return nil, fmt.Errorf("export period is too small")
	}

	log := context.Log()

	if c.exportSchedulerJob, err = scheduler.Every(exportPeriodSeconds).NotImmediately().Seconds().Run(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Telemetry export panic: %v", r)
				log.Errorf("Stacktrace:\n%s", debug.Stack())
			}
		}()

		err := c.export()

		if err != nil {
			log.Warnf("Error when exporting telemetry: %v", err)
		}
	}); err != nil {
		return nil, fmt.Errorf("unable to schedule telemetry exporter process: %v", err)
	}

	return c, nil
}

func (c *collectorT) Collect(namespace string, metric metric.Metric[float64]) error {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	return c.metricCollector.Collect(namespace, metric)
}

func (c *collectorT) CollectLog(namespace string, log telemetrylog.Entry) error {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	return c.logCollector.CollectLog(namespace, log)
}

// AddExporter adds a new Exporter to the collector with the specified export period
func (c *collectorT) AddExporter(exporter exporter.Exporter) {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	c.exporters = append(c.exporters, exporter)
}

// RemoveExporter removes an Exporter from the collector
func (c *collectorT) RemoveExporter(exporter exporter.Exporter) {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	for i, e := range c.exporters {
		if e == exporter {
			c.exporters = slices.Delete(c.exporters, i, i+1)
			break
		}
	}
}

// export exports all the telemetry the singleton holds (both in-memory and on disk) in reasonable chunks
func (c *collectorT) export() error {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	// don't want to lose telemetry until exporters are attached
	if len(c.exporters) == 0 {
		return nil
	}

	var errMetrics, errLogs error
	var exportErrs []error
	for errMetrics == nil || errLogs == nil {
		var metrics metric.NamespaceMetrics[float64]
		var logs telemetrylog.NamespaceLogs

		metrics, errMetrics = c.metricCollector.FetchAndDrop(1000) // TODO: make configurable
		logs, errLogs = c.logCollector.FetchAndDrop(1000)

		if errMetrics != nil && errMetrics != EOF {
			return errMetrics
		}

		if errLogs != nil && errLogs != EOF {
			return errLogs
		}

		if len(metrics) == 0 && len(logs) == 0 {
			return nil
		}

		// get all the unique namespaces in both metrics and logs
		namespaces := make(map[string]bool)
		for ns := range metrics {
			namespaces[ns] = true
		}
		for ns := range logs {
			namespaces[ns] = true
		}

		// send telemetry for all namespaces
		for ns := range namespaces {
			err := c.exportNamespaceTelemetry(ns, metrics[ns], logs[ns])

			exportErrs = append(exportErrs, err)
		}

		if errMetrics == EOF && errLogs == EOF {
			return nil
		}
	}

	return errors.Join(errors.Join(errMetrics, errLogs), errors.Join(exportErrs...))
}

// exportNamespaceTelemetry exportes telemetry for a specific namespace to the attached exporters
func (c *collectorT) exportNamespaceTelemetry(namespace string, metrics []metric.Metric[float64], logs []telemetrylog.Entry) error {
	if metrics == nil {
		metrics = []metric.Metric[float64]{}
	}
	if logs == nil {
		logs = []telemetrylog.Entry{}
	}

	if len(metrics) == 0 && len(logs) == 0 {
		return nil
	}

	var errs []error

	for _, exporter := range c.exporters {
		err := exporter.Export(namespace, metrics, logs)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (c *collectorT) Close() error {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	c.exportSchedulerJob.Quit <- true

	var errs []error

	errs = append(errs, c.metricCollector.Close())
	errs = append(errs, c.logCollector.Close())

	return errors.Join(errs...)
}
