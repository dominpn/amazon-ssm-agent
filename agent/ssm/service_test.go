// Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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
package ssm

import (
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/mocks/context"
	"github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/aws/amazon-ssm-agent/agent/times"
	"github.com/aws/aws-sdk-go/aws"
	awsmock "github.com/aws/aws-sdk-go/awstesting/mock"
	"github.com/aws/aws-sdk-go/service/ssm"
	ssmsdkmock "github.com/aws/aws-sdk-go/service/ssm/ssmiface/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Define the ssm service test suite. Add the log mock, external sdkmock and sdkService variable
// sdkMock use aws-sdk-go client mock object, sdkService is the struct define in service.go file
// Suite is the testify framework struct
type SsmServiceTestSuite struct {
	suite.Suite
	logMock    *log.Mock
	sdkService Service
	sdkMock    *ssm.SSM
}

// Setting up the testing environment for ssm service test.
// Give testing parameters e.g region and instanceId in awsConfig struct.
// Initialize the log mock struct.
func (suite *SsmServiceTestSuite) SetupTest() {
	logMock := log.NewMockLog()
	awsConfig := &aws.Config{}
	region := "us-east-1"
	awsConfig.Region = &region
	clientMock := awsmock.NewMockClient(awsConfig)
	// This clientMock will connect to an aws mock server which will validate the input variable
	sdkMock := &ssm.SSM{
		Client: clientMock,
	}
	suite.logMock = logMock
	suite.sdkMock = sdkMock
	suite.sdkService = &sdkService{
		context: context.NewMockDefault(),
		sdk:     sdkMock,
	}
}

// Testing function for update instance association
// Generate mock time stamp struct for testing. Set the agent mock status as "active"
func (suite *SsmServiceTestSuite) TestUpdateInstanceAssociationStatus() {
	// Prepare the testing variable
	date := times.ParseIso8601UTC("2018-07-05T13:45:23.017Z")
	executionResult := ssm.InstanceAssociationExecutionResult{
		Status:           aws.String("active"),
		ErrorCode:        aws.String("0"),
		ExecutionDate:    aws.Time(date),
		ExecutionSummary: aws.String("TestExecutionSummary"),
	}
	// Test the UpdateInstanceAssociationStatus function, assert the err is nil.
	res, err := suite.sdkService.UpdateInstanceAssociationStatus(suite.logMock, "associationID", "i-12345678", &executionResult)
	assert.Nil(suite.T(), err, "Err should be nil")
	assert.NotNil(suite.T(), res, "response shouldn't be nil")
}

// Test function for update empty instance information.
// This function only update the agent name, but not update agent version and agent status
func (suite *SsmServiceTestSuite) TestUpdateEmptyInstanceInformation() {
	// Test the UpdateEmptyInstanceInformation, assert error is nil
	response, err := suite.sdkService.UpdateEmptyInstanceInformation(suite.logMock, "2.2.3.2", "Amazon-ssm-agent")
	assert.Nil(suite.T(), err, "Err should be nil")
	assert.NotNil(suite.T(), response, "response shouldn't be nil")
}

// Test function for update instance information
// This function update the agent name, agent statuc, and agent version.
func (suite *SsmServiceTestSuite) TestUpdateInstanceInformation() {
	// Give mock value to test UpdateInstanceInformation, assert the error is nil, assert the log.Debug function get called.
	response, err := suite.sdkService.UpdateInstanceInformation(
		suite.logMock,
		"2.2.3.2",
		"active",
		"Amazon-ssm-agent",
		"us-east-1b",
		"use1-az2",
		"ssmmessages",
		"1724afd8-9092-429e-8b04-0708130c38f7",
		"Microsoft.Compute/virtualMachines",
		"centralus",
		"test-computer")
	assert.Nil(suite.T(), err, "Err should be nil")
	assert.NotNil(suite.T(), response, "response shouldn't be nil")
}

// Execute the test suite
func TestSsmServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SsmServiceTestSuite))
}

