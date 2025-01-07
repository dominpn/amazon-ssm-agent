package context

import (
	"github.com/aws/amazon-ssm-agent/agent/log"
	logmocks "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/aws/amazon-ssm-agent/common/identity"
	identityMocks "github.com/aws/amazon-ssm-agent/common/identity/mocks"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

// NewMockDefault returns an instance of Mock with default expectations set.
func NewMockDefault() *Mock {
	ctx := new(Mock)
	log := logmocks.NewMockLog()
	agentIdentity := identityMocks.NewDefaultMockAgentIdentity()
	ctx.On("Log").Return(log)
	ctx.On("Identity").Return(agentIdentity)
	return ctx
}

// Identity mocks the Identity function.
func (m *Mock) Identity() identity.IAgentIdentity {
	args := m.Called()
	return args.Get(0).(identity.IAgentIdentity)
}

// Log mocks the Log function.
func (m *Mock) Log() log.T {
	args := m.Called()
	return args.Get(0).(log.T)
}
