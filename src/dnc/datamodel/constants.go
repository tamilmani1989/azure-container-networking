package datamodel

const (
	subnetConfigurationTableName  = "SubnetConfigurationTable"
	nodeTableName                 = "NodeConfigurationTable"
	ipReservationsTableName       = "IPReservationsTable"
	pendingWorkItemsTableName     = "PendingWorkItemsTable"
	networkConfigurationTableName = "NetworkConfigurationTable"
	networkContainerTableName     = "NetworkContainerTable"
	networkKey                    = "Network_%s"
	subnetConfigurationKey        = "Network_%s_Subnet_%s"
	ipReservationKey              = "Network_%s_Reservation_%s"
	networkcontainerKey           = "NC_%s"
	nodeKey                       = "Node_%s"
	ncKey                         = "Network_%s_ID_%s"
)

const (
	subnetConfigurationTag           = "SubnetConfiguration"
	networkConfigurationTag          = "NetworkConfiguration"
	reservationConfigurationTag      = "ReservationConfiguration"
	nodeConfigurationTag             = "NodeConfiguration"
	requestContextTag                = "RequestContext"
	networkContainerConfigurationTag = "NetworkContainerConfiguration"
)

// TableToColumnMap - Table name to column map
var TableToColumnMap = map[string]string{
	subnetConfigurationTableName:  subnetConfigurationTag,
	nodeTableName:                 nodeConfigurationTag,
	ipReservationsTableName:       reservationConfigurationTag,
	pendingWorkItemsTableName:     requestContextTag,
	networkConfigurationTableName: networkConfigurationTag,
	networkContainerTableName:     networkContainerConfigurationTag}
