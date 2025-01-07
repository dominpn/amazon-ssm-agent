package telemetrylog

import (
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

// NewMockDefault returns an instance of Mock with default expectations set.
func NewMockDefault() *Mock {
	log := new(Mock)
	return log
}

func (m *Mock) EmitLog(s Severity, v ...interface{}) error {
	args := m.Called(s, v)
	return args.Error(0)
}

func (m *Mock) EmitLogf(s Severity, format string, params ...interface{}) error {
	args := m.Called(s, format, params)
	return args.Error(0)
}
