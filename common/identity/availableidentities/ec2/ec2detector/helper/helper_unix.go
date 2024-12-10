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

//go:build freebsd || linux || netbsd || openbsd
// +build freebsd linux netbsd openbsd

package helper

import (
	"os"
	"strings"
)

const (
	NitroVendorSystemInfoParam = "/sys/class/dmi/id/sys_vendor"
	NitroUuidSystemInfoParam   = "/sys/class/dmi/id/product_uuid"
	XenVersionSystemInfoParam  = "/sys/hypervisor/version/extra"
	XenUuidSystemInfoParam     = "/sys/hypervisor/uuid"
)

var readFile = os.ReadFile

var initCacheAndGetData = func(key string) (data string) {
	if bytes, err := readFile(key); err == nil {
		data = strings.TrimSpace(string(bytes))
		cache.Put(key, data)
	}

	return data
}
