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
	"testing"
	"time"

	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/stretchr/testify/assert"
)

func TestNewTimeAggregatedMetric(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		unit       metric.Unit
		kind       metric.Kind
		wantErr    bool
	}{
		{
			name:       "valid sum metric",
			metricName: "test_metric",
			unit:       metric.UnitCount,
			kind:       metric.Sum,
			wantErr:    false,
		},
		{
			name:       "unsupported metric kind",
			metricName: "test_metric",
			unit:       metric.UnitCount,
			kind:       metric.Kind("unsupported"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := newTimeAggregatedMetric[int64](tt.metricName, tt.unit, tt.kind)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, metric)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metric)
				assert.Equal(t, tt.metricName, metric.name)
				assert.Equal(t, tt.unit, metric.unit)
				assert.Equal(t, tt.kind, metric.kind)
			}
		})
	}
}

func TestTimeAggregatedMetricAggregateStartTimeEqualsEndTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m, err := newTimeAggregatedMetric[int64]("test_metric", "count", metric.Sum)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		point   metric.DataPoint[int64]
		wantErr bool
	}{
		{
			name: "valid single point",
			point: metric.DataPoint[int64]{
				StartTime: now,
				EndTime:   now,
				Value:     10,
			},
			wantErr: false,
		},
		{
			name: "invalid data point",
			point: metric.DataPoint[int64]{
				StartTime: now,
				EndTime:   now.Add(time.Second),
				Value:     10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.aggregate(tt.point)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify the point was aggregated
				found := false
				for span, point := range m.spans {
					if span.contains(tt.point.StartTime) {
						found = true
						assert.Equal(t, float64(tt.point.Value), point.Value)
					}
				}
				assert.True(t, found)
			}
		})
	}
}

func TestInt64TimeAggregatedSumMetric(t *testing.T) {
	tam, err := newTimeAggregatedMetric[int64]("test_metric", metric.UnitCount, metric.Sum)
	assert.NoError(t, err)
	assert.Equal(t, "test_metric", tam.name)
	assert.Equal(t, metric.UnitCount, tam.unit)
	assert.Equal(t, metric.Sum, tam.kind)

	now := time.Now().Truncate(time.Second)

	metrics := []metric.DataPoint[int64]{
		{
			StartTime: now,
			EndTime:   now,
			Value:     10,
		},
		{
			StartTime: now.Add(time.Second),
			EndTime:   now.Add(time.Second),
			Value:     10,
		},
		{
			StartTime: now.Add(time.Second).Add(500 * time.Millisecond),
			EndTime:   now.Add(time.Second).Add(500 * time.Millisecond),
			Value:     10,
		},
		{
			StartTime: now.Add(500 * time.Minute),
			EndTime:   now.Add(500 * time.Minute),
			Value:     10,
		},
		{
			StartTime: now.Add(500 * time.Minute).Add(999 * time.Millisecond),
			EndTime:   now.Add(500 * time.Minute).Add(999 * time.Millisecond),
			Value:     5,
		},
	}

	for _, datapoint := range metrics {
		err := tam.aggregate(datapoint)
		assert.NoError(t, err)
	}

	assert.Len(t, tam.spans, 3)

	first := tam.spans[timeSpan{
		startTime: now,
		endTime:   now.Add(time.Second),
	}]
	assert.Equal(t, float64(10), first.Value)
	assert.Equal(t, 1, first.dataPointCounts)

	second := tam.spans[timeSpan{
		startTime: now.Add(time.Second),
		endTime:   now.Add(2 * time.Second),
	}]
	assert.Equal(t, float64(20), second.Value)
	assert.Equal(t, 2, second.dataPointCounts)

	third := tam.spans[timeSpan{
		startTime: now.Add(500 * time.Minute),
		endTime:   now.Add(500 * time.Minute).Add(time.Second),
	}]
	assert.Equal(t, float64(15), third.Value)
	assert.Equal(t, 2, third.dataPointCounts)
}

