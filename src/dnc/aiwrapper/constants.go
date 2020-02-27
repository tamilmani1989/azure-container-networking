package aiwrapper

const (
	AppName = "DNC"
)

const (
	// Metrics
	NCProcessTimeMetric          = "NCProcessTimeMs"
	NCCreationReqCountMetric     = "NCCreationRequestCount"
	DataStoreOperationTimeMetric = "DataStoreOperationTimeMs"

	// Dimensions
	NetworkContainerIDStr   = "NetworkContainerID"
	NetworkIDStr            = "VnetID"
	ContainerTypeStr        = "ContainerType"
	OrchestratorStr         = "Orchestrator"
	NodeIDStr               = "NodeID"
	RegionStr               = "Region"
	DNCVersionStr           = "DNCVersion"
	StatusStr               = "Status"
	NCStateStr              = "NCState"
	NCOperationErrorCodeStr = "ErrorCode"
	SubNetStr               = "SubNet"
	DNCNCSnapshotStr        = "DNCNCSnapshot"

	// DataStore operation dimensions
	OperationStr = "DataStoreOperation"
	TableNameStr = "TableName"

	// Status
	Failed    = "Failed"
	Succeeded = "Succeeded"
	Valid     = "Valid"
	Invalid   = "Invalid"
)
