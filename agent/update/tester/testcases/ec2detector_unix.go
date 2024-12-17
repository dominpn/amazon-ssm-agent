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
// permissions and limitations under the License

//go:build freebsd || linux || netbsd || openbsd
// +build freebsd linux netbsd openbsd

package testcases

import (
	"errors"

	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/platform"
	testCommon "github.com/aws/amazon-ssm-agent/agent/update/tester/common"
)

const (
	ec2DetectorTestCaseName = "UnixEc2Detector"
)

func getSystemHostInfo(log log.T) (HostInfo, error) {
	var hostInfo HostInfo

	vendor, _ := platform.GetSystemInfo(log, platform.NitroVendorSystemInfoParamKey)
	hostInfo.Vendor = cleanBiosString(vendor)
	version, _ := platform.GetSystemInfo(log, platform.XenVersionSystemInfoParamKey)
	hostInfo.Version = cleanBiosString(version)

	if hostInfo.Version == "" && hostInfo.Vendor == "" {
		return hostInfo, errors.New(failedToGetVendorAndVersion)
	}

	var uuid string
	if uuid, _ = platform.GetSystemInfo(log, platform.XenUuidSystemInfoParamKey); uuid == "" {
		if uuid, _ = platform.GetSystemInfo(log, platform.NitroUuidSystemInfoParamKey); uuid == "" {
			return hostInfo, errors.New(failedToGetUuid)
		}
	}
	hostInfo.Uuid = cleanBiosString(uuid)

	return hostInfo, nil
}

func (l *Ec2DetectorTestCase) queryHostInfo() {
	l.systemHostInfo, l.systemErr = getSystemHostInfo(l.context.Log())
	l.smbiosHostInfo, l.smbiosErr = getSmbiosHostInfo(l.context.Log())
}

func (l *Ec2DetectorTestCase) generatePlatformTestResult() (testCommon.TestResult, string) {
	return l.generateTestResult(l.systemHostInfo, l.systemErr, l.smbiosHostInfo, l.smbiosErr)
}
