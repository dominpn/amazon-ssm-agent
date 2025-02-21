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
	"strings"
	"testing"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/mocks/context"
	"github.com/aws/amazon-ssm-agent/agent/mocks/log"

	"github.com/aws/amazon-ssm-agent/agent/telemetry/collector/mocks"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HydringMetricsCollectorTestSuite struct {
	suite.Suite
	ctx       *context.Mock
	collector *hybridMetricCollector
}

func TestHydringMetricsCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(HydringMetricsCollectorTestSuite))
}

// SetupTest makes sure that all the components referenced in the test case are initialized
// before each test
func (suite *HydringMetricsCollectorTestSuite) SetupTest() {
	suite.ctx = context.NewMockDefault()

	collector, err := NewHybridMetricCollector(suite.ctx, 10, 1024*1024, "metrics", 10)
	assert.NoError(suite.T(), err)

	suite.collector = collector
}

func (suite *HydringMetricsCollectorTestSuite) TestHybridMetricCollectorWriteToDisk() {
	suite.collector.diskWriteSchedulerJob.Quit <- true

	inMemoryCollectorMock := mocks.NewFastMetricsCollectorMock()
	diskCollectorMock := mocks.NewSlowMetricsCollectorMock()

	suite.collector.inMemoryCollector = inMemoryCollectorMock
	suite.collector.onDiskCollector = diskCollectorMock

	resultList := metric.NamespaceMetrics[float64]{
		"namespace1": []metric.Metric[float64]{{
			Name: "metric1",
			Unit: "1",
			Kind: metric.Sum,
			DataPoints: []metric.DataPoint[float64]{
				{
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Value:     1.0,
				},
			},
		},
			{
				Name: "metric2",
				Unit: "1",
				Kind: metric.Sum,
				DataPoints: []metric.DataPoint[float64]{
					{
						StartTime: time.Now(),
						EndTime:   time.Now(),
						Value:     1.0,
					},
				},
			},
		},
		"namespace2": []metric.Metric[float64]{{
			Name: "metric2",
			Unit: "1",
			Kind: metric.Sum,
			DataPoints: []metric.DataPoint[float64]{
				{
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Value:     1.0,
				},
				{
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Value:     3.0,
				},
			},
		},
			{
				Name: "metric1",
				Unit: "1",
				Kind: metric.Sum,
				DataPoints: []metric.DataPoint[float64]{
					{
						StartTime: time.Now(),
						EndTime:   time.Now(),
						Value:     1.0,
					},
				},
			}},
	}

	inMemoryCollectorMock.On("FetchAllAndDrop").Return(func() (metric.NamespaceMetrics[float64], error) {
		return resultList, nil
	})

	diskCollectorMock.On("Collect", mock.Anything, mock.Anything).Return(nil)

	inMemoryCollectorMock.On("Clean").Return(nil)

	suite.collector.writeToDisk()

	inMemoryCollectorMock.AssertNumberOfCalls(suite.T(), "FetchAllAndDrop", 1)

	diskCollectorMock.AssertNumberOfCalls(suite.T(), "Collect", 4)
}

func (suite *HydringMetricsCollectorTestSuite) TestHybridMetricCollectorWritesPeriodically() {
	inMemoryCollectorMock := mocks.NewFastMetricsCollectorMock()
	diskCollectorMock := mocks.NewSlowMetricsCollectorMock()

	suite.collector.inMemoryCollector = inMemoryCollectorMock
	suite.collector.onDiskCollector = diskCollectorMock

	resultList := metric.NamespaceMetrics[float64]{
		"namespace1": []metric.Metric[float64]{{
			Name: "metric1",
			Unit: "1",
			Kind: metric.Sum,
			DataPoints: []metric.DataPoint[float64]{
				{
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Value:     1.0,
				},
				{
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Value:     3.0,
				},
			},
		}},
	}

	inMemoryCollectorMock.On("FetchAllAndDrop").Return(func() (metric.NamespaceMetrics[float64], error) {
		return resultList, nil
	})

	diskCollectorMock.On("Collect", mock.Anything, mock.Anything).Return(nil)

	// skip the scheduler wait
	suite.collector.diskWriteSchedulerJob.SkipWait <- true

	// wait for the scheduler to complete
	select {
	case <-suite.collector.diskWriteSchedulerRunCompletedChan:
	case <-time.After(5 * time.Second):
		suite.T().Fatal("Disk write scheduler did not complete in time")
	}

	inMemoryCollectorMock.AssertNumberOfCalls(suite.T(), "FetchAllAndDrop", 1)
	diskCollectorMock.AssertNumberOfCalls(suite.T(), "Collect", 1)
}

