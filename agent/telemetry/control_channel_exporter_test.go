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
package telemetry

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/mocks/context"
	commMock "github.com/aws/amazon-ssm-agent/agent/session/communicator/mocks"
	mgsContracts "github.com/aws/amazon-ssm-agent/agent/session/contracts"
	"github.com/aws/amazon-ssm-agent/agent/telemetry/collector"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type controlChannelExporterTestSuite struct {
	suite.Suite

	ctx           *context.Mock
	mockWsChannel commMock.IWebSocketChannel
	exporter      *controlChannelTelemetryExporter
}

// TestControlChannelExporterSuite executes test suite
func TestControlChannelExporterSuite(t *testing.T) {
	suite.Run(t, new(controlChannelExporterTestSuite))
}

// SetupTest makes sure that all the components referenced in the
// test case are initialized before each test
func (suite *controlChannelExporterTestSuite) SetupTest() {
	suite.ctx = context.NewMockDefault()
	suite.mockWsChannel = commMock.IWebSocketChannel{}

	suite.exporter = GetControlChannelTelemetryExporter(suite.ctx, &suite.mockWsChannel)
}

func (suite *controlChannelExporterTestSuite) TearDownTest() {
	suite.exporter.StopExporter()
}

func (suite *controlChannelExporterTestSuite) TestStartExporter() {
	err := collector.AddExporter(suite.exporter)

	assert.ErrorContains(suite.T(), err, "cannot add exporter. telemetry collector not initialized")
}

func (suite *controlChannelExporterTestSuite) TestRemoveExporter() {
	err := collector.RemoveExporter(suite.exporter)

	assert.ErrorContains(suite.T(), err, "cannot remove exporter. telemetry collector not initialized")
}

func (suite *controlChannelExporterTestSuite) TestExportEmptyTelemetry() {
	err := suite.exporter.Export("testNamespace", []metric.Metric[float64]{}, []telemetrylog.Entry{})
	assert.NoError(suite.T(), err)

	suite.mockWsChannel.AssertNotCalled(suite.T(), "SendMessage")
}

func (suite *controlChannelExporterTestSuite) TestExportTelemetrySuccess() {

	now := time.Now().UTC()

	// prepare test telemetry
	namespace := "testNamespace"

	sentMetrics := make([]metric.Metric[float64], 0)
	sentLogs := make([]telemetrylog.Entry, 0)

	expectedMetrics := make([]Metric, 0)
	expectedLogs := make([]LogEntry, 0)

	for j := range 10 {
		metricName := fmt.Sprintf("testMetric%v", j)
		sentMetrics = append(sentMetrics, metric.Metric[float64]{
			Name:       metricName,
			Unit:       "1",
			Kind:       metric.Sum,
			DataPoints: []metric.DataPoint[float64]{{StartTime: now, EndTime: now.Add(time.Second), Value: 100}},
		})
		expectedMetrics = append(expectedMetrics, Metric{
			Name:       metricName,
			Unit:       "1",
			DataPoints: []DataPoint{{Time: now, Value: 100}},
		})

		sentLogs = append(sentLogs, telemetrylog.Entry{
			Time:     now,
			Severity: telemetrylog.ERROR,
			Body:     fmt.Sprintf("This is a test message %v", j),
		})
		expectedLogs = append(expectedLogs, LogEntry{
			Time:     now,
			Severity: telemetrylog.ERROR,
			Body:     fmt.Sprintf("This is a test message %v", j),
		})
	}

	// set expectations
	receivedMessage := &mgsContracts.AgentMessage{}

	suite.mockWsChannel.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		// verify telemetry is sent
		message := args.Get(1).([]byte)
		assert.NotEmpty(suite.T(), message)

		err := receivedMessage.Deserialize(suite.ctx.Log(), message)
		require.NoError(suite.T(), err)
	}).Return(nil)

	err := suite.exporter.Export(namespace, sentMetrics, sentLogs)
	assert.NoError(suite.T(), err)

	suite.mockWsChannel.AssertNumberOfCalls(suite.T(), "SendMessage", 1)

	payload := receivedMessage.Payload

	agentTelemetryV2 := AgentTelemetryV2{}
	err = json.Unmarshal(payload, &agentTelemetryV2)
	assert.NoError(suite.T(), err)

	receivedInnerPayload := AgentTelemetryV2Payload{}
	err = json.Unmarshal([]byte(agentTelemetryV2.Payload), &receivedInnerPayload)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), uint32(1), agentTelemetryV2.SchemaVersion, "Schema version changed! Is this expected? If yes, only then update this test")

	assert.Equal(suite.T(), expectedMetrics, receivedInnerPayload.Metrics)
	assert.Equal(suite.T(), expectedLogs, receivedInnerPayload.Logs)
}
