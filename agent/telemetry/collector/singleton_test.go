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
	"fmt"
	"strings"
	"testing"
	"time"

	logger "github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/mocks/context"
	logMock "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	collectorMocks "github.com/aws/amazon-ssm-agent/agent/telemetry/collector/mocks"
	"github.com/aws/amazon-ssm-agent/common/filewatcherbasedipc"
	channelMock "github.com/aws/amazon-ssm-agent/common/filewatcherbasedipc/mocks"
	"github.com/aws/amazon-ssm-agent/common/identity"
	"github.com/aws/amazon-ssm-agent/common/telemetry"
	telemetryContext "github.com/aws/amazon-ssm-agent/common/telemetry/context"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type singletonTestSuite struct {
	suite.Suite

	ctx *context.Mock
}

// TestRollingUtilsSuite executes test suite
func TestSingletonSuite(t *testing.T) {
	suite.Run(t, new(singletonTestSuite))
}

// SetupTest makes sure that all the components referenced in the
// test case are initialized before each test
func (suite *singletonTestSuite) SetupTest() {
	suite.ctx = context.NewMockDefault()

	channelCreator = func(log logger.T, _ identity.IAgentIdentity, filename string) (filewatcherbasedipc.IPCChannel, error, bool) {
		isFound := channelMock.IsExists(filename)
		fakeChannel := channelMock.NewFakeChannel(log, filewatcherbasedipc.ModeSurveyor, filename)
		return fakeChannel, nil, isFound
	}
}

func (suite *singletonTestSuite) TearDownTest() {
	Shutdown()
}

func (suite *singletonTestSuite) TestInitialize() {
	assert.Nil(suite.T(), singleton)

	Initialize(suite.ctx)

	assert.NotNil(suite.T(), singleton)
	assert.Equal(suite.T(), map[string]filewatcherbasedipc.IPCChannel{}, listenChannels)
	assert.Equal(suite.T(), map[string](chan bool){}, stopSignals)
	assert.NotNil(suite.T(), listenWg)
}

func (suite *singletonTestSuite) TestShutdown() {
	Initialize(suite.ctx)

	Shutdown()

	assert.Nil(suite.T(), singleton)
	assert.Nil(suite.T(), listenChannels)
	assert.Nil(suite.T(), stopSignals)
	assert.Nil(suite.T(), listenWg)
}

func (suite *singletonTestSuite) TestStartCollection() {
	Initialize(suite.ctx)

	// replace the singleton with mock
	collectorMock := collectorMocks.NewCollectorMock()
	singleton = collectorMock

	// create sender side of the IPC channel
	telemetryContext := telemetryContext.NewMockDefault()
	senderIpc := channelMock.NewFakeChannel(suite.ctx.Log(), filewatcherbasedipc.ModeRespondent, telemetryContext.ChannelName())
	defer senderIpc.Destroy()

	// set expectations
	collectorMock.On("CollectLog", mock.Anything, mock.Anything).Return(nil)
	collectorMock.On("Collect", mock.Anything, mock.Anything).Return(nil)
	collectorMock.On("Close").Return(nil).Once()

	// start telemetry collection
	err := StartCollection(telemetryContext)
	assert.NoError(suite.T(), err)

	logCounts := 10
	metricCounts := 8

	expectedLogEntries := make([]telemetrylog.Entry, 0, logCounts)
	expectedMetrics := make([]metric.Metric[float64], 0, metricCounts)

	now := time.Now()

	// send logs to the channel
	for i := range logCounts {
		expectedLogEntry := &telemetrylog.Entry{
			Time:     now.UTC(),
			Severity: telemetrylog.ERROR,
			Body:     fmt.Sprintf("This is a test message : %v", i),
		}
		entryJson, err := json.Marshal(expectedLogEntry)
		assert.NoError(suite.T(), err)
		expectedMessage := &telemetry.Message{
			Namespace: "testNamespace",
			Type:      telemetry.LOG,
			Payload:   string(entryJson),
		}

		datagram, err := json.Marshal(expectedMessage)
		assert.NoError(suite.T(), err)

		expectedLogEntries = append(expectedLogEntries, *expectedLogEntry)

		// send telemetry
		err = senderIpc.Send(string(datagram))
		assert.NoError(suite.T(), err)
	}

	// send metrics to the channel
	for i := range metricCounts {
		expectedMetric := &metric.Metric[float64]{
			Name: fmt.Sprintf("testMetric%v", i),
			Unit: "1",
			Kind: metric.Sum,
			DataPoints: []metric.DataPoint[float64]{
				{
					StartTime: now.UTC(),
					EndTime:   now.UTC(),
					Value:     float64(i),
				},
			},
		}
		entryJson, err := json.Marshal(expectedMetric)
		assert.NoError(suite.T(), err)
		expectedMessage := &telemetry.Message{
			Namespace: "testNamespace",
			Type:      telemetry.METRIC,
			Payload:   string(entryJson),
		}

		datagram, err := json.Marshal(expectedMessage)
		assert.NoError(suite.T(), err)

		expectedMetrics = append(expectedMetrics, *expectedMetric)

		// send telemetry
		err = senderIpc.Send(string(datagram))
		assert.NoError(suite.T(), err)
	}

	// assert that they were collected
	assert.EventuallyWithT(suite.T(), func(c *assert.CollectT) {
		ct := NewCommonT(c)

		collectorMock.AssertNumberOfCalls(ct, "CollectLog", logCounts)
		collectorMock.AssertNumberOfCalls(ct, "Collect", metricCounts)

		for _, e := range expectedLogEntries {
			collectorMock.AssertCalled(ct, "CollectLog", "testNamespace", e)
		}

		for _, e := range expectedMetrics {
			collectorMock.AssertCalled(ct, "Collect", "testNamespace", e)
		}
	}, 30*time.Second, 100*time.Millisecond)
}