// TestUpdateInstanceInformation_WithSourceFields verifies that sourceId, sourceType,
// and sourceLocation are included in the API call when they are non-empty.
func TestUpdateInstanceInformation_WithSourceFields(t *testing.T) {
	logMock := log.NewMockLog()
	ssmSdkMock := &ssmsdkmock.SSMAPI{}
	mockCtx := context.NewMockDefault()

	svc := &sdkService{
		context: mockCtx,
		sdk:     ssmSdkMock,
	}

	var capturedInput *ssm.UpdateInstanceInformationInput
	ssmSdkMock.On("UpdateInstanceInformation", mock.AnythingOfType("*ssm.UpdateInstanceInformationInput")).Return(
		func(input *ssm.UpdateInstanceInformationInput) *ssm.UpdateInstanceInformationOutput {
			capturedInput = input
			return &ssm.UpdateInstanceInformationOutput{}
		},
		func(input *ssm.UpdateInstanceInformationInput) error {
			return nil
		},
	)

	_, err := svc.UpdateInstanceInformation(
		logMock,
		"3.0.0",
		"Active",
		"amazon-ssm-agent",
		"us-east-1a",
		"use1-az1",
		"ssmmessages",
		"test-source-id",
		"Microsoft.Compute/virtualMachines",
		"centralus",
		"test-computer",
	)

	assert.Nil(t, err)
	assert.NotNil(t, capturedInput)
	assert.Equal(t, "test-source-id", aws.StringValue(capturedInput.SourceId))
	assert.Equal(t, "Microsoft.Compute/virtualMachines", aws.StringValue(capturedInput.SourceType))
	assert.Equal(t, "centralus", aws.StringValue(capturedInput.SourceLocation))
	assert.Equal(t, "test-computer", aws.StringValue(capturedInput.ComputerName))
}

// TestUpdateInstanceInformation_EmptySourceFields verifies that sourceId, sourceType,
// and sourceLocation are not set when provided as empty strings.
func TestUpdateInstanceInformation_EmptySourceFields(t *testing.T) {
	logMock := log.NewMockLog()
	ssmSdkMock := &ssmsdkmock.SSMAPI{}
	mockCtx := context.NewMockDefault()

	svc := &sdkService{
		context: mockCtx,
		sdk:     ssmSdkMock,
	}

	var capturedInput *ssm.UpdateInstanceInformationInput
	ssmSdkMock.On("UpdateInstanceInformation", mock.AnythingOfType("*ssm.UpdateInstanceInformationInput")).Return(
		func(input *ssm.UpdateInstanceInformationInput) *ssm.UpdateInstanceInformationOutput {
			capturedInput = input
			return &ssm.UpdateInstanceInformationOutput{}
		},
		func(input *ssm.UpdateInstanceInformationInput) error {
			return nil
		},
	)

	_, err := svc.UpdateInstanceInformation(
		logMock,
		"3.0.0",
		"Active",
		"amazon-ssm-agent",
		"us-east-1a",
		"use1-az1",
		"ssmmessages",
		"",
		"",
		"",
		"",
	)

	assert.Nil(t, err)
	assert.NotNil(t, capturedInput)
	// When sourceId/sourceType/sourceLocation are empty, they should not be set (nil)
	assert.Nil(t, capturedInput.SourceId)
	assert.Nil(t, capturedInput.SourceType)
	assert.Nil(t, capturedInput.SourceLocation)
	// When computerName is empty, it should fall back to platform.Hostname
	assert.NotNil(t, capturedInput.ComputerName)
	assert.NotEqual(t, "", aws.StringValue(capturedInput.ComputerName))
}

