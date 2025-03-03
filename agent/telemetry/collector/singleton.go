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
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/context"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/telemetry/exporter"
	"github.com/aws/amazon-ssm-agent/common/filewatcherbasedipc"
	"github.com/aws/amazon-ssm-agent/common/identity"
	"github.com/aws/amazon-ssm-agent/common/telemetry"
	telemetryContext "github.com/aws/amazon-ssm-agent/common/telemetry/context"
	"github.com/aws/amazon-ssm-agent/common/telemetry/metric"
	"github.com/aws/amazon-ssm-agent/common/telemetry/telemetrylog"
)

var (
	singleton Collector

	// namespace -> filewatcherbasedipc channel mapping
	listenChannels map[string]filewatcherbasedipc.IPCChannel

	stopSignals map[string](chan bool)

	// WaitGroup to wait until all telemetry collection is stopped during shutdown
	listenWg *sync.WaitGroup

	pkgMutex = new(sync.RWMutex)
)

// this is for mocking support
var channelCreator = func(log log.T, identity identity.IAgentIdentity, mode filewatcherbasedipc.Mode, filename string) (filewatcherbasedipc.IPCChannel, error, bool) {
	return filewatcherbasedipc.CreateFileWatcherChannel(log, identity, mode, filename, false)
}

func Initialize(context context.T) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	// TODO : make the parameters configurable
	// 10 second aggregation
	// Exports every 5 minutes
	c, err := NewCollector(context, time.Second*10, time.Minute*5)

	if err != nil {
		return err
	}

	singleton = c
	listenChannels = make(map[string]filewatcherbasedipc.IPCChannel)
	stopSignals = make(map[string](chan bool))
	listenWg = new(sync.WaitGroup)

	return nil
}

func collectMetric(namespace string, metric metric.Metric[float64]) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton == nil {
		return fmt.Errorf("telemetry collector not initialized")
	}

	return singleton.Collect(namespace, metric)
}

func collectLog(namespace string, log telemetrylog.Entry) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton == nil {
		return fmt.Errorf("telemetry collector not initialized")
	}

	return singleton.CollectLog(namespace, log)
}

// StartCollection starts telemetry collection for a specified telemetry context
// internally, it creates the receiver end of the telemetry channel and collects the telemetry
// from it.
func StartCollection(context telemetryContext.TelemetryContext) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton == nil {
		return fmt.Errorf("telemetry collector not initialized")
	}

	log := context.Log()

	ipc, err, _ := channelCreator(log, context.Identity(), filewatcherbasedipc.ModeSurveyor, context.ChannelName())

	if err != nil {
		log.Warnf("could not initialize telemetry receiver for channel %v with error: %v", context.ChannelName(), err)
		return err
	}

	listenChannels[context.ChannelName()] = ipc

	stopSignal := make(chan bool)
	stopSignals[context.ChannelName()] = stopSignal

	listenWg.Add(1)

	go listenOnChannel(log, stopSignal, ipc)

	return nil
}

func StopCollection(context telemetryContext.TelemetryContext) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if stopSignals == nil {
		return fmt.Errorf("telemetry collector not initialized")
	}

	stopSignal := stopSignals[context.ChannelName()]

	if stopSignal == nil {
		return fmt.Errorf("telemetry collection for channel %v was not started", context.ChannelName())
	}

	listenChannel := listenChannels[context.ChannelName()]
	if listenChannel == nil {
		return fmt.Errorf("telemetry collection for channel %v was not started", context.ChannelName())
	}

	stopSignal <- true

	delete(listenChannels, context.ChannelName())
	delete(stopSignals, context.ChannelName())
	return nil
}

func listenOnChannel(log log.T, stopSignal chan bool, ipc filewatcherbasedipc.IPCChannel) {
	defer listenWg.Done()

	defer func() {
		if r := recover(); r != nil {
			log.Warnf("Telemetry channel listener panic: %v", r)
			log.Warnf("Stacktrace:\n%s", debug.Stack())
		}
	}()

	for {
		select {
		case <-stopSignal:
			ipc.Close()
			return

		case datagram, more := <-ipc.GetMessage():
			if !more {
				//safe close
				log.Debug("ipc channel closed, stop telemetry listener")
				return
			}

			log.Debugf("received datagram from %v: %v", ipc.GetPath(), datagram)
			if err := processDatagam([]byte(datagram)); err != nil {
				log.Debugf("error processing telemetry message: %v", err)
			}
		}
	}
}

func processDatagam(datagram []byte) error {
	var message telemetry.Message

	if err := json.Unmarshal(datagram, &message); err != nil {
		return err
	}

	switch message.Type {
	case telemetry.LOG:
		var logEntry telemetrylog.Entry

		if err := json.Unmarshal([]byte(message.Payload), &logEntry); err != nil {
			return err
		}

		return collectLog(message.Namespace, logEntry)
	case telemetry.METRIC:
		var metric metric.Metric[float64]

		if err := json.Unmarshal([]byte(message.Payload), &metric); err != nil {
			return err
		}

		return collectMetric(message.Namespace, metric)
	default:
		return fmt.Errorf("unknown message type: %v", message.Type)
	}
}

func AddExporter(exporter exporter.Exporter) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton == nil {
		return fmt.Errorf("cannot add exporter. telemetry collector not initialized")
	}

	singleton.AddExporter(exporter)
	return nil
}

func RemoveExporter(exporter exporter.Exporter) error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if singleton == nil {
		return fmt.Errorf("cannot remove exporter. telemetry collector not initialized")
	}

	singleton.RemoveExporter(exporter)
	return nil
}

func Shutdown() error {
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	if stopSignals == nil || listenWg == nil {
		return nil
	}

	// send the stop signals
	for _, stopSignal := range stopSignals {
		stopSignal <- true
	}

	// stop the collector
	singleton.Close()

	//wait for all collections to stop
	listenWg.Wait()

	singleton = nil
	listenChannels = nil
	stopSignals = nil
	listenWg = nil

	return nil
}
