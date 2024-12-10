// Copyright 2021 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

//go:build windows
// +build windows

// Package platform contains platform specific utilities.
package platform

import (
	"fmt"
	"testing"

	logger "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/stretchr/testify/assert"
)

func TestVersion_Positive(t *testing.T) {
	ClearCache()
	logMock := logger.NewMockLog()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: "6.2323.23"}, nil
	}
	defer func() { getPlatformDetails = temp }()
	isWin2012, err := isPlatformWindowsServer2012OrEarlier(logMock)
	assert.True(t, isWin2012, "Should return true")
	assert.Nil(t, err)
	isWin2025, err := isPlatformWindowsServer2025OrLater(logMock)
	assert.False(t, isWin2025, "Should return false")
	assert.Nil(t, err)

	ClearCache()
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: "20.2323.23"}, nil
	}
	isWin2012, err = isPlatformWindowsServer2012OrEarlier(logMock)
	assert.False(t, isWin2012, "Should return false")
	assert.Nil(t, err)
	isWin2025, err = isPlatformWindowsServer2025OrLater(logMock)
	assert.True(t, isWin2025, "Should return true")
	assert.Nil(t, err)
}

func TestVersion_Negative(t *testing.T) {
	ClearCache()
	logMock := logger.NewMockLog()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: "0.022"}, nil
	}
	defer func() { getPlatformDetails = temp }()
	isWin2012, err := isPlatformWindowsServer2012OrEarlier(logMock)
	assert.True(t, isWin2012, "Should return true")
	assert.Nil(t, err)
	isWin2025, err := isPlatformWindowsServer2025OrLater(logMock)
	assert.False(t, isWin2025, "Should return false")
	assert.Nil(t, err)

	ClearCache()
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: "dsdsds23323"}, nil
	}
	isWin2012, err = isPlatformWindowsServer2012OrEarlier(logMock)
	assert.False(t, isWin2012, "Should return false")
	assert.NotNil(t, err)
	isWin2025, err = isPlatformWindowsServer2025OrLater(logMock)
	assert.False(t, isWin2025, "Should return false")
	assert.NotNil(t, err)

	ClearCache()
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: ""}, nil
	}
	isWin2012, err = isPlatformWindowsServer2012OrEarlier(logMock)
	assert.False(t, isWin2012, "Should return false")
	assert.NotNil(t, err)
	isWin2025, err = isPlatformWindowsServer2025OrLater(logMock)
	assert.False(t, isWin2025, "Should return false")
	assert.NotNil(t, err)

	ClearCache()
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: ""}, fmt.Errorf("test1")
	}
	isWin2012, err = isPlatformWindowsServer2012OrEarlier(logMock)
	assert.False(t, isWin2012, "Should return false")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test1")
	isWin2025, err = isPlatformWindowsServer2025OrLater(logMock)
	assert.False(t, isWin2025, "Should return false")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test1")
}

func TestPlatformName(t *testing.T) {
	ClearCache()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Caption: "Windows 2025"}, nil
	}
	defer func() { getPlatformDetails = temp }()
	logObj := logger.NewMockLog()
	platformName, err := PlatformName(logObj)
	assert.Equal(t, "Windows 2025", platformName)
	assert.Nil(t, err)
}

func TestPlatformNameWithError(t *testing.T) {
	ClearCache()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{}, fmt.Errorf("platform name error")
	}
	defer func() { getPlatformDetails = temp }()
	logObj := logger.NewMockLog()
	platformName, err := PlatformName(logObj)
	assert.Equal(t, notAvailableMessage, platformName)
	assert.NotNil(t, err)
}

func TestPlatformVersion(t *testing.T) {
	ClearCache()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{Version: "20.2323.23"}, nil
	}
	defer func() { getPlatformDetails = temp }()
	logObj := logger.NewMockLog()
	platformVersion, err := PlatformVersion(logObj)
	assert.Equal(t, "20.2323.23", platformVersion)
	assert.Nil(t, err)
}

func TestPlatformVersionWithError(t *testing.T) {
	ClearCache()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{}, fmt.Errorf("platform version error")
	}
	defer func() { getPlatformDetails = temp }()
	logObj := logger.NewMockLog()
	platformVersion, err := PlatformVersion(logObj)
	assert.Equal(t, notAvailableMessage, platformVersion)
	assert.NotNil(t, err)
}

func TestPlatformSku(t *testing.T) {
	ClearCache()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{OperatingSystemSKU: 123}, nil
	}
	defer func() { getPlatformDetails = temp }()
	logObj := logger.NewMockLog()
	platformSku, err := PlatformSku(logObj)
	assert.Equal(t, "123", platformSku)
	assert.Nil(t, err)
}

func TestPlatformSkuWithError(t *testing.T) {
	ClearCache()
	temp := getPlatformDetails
	getPlatformDetails = func(_ Win32_OperatingSystem) (Win32_OperatingSystem, error) {
		return Win32_OperatingSystem{}, fmt.Errorf("platform sku error")
	}
	defer func() { getPlatformDetails = temp }()
	logObj := logger.NewMockLog()
	platformSku, err := PlatformSku(logObj)
	assert.Equal(t, notAvailableMessage, platformSku)
	assert.NotNil(t, err)
}
