package aiwrapper

const (
	AppName = "DNC"
)

const (
	// Metrics
	NCProcessTimeMetric      = "NCProcessTimeMs"
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
	NCStateStr            = "NCState"

	// Status
	Failed    = "Failed"
	Succeeded = "Succeeded"
	Valid     = "Valid"
	Invalid   = "Invalid"
)
