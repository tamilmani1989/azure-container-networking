package loggermock

import (
	"github.com/Azure/azure-container-networking/telemetry"
)

// NoOpLogger can be used when a loggershim.LoggerInterface is required, but does not log anything.
// Useful in testing when a component's logging is more chatty than helpful.
type NoOpLogger struct{}

// NewNoOpLogger returns a new NoOpLogger
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

func (d *NoOpLogger) Debugf(_ string, _ ...interface{}) {}

func (d *NoOpLogger) Errorf(_ string, _ ...interface{}) {}

func (_ *NoOpLogger) Init(_ bool, _ chan telemetry.DNCReport) {}

func (_ *NoOpLogger) Close() {}

func (_ *NoOpLogger) SetLogFileLimits(_, _ int) {}
