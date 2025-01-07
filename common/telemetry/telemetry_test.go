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

// Package agent represents the core SSM agent object
package telemetry

import (
	"encoding/json"
	"testing"
	"time"

	logger "github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/mocks/context"
	"github.com/aws/amazon-ssm-agent/common/filewatcherbasedipc"
	channelmock "github.com/aws/amazon-ssm-agent/common/filewatcherbasedipc/mocks"
	"github.com/aws/amazon-ssm-agent/common/identity"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TelemetryTestSuite define agent test suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type TelemetryTestSuite struct {
	suite.Suite
	mockContext *context.Mock
	mockIpc     *channelmock.MockedChannel
}

// TestChannelSuite executes test suite
func TestTelemetrySuite(t *testing.T) {
	suite.Run(t, new(TelemetryTestSuite))
}

// SetupTest makes sure that all the components referenced in the test case are initialized
// before each test
func (suite *TelemetryTestSuite) SetupTest() {

	suite.mockContext = context.NewMockDefault()
	suite.mockIpc = new(channelmock.MockedChannel)
	suite.mockIpc.On("Destroy").Return(nil)

	channelCreator = func(log logger.T, _ identity.IAgentIdentity, mode filewatcherbasedipc.Mode, filename string) (filewatcherbasedipc.IPCChannel, error, bool) {
		isFound := channelmock.IsExists(filename)
		fakeChannel := channelmock.NewFakeChannel(log, mode, filename)
		return fakeChannel, nil, isFound
	}
}

func (suite *TelemetryTestSuite) TearDownTest() {
	Shutdown()
}

func (suite *TelemetryTestSuite) TestInitialize() {

	telemetryInstance, _ := getTelemetry()
	assert.Nil(suite.T(), telemetryInstance)

	Initialize(suite.mockContext)

	telemetryInstance, _ = getTelemetry()
	assert.NotNil(suite.T(), telemetryInstance)
	assert.NotNil(suite.T(), telemetryInstance.fileChannel)

	fakeChannel := telemetryInstance.fileChannel.(*channelmock.FakeChannel)

	assert.Equal(suite.T(), "telemetry", fakeChannel.GetPath())
	assert.Equal(suite.T(), filewatcherbasedipc.ModeRespondent, fakeChannel.GetMode())
}

// TestTelemetryAlreadyInitialized verifies that the telemetry initialization
// can only occur once
func (suite *TelemetryTestSuite) TestTelemetryAlreadyInitialized() {

	telemetryInstance, _ := getTelemetry()
	assert.Nil(suite.T(), telemetryInstance)

	Initialize(suite.mockContext)

	telemetryInstance, _ = getTelemetry()

	assert.NotNil(suite.T(), telemetryInstance)
	err := Initialize(suite.mockContext)

	assert.Equal(suite.T(), "telemetry is already initialized", err.Error())
}

func (suite *TelemetryTestSuite) TestShutdown() {
	Initialize(suite.mockContext)

	telemetryInstance, _ := getTelemetry()
	assert.NotNil(suite.T(), telemetryInstance.fileChannel)

	telemetryInstance.fileChannel = suite.mockIpc

	Shutdown()
	suite.mockIpc.AssertCalled(suite.T(), "Destroy")

	telemetryInstance, _ = getTelemetry()
	assert.Nil(suite.T(), telemetryInstance)
}

func (suite *TelemetryTestSuite) Test_emitLog() {
	Initialize(suite.mockContext)

	telemetryInstance, _ := getTelemetry()

	now := time.Now()

	err := telemetryInstance.emitLog("testNamespace", now, telemetrylog.ERROR, "This is a test message")
	assert.Nil(suite.T(), err)

	expectedLogEntry := &telemetrylog.Entry{
		Time:     now,
		Severity: telemetrylog.ERROR,
		Body:     "This is a test message",
	}

	entryJson, err := json.Marshal(expectedLogEntry)
	assert.Nil(suite.T(), err)

	expectedMessage := &Message{
		Namespace: "testNamespace",
		Type:      LOG,
		Payload:   string(entryJson),
	}

	// create other side of the IPC channel
	receiveIpc := channelmock.NewFakeChannel(suite.mockContext.Log(), filewatcherbasedipc.ModeSurveyor, "telemetry")

	msg := <-receiveIpc.GetMessage()

	var actualMessage *Message
	err = json.Unmarshal([]byte(msg), &actualMessage)
	assert.Nil(suite.T(), err)

	// Check that correct message was received on the other side
	assert.Equal(suite.T(), expectedMessage, actualMessage)
}

func (suite *TelemetryTestSuite) TestEmitLog() {
	Initialize(suite.mockContext)

	logger := GetLogger("testNamespace")
	logger.EmitLog(telemetrylog.ERROR, "This is a test message")

	// create other side of the IPC channel
	receiveIpc := channelmock.NewFakeChannel(suite.mockContext.Log(), filewatcherbasedipc.ModeSurveyor, "telemetry")

	msg := <-receiveIpc.GetMessage()

	var actualMessage *Message
	err := json.Unmarshal([]byte(msg), &actualMessage)
	assert.Nil(suite.T(), err)

	var actualLogEntry *telemetrylog.Entry
	err = json.Unmarshal([]byte(actualMessage.Payload), &actualLogEntry)
	assert.Nil(suite.T(), err)

	assert.Equal(suite.T(), "testNamespace", actualMessage.Namespace)
	assert.Equal(suite.T(), "This is a test message", actualLogEntry.Body)
	assert.Equal(suite.T(), telemetrylog.ERROR, actualLogEntry.Severity)
}

func (suite *TelemetryTestSuite) TestEmitLogf() {
	Initialize(suite.mockContext)

	logger := GetLogger("testNamespace")
	logger.EmitLogf(telemetrylog.ERROR, "This is a test message %v, %v", 1, "hi")

	// create other side of the IPC channel
	receiveIpc := channelmock.NewFakeChannel(suite.mockContext.Log(), filewatcherbasedipc.ModeSurveyor, "telemetry")

	msg := <-receiveIpc.GetMessage()

	var actualMessage *Message
	err := json.Unmarshal([]byte(msg), &actualMessage)
	assert.Nil(suite.T(), err)

	var actualLogEntry *telemetrylog.Entry
	err = json.Unmarshal([]byte(actualMessage.Payload), &actualLogEntry)
	assert.Nil(suite.T(), err)

	assert.Equal(suite.T(), "testNamespace", actualMessage.Namespace)
	assert.Equal(suite.T(), "This is a test message 1, hi", actualLogEntry.Body)
	assert.Equal(suite.T(), telemetrylog.ERROR, actualLogEntry.Severity)
}
