// Copyright 2024 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

//go:build freebsd || linux || netbsd || openbsd
// +build freebsd linux netbsd openbsd

// Package platform contains platform specific utilities.
package platform

import (
	"fmt"
	"testing"

	logger "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/stretchr/testify/assert"
)

func TestVersion_PlatformWithBrackets(t *testing.T) {
	logMock := logger.NewMockLog()
	tmpFileExists := fileExists
	fileExists = func(filePath string) bool {
		return filePath == systemReleaseFile
	}
	defer func() { fileExists = tmpFileExists }()
	tmpReadAllText := readAllText
	readAllText = func(filePath string) (text string, err error) {
		return "Red Hat Enterprise Linux Server release 6.10 (Santiago)", nil
	}
	defer func() { readAllText = tmpReadAllText }()
	name, version, err := getPlatformDetails(logMock)
	assert.Equal(t, "Red Hat Enterprise Linux Server", name)
	assert.Equal(t, "6.10", version)
	assert.Nil(t, err)
}

func TestVersion_PlatformWithOutBrackets(t *testing.T) {
	logMock := logger.NewMockLog()
	tmpFileExists := fileExists
	fileExists = func(filePath string) bool {
		return filePath == systemReleaseFile
	}
	defer func() { fileExists = tmpFileExists }()
	tmpReadAllText := readAllText
	readAllText = func(_ string) (text string, err error) {
		return "Red Hat Enterprise Linux Server release 7", nil
	}
	defer func() { readAllText = tmpReadAllText }()
	name, version, err := getPlatformDetails(logMock)
	assert.Equal(t, "Red Hat Enterprise Linux Server", name)
	assert.Equal(t, "7", version)
	assert.Nil(t, err)
}

func TestPlatformNameAndVersion(t *testing.T) {
	ClearCache()
	tmpFileExists := fileExists
	fileExists = func(filePath string) bool {
		return filePath == systemReleaseFile
	}
	defer func() { fileExists = tmpFileExists }()
	tmpReadAllText := readAllText
	readAllText = func(_ string) (text string, err error) {
		return "Red Hat Enterprise Linux Server release 6.78", nil
	}
	defer func() { readAllText = tmpReadAllText }()
	logObj := logger.NewMockLog()
	platformName, err := PlatformName(logObj)
	assert.Equal(t, "Red Hat Enterprise Linux Server", platformName)
	assert.Nil(t, err)
	platformVersion, err := PlatformVersion(logObj)
	assert.Equal(t, "6.78", platformVersion)
	assert.Nil(t, err)
}

func TestPlatformNameAndVersionWithError(t *testing.T) {
	ClearCache()
	tmpFileExists := fileExists
	fileExists = func(filePath string) bool {
		return filePath == systemReleaseFile
	}
	defer func() { fileExists = tmpFileExists }()
	tmpReadAllText := readAllText
	readAllText = func(_ string) (text string, err error) {
		return "", fmt.Errorf("platform name and version error")
	}
	defer func() { readAllText = tmpReadAllText }()
	logObj := logger.NewMockLog()
	platformName, err := PlatformName(logObj)
	assert.Equal(t, notAvailableMessage, platformName)
	assert.NotNil(t, err)
	platformVersion, err := PlatformVersion(logObj)
	assert.Equal(t, notAvailableMessage, platformVersion)
	assert.NotNil(t, err)
}
