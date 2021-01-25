package loggermock

import (
	"fmt"
	"github.com/Azure/azure-container-networking/telemetry"
)

type DummyLogger struct {

}

// NewDummyLogger :- Create new DummyLogger
func NewDummyLogger() *DummyLogger {
	dummyLogger := new(DummyLogger)
	return dummyLogger
}


func (d *DummyLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (d *DummyLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (_ *DummyLogger) Init(_ bool, _ chan telemetry.DNCReport) {}

func (_ *DummyLogger) Close() {}

func (_ *DummyLogger) SetLogFileLimits(_, _ int) {}