func (suite *HydringMetricsCollectorTestSuite) TestHybridMetricCollectorPanicRecovered() {
	inMemoryCollectorMock := mocks.NewFastMetricsCollectorMock()
	diskCollectorMock := mocks.NewSlowMetricsCollectorMock()

	suite.collector.inMemoryCollector = inMemoryCollectorMock
	suite.collector.onDiskCollector = diskCollectorMock

	inMemoryCollectorMock.On("FetchAllAndDrop").Panic("error fetching metrics")

	// skip the scheduler wait
	suite.collector.diskWriteSchedulerJob.SkipWait <- true

	// wait for the scheduler to complete
	select {
	case <-suite.collector.diskWriteSchedulerRunCompletedChan:
	case <-time.After(5 * time.Second):
		suite.T().Fatal("Disk write scheduler did not complete in time")
	}

	logMock := suite.ctx.Log().(*log.Mock)
	logMock.AssertCalled(suite.T(), "Errorf", mock.MatchedBy(func(msg string) bool {
		return strings.Contains(msg, "Metric disk write panic")
	}), mock.Anything)

	inMemoryCollectorMock.AssertNumberOfCalls(suite.T(), "FetchAllAndDrop", 1)
	inMemoryCollectorMock.AssertNumberOfCalls(suite.T(), "Clean", 0)

	diskCollectorMock.AssertNumberOfCalls(suite.T(), "Collect", 0)
}

func (suite *HydringMetricsCollectorTestSuite) TestHybridMetricCollectorFlush() {
	// create a hybrid metric collector
	collector, err := NewHybridMetricCollector(context.NewMockDefault(), 10, 1024*1024, "metrics", 10)
	assert.NoError(suite.T(), err)

	// mock the in memory collector
	inMemoryCollectorMock := mocks.NewFastMetricsCollectorMock()
	collector.inMemoryCollector = inMemoryCollectorMock

	// mock the on disk collector
	diskCollectorMock := mocks.NewSlowMetricsCollectorMock()
	collector.onDiskCollector = diskCollectorMock

	resultErr := errors.New("test error")
	diskCollectorMock.On("Flush").Return(resultErr)

	inMemoryCollectorMock.On("FetchAllAndDrop").Return(func() (metric.NamespaceMetrics[float64], error) {
		return metric.NamespaceMetrics[float64]{}, nil
	}).Once()

	// call the flush method
	err = collector.Flush()
	assert.Equal(suite.T(), resultErr, err)

	// Assert writeToDisk was called
	inMemoryCollectorMock.AssertNumberOfCalls(suite.T(), "FetchAllAndDrop", 1)
	diskCollectorMock.AssertNumberOfCalls(suite.T(), "Collect", 0)

	// assert the flush method was called
	diskCollectorMock.AssertCalled(suite.T(), "Flush")
}

func (suite *HydringMetricsCollectorTestSuite) TestHybridMetricCollectorClean() {
	// create a hybrid metric collector
	collector, err := NewHybridMetricCollector(context.NewMockDefault(), 10, 1024*1024, "metrics", 10)
	assert.NoError(suite.T(), err)

	// mock the in memory collector
	inMemoryCollectorMock := mocks.NewFastMetricsCollectorMock()
	collector.inMemoryCollector = inMemoryCollectorMock

	// mock the on disk collector
	diskCollectorMock := mocks.NewSlowMetricsCollectorMock()
	collector.onDiskCollector = diskCollectorMock

	resultErr1 := errors.New("test error 1")
	inMemoryCollectorMock.On("Clean").Return(resultErr1)

	resultErr2 := errors.New("test error 2")
	diskCollectorMock.On("Clean").Return(resultErr2)

	// call the Clean method
	err = collector.Clean()
	assert.Equal(suite.T(), errors.Join(resultErr1, resultErr2), err)
}

func (suite *HydringMetricsCollectorTestSuite) TestHybridMetricCollectorClose() {
	// mock the in memory collector
	inMemoryCollectorMock := mocks.NewFastMetricsCollectorMock()
	suite.collector.inMemoryCollector = inMemoryCollectorMock

	// mock the on disk collector
	diskCollectorMock := mocks.NewSlowMetricsCollectorMock()
	suite.collector.onDiskCollector = diskCollectorMock

	resultErr1 := errors.New("test error 1")
	inMemoryCollectorMock.On("Close").Return(resultErr1)

	resultErr2 := errors.New("test error 2")
	diskCollectorMock.On("Close").Return(resultErr2)

	// call the Clean method
	err := suite.collector.Close()
	assert.Equal(suite.T(), errors.Join(resultErr1, resultErr2), err)
}
