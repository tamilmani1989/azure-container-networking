package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"fmt"

	"github.com/Azure/azure-container-networking/telemetry"
)

type fakeCNSChannel struct {
	limit int
	err   *api.JsonErr
}

func (f *fakeCNSChannel) DeleteNC(_ *datamodel.NCConfiguration, _ map[string]string) *api.JsonErr {
	return nil
}

func (f *fakeCNSChannel) GetNCLimitForNode(_ string) (int, *api.JsonErr) {
	return f.limit, f.err
}

func (f *fakeCNSChannel) SetOrchestratorType(_, _, _ string) *api.JsonErr {
	return nil
}

type dummyLogger struct {
	noOpLogger
}

func (d *dummyLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (d *dummyLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

type noOpLogger struct{}

func (_ *noOpLogger) Init(_ bool, _ chan telemetry.DNCReport) {}

func (_ *noOpLogger) Debugf(_ string, _ ...interface{}) {}

func (_ *noOpLogger) Errorf(_ string, _ ...interface{}) {}

func (_ *noOpLogger) Close() {}

func (_ *noOpLogger) SetLogFileLimits(_, _ int) {}
