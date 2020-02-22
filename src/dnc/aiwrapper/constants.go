package aiwrapper

const (
	AppName = "DNC"
)

const (
	// Metrics
	NCProcessTimeMetric          = "NCProcessTimeMs"
	NCCreationReqCountMetric     = "NCCreationRequestCount"
	DataStoreOperationTimeMetric = "DataStoreOperationTimeMs"

	// NC dimensions
	NetworkContainerIDStr = "NetworkContainerID"
	NetworkIDStr          = "VnetID"
	ContainerTypeStr      = "ContainerType"
	OrchestratorStr       = "Orchestrator"
	NodeIDStr             = "NodeID"
	RegionStr             = "Region"
	DNCVersionStr         = "DNCVersion"
	StatusStr             = "Status"
	NCStateStr            = "NCState"

	// DataStore operation dimensions
	OperationStr = "DataStoreOperation"
	TableNameStr = "TableName"

	// Status
	Failed    = "Failed"
	Succeeded = "Succeeded"
	Valid     = "Valid"
	Invalid   = "Invalid"
)
