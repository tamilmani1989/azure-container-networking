package datamodel

const (
	subnetConfigurationTableName                     = "SubnetConfigurationTable"
	nodeTableName                                    = "NodeConfigurationTable"
	ipReservationsTableName                          = "IPReservationsTable"
	pendingWorkItemsTableName                        = "PendingWorkItemsTable"
	networkConfigurationTableName                    = "NetworkConfigurationTable"
	networkContainerTableName                        = "NetworkContainerTable"
	tagDefinitionTableName                           = "TagDefinitionsTable"
	ipAllocationsTableName                           = "IPAllocationsTable"
	sharedSubnetDelegationMigrationTableName         = "SharedSubnetDelegationMigrationTable"
	sharedSubnetDelegationMigrationRegionalTableName = "SharedSubnetDelegationMigrationRegionalTable"
	MasterElectionTableName                          = "MasterElectionTable" // MasterElectionTableName - DNC master election table name
	ncPendingPublishContextTableName                 = "NcPendingPublishContextTable" // Table containing pending NC publish / unpublish context
	featureMigrationTableName                        = "FeatureMigrationTable" // Holds the info about different features that need migration
	// Table containing pending cleanup request created for asynchronous cleanup of internal state
	// For now it will be used to store IP cleanup request for a given NC during NC delete or secondary ips scale down.
	pendingCleanupRequestTableName                   = "PendingCleanupRequestTable"

	networkKey                 = "Network_%s"
	subnetConfigurationKey     = "Network_%s_Subnet_%s"
	ipReservationKey           = "Network_%s_Reservation_%s"
	networkcontainerKey        = "NC_%s"
	nodeKey                    = "Node_%s"
	ncKey                      = "Network_%s_ID_%s"
	tagDefinitionKey           = "Tag_%s_%s"
	ncPendingPublishContextKey = "Network_%s_NC_%s_PubSubId_%s"
)

const (
	subnetConfigurationTag             = "SubnetConfiguration"
	networkConfigurationTag            = "NetworkConfiguration"
	reservationConfigurationTag        = "ReservationConfiguration"
	nodeConfigurationTag               = "NodeConfiguration"
	requestContextTag                  = "RequestContext"
	networkContainerConfigurationTag   = "NetworkContainerConfiguration"
	tagDefinitionsTag                  = "TagDefinitions"
	ipAllocationTag                    = "IPAllocation"
	sharedSubnetDelegationMigrationTag = "SharedSubnetDelegationMigrationComplete"
	masterElectionTag                  = "MasterElection"
	ncPendingPublishContextTag         = "NcPendingPublishContext"
	featureMigrationTag                = "FeatureMigrationComplete"
	pendingCleanupRequestTag           = "PendingCleanupRequest"
)

const (
	// NCRepositoryName - Repository name for network container.
	NCRepositoryName = "NCRepository"

	// ReservationRepositoryName - Repository name for reservation.
	ReservationRepositoryName = "ReservationRepository"

	// NodeRepositoryName - Repository name for node.
	NodeRepositoryName = "NodeRepository"

	// NetworkRepositoryName - Repository name for network.
	NetworkRepositoryName = "NetworkRepository"

	// TagRepositoryName - Repository name for tag defintions
	TagRepositoryName = "TagRepository"

	// NCPendingPublishRepositoryName - Repository name for NC Pending publish/unpublish tasks
	NCPendingPublishRepositoryName = "NCPendingPublishRepository"

	// FeatureMigrationRespositoryName  - Repository name for FeatureMigration table
	FeatureMigrationRespositoryName = "FeatureMigrationRespository"
)

const (
	// AddToPendingQ - add item to pending queue
	AddToPendingQ = true
	// DontAddToPendingQ - do not add to pending queue
	DontAddToPendingQ = false
)

// Properties in tables
const (
	// Properties in the IPReservationsTable
	reservationConfigProperty = "ReservationConfiguration"
	subnetKeyProperty         = "SubnetKey"
)

// TableToColumnMap - Table name to column map
var TableToColumnMap = map[string]string{
	subnetConfigurationTableName:                     subnetConfigurationTag,
	nodeTableName:                                    nodeConfigurationTag,
	ipReservationsTableName:                          reservationConfigurationTag,
	pendingWorkItemsTableName:                        requestContextTag,
	networkConfigurationTableName:                    networkConfigurationTag,
	networkContainerTableName:                        networkContainerConfigurationTag,
	tagDefinitionTableName:                           tagDefinitionsTag,
	ipAllocationsTableName:                           ipAllocationTag,
	sharedSubnetDelegationMigrationTableName:         sharedSubnetDelegationMigrationTag,
	sharedSubnetDelegationMigrationRegionalTableName: sharedSubnetDelegationMigrationTag,
	MasterElectionTableName:                          masterElectionTag,
	ncPendingPublishContextTableName:                 ncPendingPublishContextTag,
	featureMigrationTableName:                        featureMigrationTag,
	pendingCleanupRequestTableName:                   pendingCleanupRequestTag}