func TestFloat64TimeAggregatedSumMetric(t *testing.T) {
	tam, err := newTimeAggregatedMetric[float64]("test_metric", metric.UnitCount, metric.Sum)
	assert.NoError(t, err)
	assert.Equal(t, "test_metric", tam.name)
	assert.Equal(t, metric.UnitCount, tam.unit)
	assert.Equal(t, metric.Sum, tam.kind)

	now := time.Now().Truncate(time.Second)

	metrics := []metric.DataPoint[float64]{
		{
			StartTime: now,
			EndTime:   now,
			Value:     10,
		},
		{
			StartTime: now.Add(time.Second),
			EndTime:   now.Add(time.Second),
			Value:     10,
		},
		{
			StartTime: now.Add(time.Second).Add(500 * time.Millisecond),
			EndTime:   now.Add(time.Second).Add(500 * time.Millisecond),
			Value:     10,
		},
		{
			StartTime: now.Add(500 * time.Minute),
			EndTime:   now.Add(500 * time.Minute),
			Value:     10,
		},
		{
			StartTime: now.Add(500 * time.Minute).Add(999 * time.Millisecond),
			EndTime:   now.Add(500 * time.Minute).Add(999 * time.Millisecond),
			Value:     5,
		},
	}

	for _, datapoint := range metrics {
		err := tam.aggregate(datapoint)
		assert.NoError(t, err)
	}

	assert.Len(t, tam.spans, 3)

	first := tam.spans[timeSpan{
		startTime: now,
		endTime:   now.Add(time.Second),
	}]
	assert.Equal(t, float64(10), first.Value)
	assert.Equal(t, 1, first.dataPointCounts)

	second := tam.spans[timeSpan{
		startTime: now.Add(time.Second),
		endTime:   now.Add(2 * time.Second),
	}]
	assert.Equal(t, float64(20), second.Value)
	assert.Equal(t, 2, second.dataPointCounts)

	third := tam.spans[timeSpan{
		startTime: now.Add(500 * time.Minute),
		endTime:   now.Add(500 * time.Minute).Add(time.Second),
	}]
	assert.Equal(t, float64(15), third.Value)
	assert.Equal(t, 2, third.dataPointCounts)
}

func TestInt64TimeAggregatedGaugeMetric(t *testing.T) {
	tam, err := newTimeAggregatedMetric[int64]("test_metric", metric.UnitCount, metric.Gauge)
	assert.NoError(t, err)
	assert.Equal(t, "test_metric", tam.name)
	assert.Equal(t, metric.UnitCount, tam.unit)
	assert.Equal(t, metric.Gauge, tam.kind)

	now := time.Now().Truncate(time.Second)

	metrics := []metric.DataPoint[int64]{
		{
			StartTime: now,
			EndTime:   now,
			Value:     10,
		},
		{
			StartTime: now.Add(time.Second),
			EndTime:   now.Add(time.Second),
			Value:     10,
		},
		{
			StartTime: now.Add(time.Second).Add(500 * time.Millisecond),
			EndTime:   now.Add(time.Second).Add(500 * time.Millisecond),
			Value:     10,
		},
		{
			StartTime: now.Add(500 * time.Minute),
			EndTime:   now.Add(500 * time.Minute),
			Value:     10,
		},
		{
			StartTime: now.Add(500 * time.Minute).Add(999 * time.Millisecond),
			EndTime:   now.Add(500 * time.Minute).Add(999 * time.Millisecond),
			Value:     5,
		},
	}

	for _, datapoint := range metrics {
		err := tam.aggregate(datapoint)
		assert.NoError(t, err)
	}

	assert.Len(t, tam.spans, 3)

	first := tam.spans[timeSpan{
		startTime: now,
		endTime:   now.Add(time.Second),
	}]
	assert.Equal(t, float64(10), first.Value)
	assert.Equal(t, 1, first.dataPointCounts)

	second := tam.spans[timeSpan{
		startTime: now.Add(time.Second),
		endTime:   now.Add(2 * time.Second),
	}]
	assert.Equal(t, float64(10), second.Value)
	assert.Equal(t, 2, second.dataPointCounts)

	third := tam.spans[timeSpan{
		startTime: now.Add(500 * time.Minute),
		endTime:   now.Add(500 * time.Minute).Add(time.Second),
	}]
	assert.Equal(t, float64(7.5), third.Value)
	assert.Equal(t, 2, third.dataPointCounts)
}
