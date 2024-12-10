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
	"github.com/aws/amazon-ssm-agent/agent/platform"
)

const (
	NitroUuidSystemInfoParam   = "SerialNumber"
	NitroVendorSystemInfoParam = "Manufacturer"
	XenVersionSystemInfoParam  = "SMBIOSBIOSVersion"
	XenUuidSystemInfoParam     = "SerialNumber"
)

var initCacheAndGetData = func(key string) (data string) {
	if biosData, err := platform.GetSingleWMIObject(platform.Win32_BIOS{}); err == nil {
		cache.Flush()

		cache.Put(NitroUuidSystemInfoParam, biosData.SerialNumber)
		cache.Put(NitroVendorSystemInfoParam, biosData.Manufacturer)
		cache.Put(XenVersionSystemInfoParam, biosData.SMBIOSBIOSVersion)
		cache.Put(XenUuidSystemInfoParam, biosData.SerialNumber) //itentionally overwriting SerialNumber in case the XenUuidSystemInfoParam value get changed

		data, _ = cache.Get(key)
	}

	return data
}
