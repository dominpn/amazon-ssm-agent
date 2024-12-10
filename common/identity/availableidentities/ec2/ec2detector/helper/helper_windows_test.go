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

//go:build windows
// +build windows

package helper

import (
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/platform"
	"github.com/stretchr/testify/assert"
)

var cacheInitCount int

func TestReadSystemProductInfo(t *testing.T) {
	cache.Flush()
	cacheInitCount = 0
	temp := initCacheAndGetData
	initCacheAndGetData = initCacheAndGetDataMock
	defer func() { initCacheAndGetData = temp }()

	assert.Equal(t, "version123", GetSystemInfo(XenVersionSystemInfoParam))
	assert.Equal(t, "uuid123", GetSystemInfo(NitroUuidSystemInfoParam))
	assert.Equal(t, "vendor123", GetSystemInfo(NitroVendorSystemInfoParam))
	assert.Equal(t, 1, cacheInitCount)
}

func TestReadSystemProductInfo_CacheMiss(t *testing.T) {
	cache.Flush()
	cacheInitCount = 0
	temp := initCacheAndGetData
	initCacheAndGetData = initCacheAndGetDataMock
	defer func() { initCacheAndGetData = temp }()

	assert.Equal(t, "version123", GetSystemInfo(XenVersionSystemInfoParam))
	assert.Equal(t, "uuid123", GetSystemInfo(NitroUuidSystemInfoParam))
	assert.Equal(t, "vendor123", GetSystemInfo(NitroVendorSystemInfoParam))
	assert.Equal(t, 1, cacheInitCount)

	GetSystemInfo("NonExistentAttribute")
	assert.Equal(t, 2, cacheInitCount)

	assert.Equal(t, "version123", GetSystemInfo(XenVersionSystemInfoParam))
	assert.Equal(t, "uuid123", GetSystemInfo(NitroUuidSystemInfoParam))
	assert.Equal(t, "vendor123", GetSystemInfo(NitroVendorSystemInfoParam))
	assert.Equal(t, 2, cacheInitCount)

}

func initCacheAndGetDataMock(key string) (data string) {
	cacheInitCount++
	biosData := platform.Win32_BIOS{
		Manufacturer:      "vendor123",
		SerialNumber:      "uuid123",
		SMBIOSBIOSVersion: "version123",
	}

	cache.Flush()
	cache.Put(NitroUuidSystemInfoParam, biosData.SerialNumber)
	cache.Put(NitroVendorSystemInfoParam, biosData.Manufacturer)
	cache.Put(XenVersionSystemInfoParam, biosData.SMBIOSBIOSVersion)
	cache.Put(XenUuidSystemInfoParam, biosData.SerialNumber) //itentionally overwriting SerialNumber in case the XenUuidSystemInfoParam value get changed

	data, _ = cache.Get(key)
	return data
}
