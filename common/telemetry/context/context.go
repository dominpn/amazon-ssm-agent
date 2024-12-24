package context

import (
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/common/identity"
)

type TelemetryContext interface {
	Log() log.T
	Identity() identity.IAgentIdentity
}
