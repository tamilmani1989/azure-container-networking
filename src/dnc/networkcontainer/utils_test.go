package networkcontainer

import (
	"dnc/api"
	"fmt"

	"github.com/Azure/azure-container-networking/telemetry"
)

type fakeCNSChannel struct {
	limit int
	err   *api.JsonErr
}

func (f *fakeCNSChannel) GetNCLimitForNode(_ string) (int, *api.JsonErr) {
	return f.limit, f.err
}

type dummyLogger struct{}

func (d *dummyLogger) Init(_ bool, _ chan telemetry.DNCReport) {}

func (d *dummyLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (d *dummyLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (d *dummyLogger) Close() {}

func (d *dummyLogger) SetLogFileLimits(_, _ int) {}
