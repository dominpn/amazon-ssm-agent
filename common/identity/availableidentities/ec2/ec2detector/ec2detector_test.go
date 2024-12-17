// Copyright 2022 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

package ec2detector

import (
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/aws/amazon-ssm-agent/common/identity/availableidentities/ec2/ec2detector/mocks"
	"github.com/stretchr/testify/assert"
)

func TestIsEC2Instance(t *testing.T) {
	logMock := log.NewMockLog()
	detector := ec2Detector{}
	trueSubDetector := &mocks.Detector{}
	falseSubDetector := &mocks.Detector{}
	temp := detectors
	defer func() { detectors = temp }()

	trueSubDetector.On("IsEc2", logMock).Return(true).Once()
	detectors = []Detector{trueSubDetector}
	assert.True(t, detector.IsEC2Instance(logMock))

	trueSubDetector.On("IsEc2", logMock).Return(true).Once()
	detectors = []Detector{trueSubDetector, falseSubDetector}
	assert.True(t, detector.IsEC2Instance(logMock))

	falseSubDetector.On("IsEc2", logMock).Return(false).Once()
	trueSubDetector.On("IsEc2", logMock).Return(true).Once()
	detectors = []Detector{falseSubDetector, trueSubDetector}
	assert.True(t, detector.IsEC2Instance(logMock))

	falseSubDetector.On("IsEc2", logMock).Return(false).Once()
	detectors = []Detector{falseSubDetector}
	assert.False(t, detector.IsEC2Instance(logMock))

	trueSubDetector.AssertExpectations(t)
	falseSubDetector.AssertExpectations(t)
}

func TestIsEC2Instance_ConfiguredReturnValue(t *testing.T) {
	logMock := log.NewMockLog()
	detector := ec2Detector{}
	subDetector := &mocks.Detector{}
	temp := detectors
	defer func() { detectors = temp }()

	detectors = []Detector{subDetector}
	detector.config = appconfig.SsmagentConfig{Identity: appconfig.IdentityCfg{Ec2SystemInfoDetectionResponse: "true"}}
	assert.True(t, detector.IsEC2Instance(logMock))

	detectors = []Detector{subDetector}
	detector.config = appconfig.SsmagentConfig{Identity: appconfig.IdentityCfg{Ec2SystemInfoDetectionResponse: "false"}}
	assert.False(t, detector.IsEC2Instance(logMock))

	subDetector.On("IsEc2", logMock).Return(true).Once()
	detectors = []Detector{subDetector}
	detector.config = appconfig.SsmagentConfig{Identity: appconfig.IdentityCfg{Ec2SystemInfoDetectionResponse: "unsupportedValue"}}
	assert.True(t, detector.IsEC2Instance(logMock))
}
