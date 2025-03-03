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
	SSMAgentNamespace    = "SSMAgent"
	CoreAgentChannelName = "core"
)

var pkgMutex = new(sync.RWMutex)

type telemetry struct {
	context     context.TelemetryContext
	fileChannel filewatcherbasedipc.IPCChannel
}

func getTelemetry() (*telemetry, error) {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton == nil {
		return nil, errors.New("telemetry is not initialized")
	}

	return singleton, nil
}

// this is for mocking support
var channelCreator = func(log logger.T, identity identity.IAgentIdentity, mode filewatcherbasedipc.Mode, filename string) (filewatcherbasedipc.IPCChannel, error, bool) {
	return filewatcherbasedipc.CreateFileWatcherChannel(log, identity, mode, filename, false)
}

func Initialize(context context.TelemetryContext) (err error) {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton != nil {
		return errors.New("telemetry is already initialized")
	}

	log := context.Log()

	ipc, err, _ := channelCreator(log, context.Identity(), filewatcherbasedipc.ModeRespondent, context.ChannelName())

	if err != nil {
		log.Errorf(err.Error())
		return err
	}

	singleton = &telemetry{context: context, fileChannel: ipc}

	log.Info("Telemetry initialized")
	return nil
}

func Shutdown() {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton != nil {
		singleton.shutdown()
	}
	singleton = nil
}

func (t *telemetry) shutdown() {
	t.fileChannel.Destroy()
}

// emitLog is the internal function which emits logs to the IPC channel
func (t *telemetry) emitLog(namespace string, time time.Time, severity telemetrylog.Severity, message string) (err error) {
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
	return t.fileChannel.Send(string(ipcMessageJson))
}

func (t *telemetry) emitIntegerMetric(namespace string, name string, unit string, kind metric.Kind, time time.Time, value int64) (err error) {
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
	return t.fileChannel.Send(string(ipcMessageJson))
}
