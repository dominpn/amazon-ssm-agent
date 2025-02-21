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
	"sync"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/telemetry/exporter"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
)

// EOF is returned as error in Fetch when no new log entries are present
var EOF = errors.New("EOF")

type Flushable interface {
	Flush() error
}

// type Fetchable[N interface{}] interface {
// 	// Fetch fetches all the available records, runs the function on them, and drops them

// }

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

	AddExporter(exportPeriod time.Duration, exporter exporter.Exporter)

	Close() error
}

type exporterConfig struct {
	exportPeriod time.Duration
	exporter     exporter.Exporter
}

type collectorT struct {
	aggregationPeriod time.Duration
	metricCollector   MetricsCollector
	logCollector      LogCollector
	exporterMtx       *sync.Mutex
	exporters         []exporterConfig
}

func NewCollector(context context.T, aggregationPeriod time.Duration) (Collector, error) {
	// TODO : make the parameters configurable. Currently set to 10 max rolling files, 100 KB each
	logCollector := newRollingLogCollector(context, 10, 100*1024, "logs")

	// TODO make these configurable
	metricCollector, err := NewHybridMetricCollector(context, 10, 1024*1024, "metrics", 10)

	if err != nil {
		return nil, err
	}

	collector := &collectorT{
		aggregationPeriod: aggregationPeriod,
		metricCollector:   metricCollector,
		logCollector:      logCollector,
		exporterMtx:       &sync.Mutex{},
	}

	return collector, nil
}

func (c *collectorT) Collect(namespace string, metric metric.Metric[float64]) error {
	//TODO implement me
	panic("implement me")
}

func (c *collectorT) CollectLog(namespace string, log telemetrylog.Entry) error {
	//TODO implement me
	panic("implement me")
}

// AddExporter adds a new Exporter to the collector with the specified export period
func (c *collectorT) AddExporter(exportPeriod time.Duration, exporter exporter.Exporter) {
	c.exporterMtx.Lock()
	defer c.exporterMtx.Unlock()

	c.exporters = append(c.exporters, exporterConfig{
		exportPeriod: exportPeriod,
		exporter:     exporter,
	})
}

func (c *collectorT) Close() error {
	//TODO implement me
	panic("implement me")
}
