package cnsinterface

import (
	"dnc/api"
	"dnc/datamodel"
)

// DisabledChannel bypasses CNS interaction, and returns fixed values
type DisabledChannel struct {
	NCLimit int
}

func (n *DisabledChannel) GetNCLimitForNode(_ string) (int, *api.JsonErr) {
	return n.NCLimit, nil
}

func (n *DisabledChannel) DeleteNC(_ *datamodel.NCConfiguration, _ map[string]string) *api.JsonErr {
	return nil
}
