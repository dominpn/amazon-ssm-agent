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
	"fmt"
	"math/rand/v2"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/jsonutil"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/session/communicator"
	mgsContracts "github.com/aws/amazon-ssm-agent/agent/session/contracts"
	"github.com/aws/amazon-ssm-agent/agent/telemetry/collector"
	"github.com/aws/amazon-ssm-agent/agent/telemetry/datastores"
	dynamicconfiguration "github.com/aws/amazon-ssm-agent/agent/telemetry/dynamic_configuration"
	"github.com/aws/amazon-ssm-agent/agent/version"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
	"github.com/gorilla/websocket"
	"github.com/twinj/uuid"
)

const (
	UnknownInstanceId           = "unknown"
	AgentTelemetryV2MessageType = "agent_telemetry_v2" // represents message type for V2 agent telemetry
)

// used to lock the send message process.
var pkgMutex = &sync.Mutex{}
var telemetryExporterInstance *controlChannelTelemetryExporter

type ITelemetryExporter interface {
	StartExporter()
	StopExporter()
	ShouldExport(namespace string) bool
}

// controlChannelTelemetryExporter helps us in scheduling the process to send telemetry to MGS
type controlChannelTelemetryExporter struct {
	channel communicator.IWebSocketChannel
	ctx     context.T
}

// GetControlChannelTelemetryExporter returns us the singleton instance of AuditLogTelemetry
func GetControlChannelTelemetryExporter(ctx context.T, channel communicator.IWebSocketChannel) *controlChannelTelemetryExporter {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if telemetryExporterInstance != nil {
		return telemetryExporterInstance
	}

	telemetryExporterInstance = &controlChannelTelemetryExporter{
		channel: channel,
		ctx:     ctx,
	}
	return telemetryExporterInstance
}

func (t *controlChannelTelemetryExporter) StartExporter() {
	defer func() {
		if r := recover(); r != nil {
			t.ctx.Log().Warnf("controlChannelTelemetryExporter StartExporter panic: %v", r)
			t.ctx.Log().Warnf("Stacktrace:\n%s", debug.Stack())
		}
	}()

	err := collector.AddExporter(t)

	if err != nil {
		t.ctx.Log().Errorf("Error while adding telemetry exporter: %v", err)
	}
}

func (t *controlChannelTelemetryExporter) StopExporter() {
	defer func() {
		if r := recover(); r != nil {
			t.ctx.Log().Warnf("controlChannelTelemetryExporter StopExporter panic: %v", r)
			t.ctx.Log().Warnf("Stacktrace:\n%s", debug.Stack())
		}
	}()

	err := collector.RemoveExporter(t)

	if err != nil {
		t.ctx.Log().Errorf("Error while removing telemetry exporter: %v", err)
	}
}

// For mocking support
var getDynamicConfig = dynamicconfiguration.GetCachedDynamicConfiguration
var controlChannelCheckTelemetryExportLuck = checkTelemetryExportLuck
var controlChannelIsTelemetryEnabled = isTelemetryEnabled
var controlChannelTooSoontoExportTelemetry = tooSoontoExportTelemetry

var randomPercentage = getRandomPercentage

func getRandomPercentage() int {
	return rand.IntN(100)
}

func checkTelemetryExportLuck(log log.T, namespace string) bool {
	telemetryEmissionLuck := randomPercentage()
	cachedDynamicConfiguration := getDynamicConfig()

	if cachedDynamicConfiguration == nil {
		log.Debugf("Dynamic Configuration Cache is empty, not emitting metrics for namespace %v", namespace)
		return false
	}

	config, ok := cachedDynamicConfiguration[namespace]
	if !ok {
		log.Debugf("No configuration found for namespace %v", namespace)
		return false
	}
	configuredPercentageLimit := config.PercentageLimit
	return telemetryEmissionLuck < configuredPercentageLimit
}

func isTelemetryEnabled(log log.T, namespace string) bool {
	cachedDynamicConfiguration := getDynamicConfig()
	if cachedDynamicConfiguration == nil {
		log.Debugf("Dynamic Configuration Cache is empty, defaulting to telemetry disabled for namespace %v", namespace)
		return false
	}

	config, ok := cachedDynamicConfiguration[namespace]
	if !ok {
		log.Debugf("No configuration found for namespace %v, defaulting to telemetry disabled", namespace)
		return false
	}

	return config.TelemetryDisabledTill < time.Now().Unix()
}

