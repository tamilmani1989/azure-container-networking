package netagentcontracts

import "github.com/Azure/azure-container-networking/telemetry"

// DNCReport structure.
type DNCReport struct {
	IsNewInstance bool
	CPUUsage      string
	MemoryUsage   string
	Processes     string
	EventMessage  string
	PartitionKey  string
	Allocations   string
	Metadata      telemetry.Metadata `json:"compute"`
}
