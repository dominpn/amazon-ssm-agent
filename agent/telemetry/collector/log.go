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

package collector

import (
	"sync"

	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
)

type logCollector struct {
	mtx        *sync.Mutex
	folderPath string
}

func newLogCollector(context context.T) *logCollector {
	return &logCollector{
		mtx:        &sync.Mutex{},
		folderPath: "",
	}
}

func (c *logCollector) CollectLog(namespace string, log telemetrylog.Entry) error {
	//TODO implement me
	panic("implement me")
}

func (c *logCollector) Fetch(namespace string, limit int) ([]telemetrylog.Entry, error) {
	//TODO implement me
	panic("implement me")
}

func (c *logCollector) Close() error {
	//TODO implement me
	panic("implement me")
}
