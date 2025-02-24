package context

import (
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/common/identity"
)

type TelemetryContext interface {
	ChannelName() string
	Log() log.T
	Identity() identity.IAgentIdentity
}

type telemetryContext struct {
	channelName string
	log         log.T
	identity    identity.IAgentIdentity
}

func NewTelemetryContext(channelName string, log log.T, identity identity.IAgentIdentity) TelemetryContext {
	return &telemetryContext{
		channelName: channelName + "_telemetry",
		log:         log,
		identity:    identity,
	}
}

// Identity implements TelemetryContext.
func (t *telemetryContext) Identity() identity.IAgentIdentity {
	return t.identity
}

// Log implements TelemetryContext.
func (t *telemetryContext) Log() log.T {
	return t.log
}

// MainChannelName implements TelemetryContext.
func (t *telemetryContext) ChannelName() string {
	return t.channelName
}
