package aiwrapper

const (
	AppName = "DNC"
)

const (
	// Metrics
	NCProcessTimeMetric                = "NCProcessTimeMs"
	NCCreationReqCountMetric           = "NCCreationRequestCount"
	SubnetAllocationsProcessTimeMetric = "SubnetAllocationProcessTimeMs"
	DataStoreOperationTimeMetric       = "DataStoreOperationTimeMs"
	LeakedNCEventMetric                = "LeakedNCEvent"

	// Dimensions
	NetworkContainerIDStr       = "NetworkContainerID"
	NetworkIDStr                = "VnetID"
	ContainerTypeStr            = "ContainerType"
	OrchestratorStr             = "Orchestrator"
	NodeIDStr                   = "NodeID"
	DNCVersionStr               = "DNCVersion"
	StatusStr                   = "Status"
	NCStateStr                  = "NCState"
	NCOperationErrorCodeStr     = "ErrorCode"
	NCOperationErrorHttpCodeStr = "ErrorHttpCode"
	SubNetStr                   = "SubNet"
	DNCNCSnapshotStr            = "DNCNCSnapshot"
	AzIdorNameStr               = "AzId"
	AznmAPINameStr              = "AznmAPIName"
	AznmAPIResultStr            = "AznmAPIResult"
	AllocationsPersistedStr     = "AllocationsPersisted"
	AllocationsRemovedStr       = "AllocationsRemoved"
	SubnetSizeStr               = "SubnetSize"
	SubnetAllocationOpStr       = "SubnetAllocationOperation"

	// DataStore operation dimensions
	OperationStr = "DataStoreOperation"
	TableNameStr = "TableName"

	// Status
	Failed    = "Failed"
	Succeeded = "Succeeded"
	Valid     = "Valid"
	Invalid   = "Invalid"
)
