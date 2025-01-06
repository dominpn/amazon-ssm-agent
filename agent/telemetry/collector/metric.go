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
	"sync"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
)

type timeSpan struct {
	startTime time.Time
	endTime   time.Time
}

// timeAggregatedMetric holds metrics aggregated by time spans
type timeAggregatedMetric[N int64 | float64] struct {
	Name string
	Unit string

	mtx   *sync.Mutex
	spans map[timeSpan]*metric.DataPoint[N]
}

type metricCollector struct {
	mtx            *sync.Mutex
	int64metrics   map[string]*map[string]*timeAggregatedMetric[int64]
	float64metrics map[string]*map[string]*timeAggregatedMetric[int64]
}

func newMetricCollector(context context.T) *metricCollector {
	return &metricCollector{
		mtx: &sync.Mutex{},
	}
}

func (c *metricCollector) Collect(namespace string, metric metric.Metric[float64]) error {
	//TODO implement me
	panic("implement me")
}

func (c *metricCollector) Fetch(namespace string, limit int) ([]metric.Metric[float64], error) {
	//TODO implement me
	panic("implement me")
}

func (c *metricCollector) Close() error {
	//TODO implement me
	panic("implement me")
}
