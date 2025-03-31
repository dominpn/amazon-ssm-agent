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

package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	logger "github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/common/filewatcherbasedipc"
	"github.com/aws/amazon-ssm-agent/common/identity"
	"github.com/aws/amazon-ssm-agent/common/telemetry/context"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
)

var (
	singleton *telemetry
)

const (
	SSMAgentNamespace      = "SSMAgent"
	CoreAgentChannelName   = "core"
	AgentWorkerChannelName = "agent_worker"
)

var singletonMtx = new(sync.RWMutex)

type telemetry struct {
	context context.TelemetryContext

	// fileChannalMtx protects the fileChannel variable
	fileChannalMtx *sync.RWMutex
	fileChannel    filewatcherbasedipc.IPCChannel
}

func getTelemetry() (*telemetry, error) {
	singletonMtx.RLock()
	defer singletonMtx.RUnlock()

	if singleton == nil {
		return nil, errors.New("telemetry is not initialized")
	}

	return singleton, nil
}

// this is for mocking support
var channelCreator = func(log logger.T, identity identity.IAgentIdentity,
	filename string) (filewatcherbasedipc.IPCChannel, error, bool) {
	return filewatcherbasedipc.CreateRollingFileWatcherChannel(log, identity, filewatcherbasedipc.ModeMaster, filename, false, 1000)
}

func Initialize(context context.TelemetryContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			context.Log().Warnf("telemetry Initialize panic: %v", r)
			context.Log().Warnf("Stacktrace:\n%s", debug.Stack())
			err = fmt.Errorf("panic in telemetry.Initialize %v", r)
		}
	}()

	singletonMtx.Lock()
	defer singletonMtx.Unlock()

	if singleton != nil {
		return errors.New("telemetry is already initialized")
	}

	log := context.Log()

	ipc, err, _ := channelCreator(log, context.Identity(), context.ChannelName())

	if err != nil {
		log.Errorf(err.Error())
		return err
	}

	singleton = &telemetry{context: context, fileChannalMtx: &sync.RWMutex{}, fileChannel: ipc}

	log.Info("Telemetry initialized")
	return nil
}

func Shutdown() {
	defer func() {
		if r := recover(); r != nil && singleton != nil {
			singleton.context.Log().Warnf("telemetry Shutdown panic: %v", r)
			singleton.context.Log().Warnf("Stacktrace:\n%s", debug.Stack())
		}
	}()

	singletonMtx.Lock()
	defer singletonMtx.Unlock()

	if singleton != nil {
		singleton.shutdown()
	}
	singleton = nil
}

func (t *telemetry) shutdown() {
	t.fileChannalMtx.Lock()
	defer t.fileChannalMtx.Unlock()

	if t.fileChannel != nil {
		t.fileChannel.Destroy()
		t.fileChannel = nil
	}
}

// emitLog is the internal function which emits logs to the IPC channel
func (t *telemetry) emitLog(namespace string, time time.Time, severity telemetrylog.Severity, message string) (err error) {
	defer func() {
		if r := recover(); r != nil && singleton != nil {
			singleton.context.Log().Warnf("telemetry emitLog panic: %v", r)
			singleton.context.Log().Warnf("Stacktrace:\n%s", debug.Stack())
			err = fmt.Errorf("panic in telemetry.emitLog %v", r)
		}
	}()

	message = TruncateLog(message)
	entry := &telemetrylog.Entry{Time: time.UTC(), Severity: severity, Body: message}

	entryJson, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	ipcMessage := &Message{
		Namespace: namespace,
		Type:      LOG,
		Payload:   string(entryJson),
	}

	ipcMessageJson, err := json.Marshal(ipcMessage)
	if err != nil {
		return err
	}

	t.fileChannalMtx.RLock()
	defer t.fileChannalMtx.RUnlock()

	if t.fileChannel == nil {
		return errors.New("telemetry is not initialized")
	}
	go t.sendIpcMessage(string(ipcMessageJson))
	return nil
}

func (t *telemetry) emitIntegerMetric(namespace, name string, unit metric.Unit, kind metric.Kind, time time.Time, value int64) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if singleton != nil {
				singleton.context.Log().Warnf("telemetry emitIntegerMetric panic: %v", r)
				singleton.context.Log().Warnf("Stacktrace:\n%s", debug.Stack())
			}
			err = fmt.Errorf("panic in telemetry.emitIntegerMetric %v", r)
		}
	}()

	entry := &metric.Metric[int64]{
		Name:       name,
		Unit:       unit,
		Kind:       kind,
		DataPoints: []metric.DataPoint[int64]{{StartTime: time.UTC(), EndTime: time.UTC(), Value: value}},
	}

	entryJson, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	ipcMessage := &Message{
		Namespace: namespace,
		Type:      METRIC,
		Payload:   string(entryJson),
	}

	ipcMessageJson, err := json.Marshal(ipcMessage)
	if err != nil {
		return err
	}

	t.fileChannalMtx.RLock()
	defer t.fileChannalMtx.RUnlock()

	if t.fileChannel == nil {
		return errors.New("telemetry is not initialized")
	}
	go t.sendIpcMessage(string(ipcMessageJson))
	return nil
}

// sendIpcMessage sends message to the IPC channel
func (t *telemetry) sendIpcMessage(message string) {
	t.fileChannalMtx.RLock()
	defer t.fileChannalMtx.RUnlock()

	err := t.fileChannel.Send(message)

	t.context.Log().Warnf("Sending telemetry IPC message failed with: %v", err)
}
