package aiwrapper

const (
	appName = "DNC"
)

const (
	// Metrics
	NCCreationTimeMetric     = "NCCreationTimeMs"
	NCCreationReqCountMetric = "NCCreationRequestCount"

	// Dimensions
	NetworkContainerIDStr = "NetworkContainerID"
	NetworkIDStr          = "VnetID"
	ContainerTypeStr      = "ContainerType"
	OrchestratorStr       = "Orchestrator"
	NodeIDStr             = "NodeID"
	RegionStr             = "Region"
	DNCVersionStr         = "DNCVersion"
	StatusStr             = "Status"

	// Status
	Failed    = "Failed"
	Succeeded = "Succeeded"
	Valid     = "Valid"
	Invalid   = "Invalid"
)
