package datamodel

const (
	subnetConfigurationTableName  = "SubnetConfigurationTable"
	nodeTableName                 = "NodeConfigurationTable"
	ipReservationsTableName       = "IPReservationsTable"
	pendingWorkItemsTableName     = "PendingWorkItemsTable"
	networkConfigurationTableName = "NetworkConfigurationTable"
	networkContainerTableName     = "NetworkContainerTable"
	tagDefintitionTableName       = "TagDefinitionsTable"
	networkKey                    = "Network_%s"
	subnetConfigurationKey        = "Network_%s_Subnet_%s"
	ipReservationKey              = "Network_%s_Reservation_%s"
	networkcontainerKey           = "NC_%s"
	nodeKey                       = "Node_%s"
	ncKey                         = "Network_%s_ID_%s"
	tagDefintionKey               = "Tag_%s_%s"
)

const (
	subnetConfigurationTag           = "SubnetConfiguration"
	networkConfigurationTag          = "NetworkConfiguration"
	reservationConfigurationTag      = "ReservationConfiguration"
	nodeConfigurationTag             = "NodeConfiguration"
	requestContextTag                = "RequestContext"
	networkContainerConfigurationTag = "NetworkContainerConfiguration"
	tagDefintionsTag                 = "TagDefintions"
)

const (
	// NodeRepositoryName - Repository name for node.
	NodeRepositoryName = "NodeRepository"

	// NetworkRepositoryName - Repository name for network.
	NetworkRepositoryName = "NetworkRepository"

	// TagRepositoryName - Repository name for tag defintions
	TagRepositoryName = "TagRepository"
)

// TableToColumnMap - Table name to column map
var TableToColumnMap = map[string]string{
	subnetConfigurationTableName:  subnetConfigurationTag,
	nodeTableName:                 nodeConfigurationTag,
	ipReservationsTableName:       reservationConfigurationTag,
	pendingWorkItemsTableName:     requestContextTag,
	networkConfigurationTableName: networkConfigurationTag,
	networkContainerTableName:     networkContainerConfigurationTag,
	tagDefintitionTableName:       tagDefintionsTag}