// TestUpdateInstanceInformation_ComputerNameFallback verifies that when computerName
// is empty, the function falls back to platform.Hostname instead of using the empty value.
func TestUpdateInstanceInformation_ComputerNameFallback(t *testing.T) {
	logMock := log.NewMockLog()
	ssmSdkMock := &ssmsdkmock.SSMAPI{}
	mockCtx := context.NewMockDefault()

	svc := &sdkService{
		context: mockCtx,
		sdk:     ssmSdkMock,
	}

	var capturedInput *ssm.UpdateInstanceInformationInput
	ssmSdkMock.On("UpdateInstanceInformation", mock.AnythingOfType("*ssm.UpdateInstanceInformationInput")).Return(
		func(input *ssm.UpdateInstanceInformationInput) *ssm.UpdateInstanceInformationOutput {
			capturedInput = input
			return &ssm.UpdateInstanceInformationOutput{}
		},
		func(input *ssm.UpdateInstanceInformationInput) error {
			return nil
		},
	)

	_, err := svc.UpdateInstanceInformation(
		logMock,
		"3.0.0",
		"Active",
		"amazon-ssm-agent",
		"us-east-1a",
		"use1-az1",
		"ssmmessages",
		"source-123",
		"AWS::EC2::Instance",
		"us-east-1",
		"", // empty computerName should trigger fallback
	)

	assert.Nil(t, err)
	assert.NotNil(t, capturedInput)
	// ComputerName should not be empty - it should have fallen back to platform.Hostname
	assert.NotNil(t, capturedInput.ComputerName)
	assert.NotEqual(t, "", aws.StringValue(capturedInput.ComputerName))
}

// TestUpdateInstanceInformation_ComputerNameProvided verifies that when computerName
// is non-empty, it is used directly without calling platform.Hostname.
func TestUpdateInstanceInformation_ComputerNameProvided(t *testing.T) {
	logMock := log.NewMockLog()
	ssmSdkMock := &ssmsdkmock.SSMAPI{}
	mockCtx := context.NewMockDefault()

	svc := &sdkService{
		context: mockCtx,
		sdk:     ssmSdkMock,
	}

	var capturedInput *ssm.UpdateInstanceInformationInput
	ssmSdkMock.On("UpdateInstanceInformation", mock.AnythingOfType("*ssm.UpdateInstanceInformationInput")).Return(
		func(input *ssm.UpdateInstanceInformationInput) *ssm.UpdateInstanceInformationOutput {
			capturedInput = input
			return &ssm.UpdateInstanceInformationOutput{}
		},
		func(input *ssm.UpdateInstanceInformationInput) error {
			return nil
		},
	)

	_, err := svc.UpdateInstanceInformation(
		logMock,
		"3.0.0",
		"Active",
		"amazon-ssm-agent",
		"us-east-1a",
		"use1-az1",
		"ssmmessages",
		"source-123",
		"AWS::EC2::Instance",
		"us-east-1",
		"my-custom-hostname",
	)

	assert.Nil(t, err)
	assert.NotNil(t, capturedInput)
	assert.Equal(t, "my-custom-hostname", aws.StringValue(capturedInput.ComputerName))
}

// TestUpdateInstanceInformation_PartialSourceFields verifies behavior when only some
// source fields are provided (e.g., sourceId is set but sourceType and sourceLocation are empty).
func TestUpdateInstanceInformation_PartialSourceFields(t *testing.T) {
	logMock := log.NewMockLog()
	ssmSdkMock := &ssmsdkmock.SSMAPI{}
	mockCtx := context.NewMockDefault()

	svc := &sdkService{
		context: mockCtx,
		sdk:     ssmSdkMock,
	}

	var capturedInput *ssm.UpdateInstanceInformationInput
	ssmSdkMock.On("UpdateInstanceInformation", mock.AnythingOfType("*ssm.UpdateInstanceInformationInput")).Return(
		func(input *ssm.UpdateInstanceInformationInput) *ssm.UpdateInstanceInformationOutput {
			capturedInput = input
			return &ssm.UpdateInstanceInformationOutput{}
		},
		func(input *ssm.UpdateInstanceInformationInput) error {
			return nil
		},
	)

	_, err := svc.UpdateInstanceInformation(
		logMock,
		"3.0.0",
		"Active",
		"amazon-ssm-agent",
		"us-east-1a",
		"use1-az1",
		"ssmmessages",
		"source-123", // sourceId provided
		"",           // sourceType empty
		"",           // sourceLocation empty
		"my-host",
	)

	assert.Nil(t, err)
	assert.NotNil(t, capturedInput)
	// sourceId should be set since it's non-empty
	assert.Equal(t, "source-123", aws.StringValue(capturedInput.SourceId))
	// sourceType and sourceLocation should be nil since they are empty
	assert.Nil(t, capturedInput.SourceType)
	assert.Nil(t, capturedInput.SourceLocation)
}