func tooSoontoExportTelemetry(log log.T, namespace string) bool {
	lastEmittedDataStore := datastores.TelemetryLastEmittedDataStore()
	lastEmittedTimeStamp := lastEmittedDataStore.Read(namespace)
	log.Debugf("Last emitted timestamp for namespace %v is %v", namespace, lastEmittedTimeStamp)
	exportPeriod := dynamicconfiguration.ExportPeriod(namespace)
	return (time.Now().Unix() - lastEmittedTimeStamp) < int64(exportPeriod*60)
}

func updateLastEmittedTimestamp(namespace string, lastEmittedTimeStamp int64) {
	lastEmittedDataStore := datastores.TelemetryLastEmittedDataStore()
	lastEmittedDataStore.Write(namespace, lastEmittedTimeStamp)
}

func (t *controlChannelTelemetryExporter) ShouldExport(namespace string) bool {
	return controlChannelCheckTelemetryExportLuck(t.ctx.Log(), namespace) && controlChannelIsTelemetryEnabled(t.ctx.Log(), namespace) && !controlChannelTooSoontoExportTelemetry(t.ctx.Log(), namespace)
}

func (t *controlChannelTelemetryExporter) Export(namespace string,
	metrics []metric.Metric[float64], logs []telemetrylog.Entry) (err error) {
	defer func() {
		if r := recover(); r != nil {
			t.ctx.Log().Warnf("controlChannelTelemetryExporter Export panic: %v", r)
			t.ctx.Log().Warnf("Stacktrace:\n%s", debug.Stack())
			err = fmt.Errorf("panic in controlChannelTelemetryExporter Export %v", r)
		}
	}()

	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	logger := t.ctx.Log()

	if !t.ShouldExport(namespace) {
		logger.Warnf("Skipping telemetry export for namespace %s", namespace)
		return nil
	}

	if len(metrics) == 0 && len(logs) == 0 {
		return nil
	}

	var payloadBytes []byte
	payloadBytes, err = createTelemetryPayload(metrics, logs)
	if err != nil {
		logger.Debugf("Error while preparing payload for telemetry: %v", err)
		return err
	}

	instanceId, err := t.ctx.Identity().InstanceID()
	if err != nil {
		instanceId = UnknownInstanceId
	}

	agentTelemetryV2 := &AgentTelemetryV2{
		SchemaVersion: 1,
		Namespace:     namespace,
		AgentVersion:  version.Version,
		InstanceId:    instanceId,
		Payload:       string(payloadBytes),
	}

	agentTelemetryBytes, err := json.Marshal(agentTelemetryV2)
	if err != nil { // return error only when telemetry to MGS is enabled
		return fmt.Errorf("unable to marshal AgentTelemetryV2 payload to json string: %s, err: %s", agentTelemetryBytes, err)
	}

	err = t.sendChannelContract(agentTelemetryBytes, AgentTelemetryV2MessageType)
	if err != nil {
		err = fmt.Errorf("unable to send message to MGS: %s", err)
	}

	updateLastEmittedTimestamp(namespace, time.Now().Unix())

	return err
}

// sendChannelContract sends the payload through the web socket connection with necessary packaging
func (t *controlChannelTelemetryExporter) sendChannelContract(payload []byte, messageType string) error {
	log := t.ctx.Log()
	agentMessage := &mgsContracts.AgentMessage{
		MessageType:    messageType,
		MessageId:      uuid.NewV4(),
		SchemaVersion:  1,
		CreatedDate:    uint64(time.Now().UnixNano() / 1000000), //nolint:gosec
		SequenceNumber: 0,
		Flags:          0,
		Payload:        payload,
	}
	log.Debugf("Sending payload to MGS: %v", jsonutil.Indent(string(payload)))
	agentBytes, err := agentMessage.Serialize(log)
	if err != nil {
		return err
	}
	return t.channel.SendMessage(log, agentBytes, websocket.BinaryMessage)
}
