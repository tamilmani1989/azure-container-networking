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
	networkKey                                       = "Network_%s"
	subnetConfigurationKey                           = "Network_%s_Subnet_%s"
	ipReservationKey                                 = "Network_%s_Reservation_%s"
	networkcontainerKey                              = "NC_%s"
	nodeKey                                          = "Node_%s"
	ncKey                                            = "Network_%s_ID_%s"
	tagDefinitionKey                                 = "Tag_%s_%s"
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
)

const (
	// NodeRepositoryName - Repository name for node.
	NodeRepositoryName = "NodeRepository"

	// NetworkRepositoryName - Repository name for network.
	NetworkRepositoryName = "NetworkRepository"

	// TagRepositoryName - Repository name for tag defintions
	TagRepositoryName = "TagRepository"
)

const (
	// AddToPendingQ - add item to pending queue
	AddToPendingQ = true
	// DontAddToPendingQ - do not add to pending queue
	DontAddToPendingQ = false
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
	sharedSubnetDelegationMigrationRegionalTableName: sharedSubnetDelegationMigrationTag}
