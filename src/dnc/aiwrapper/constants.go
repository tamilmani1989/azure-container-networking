package aiwrapper

const (
	AppName = "DNC"
)

const (
	// Metrics
	NCProcessTimeMetric                = "NCProcessTimeMs"
	NCRequestCountMetric               = "NCRequestCount"
	SubnetAllocationsProcessTimeMetric = "SubnetAllocationProcessTimeMs"
	DataStoreOperationTimeMetric       = "DataStoreOperationTimeMs"
	LeakedNCEventMetric                = "LeakedNCEvent"

	// DNC-PubSub metric
	PubsubOpFailureMetric         = "PubsubOpFailure"
	PubsubConnectionFailureMetric = "PubsubConnectionFailure"
	PubsubOpLatencyMetric         = "PubsubOpLatency"

	PubsubPathStr                = "PubsubPath"
	PubsubOperationStr           = "PubsubOperation"
	PubsubJobIdStr               = "PubsubJobId"
	PubsubAzIdStr                = "PubsubAzId"
	PubsubAzVipStr               = "PubsubAzVip"
	PubsubConnectionStringStr    = "PubsubConnectionEp"
	PubsubConnectionTypeStr      = "PubsubConnectionType"
	PubsubCertNameStr            = "PubsubCertName"
	PubsubConnectionErrorCodeStr = "PubsubConnectionErrorCode"

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
	NCRequestTypeStr            = "NCRequestType"

	// DataStore operation dimensions
	OperationStr = "DataStoreOperation"
	TableNameStr = "TableName"

	// Status
	Failed    = "Failed"
	Succeeded = "Succeeded"
	Valid     = "Valid"
	Invalid   = "Invalid"
)
