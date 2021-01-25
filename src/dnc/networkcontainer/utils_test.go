package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
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