func (suite *singletonTestSuite) TestStartCollectionMalformedMessage() {
	Initialize(suite.ctx)

	// replace the singleton with mock
	collectorMock := collectorMocks.NewCollectorMock()
	singleton = collectorMock

	// create sender side of the IPC channel
	telemetryContext := telemetryContext.NewMockDefault()
	mocklog := telemetryContext.Log().(*logMock.Mock)

	senderIpc := channelMock.NewFakeChannel(suite.ctx.Log(), filewatcherbasedipc.ModeRespondent, telemetryContext.ChannelName())
	defer senderIpc.Destroy()

	// set expectations
	collectorMock.On("CollectLog", mock.Anything, mock.Anything).Return(nil)
	collectorMock.On("Collect", mock.Anything, mock.Anything).Return(nil)
	collectorMock.On("Close").Return(nil).Once()

	// start telemetry collection
	err := StartCollection(telemetryContext)
	assert.NoError(suite.T(), err)

	logCounts := 10

	// send malformed messages to the channel
	for range logCounts {
		err = senderIpc.Send(string("malformed messaged"))
		assert.NoError(suite.T(), err)
	}

	assert.EventuallyWithT(suite.T(), func(c *assert.CollectT) {
		ct := NewCommonT(c)

		collectorMock.AssertNumberOfCalls(ct, "CollectLog", 0)
		collectorMock.AssertNumberOfCalls(ct, "Collect", 0)

		logsCalls := mocklog.Calls
		erroredLogCounts := 0
		for _, call := range logsCalls {
			if call.Method == "Debugf" && strings.Contains(call.Arguments.Get(0).(string), "error processing telemetry message") {
				erroredLogCounts++
			}
		}

		assert.Equal(ct, erroredLogCounts, logCounts)
	}, 20*time.Second, 100*time.Millisecond)
}

// interface which allows us to use assert.CollectT as testing.T
// open issue in testify: https://github.com/stretchr/testify/issues/1414
type commonT struct {
	c *assert.CollectT
}

func (c *commonT) FailNow() {
	c.c.FailNow()
}

func (c *commonT) Errorf(format string, args ...interface{}) {
	c.c.Errorf(format, args...)
}

func (c *commonT) Logf(format string, args ...interface{}) {
	c.c.Errorf(format, args...)
}

func NewCommonT(c *assert.CollectT) *commonT {
	return &commonT{
		c: c,
	}
}
