package server

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Index,
	},
	Route{
		"CreateOrUpdateNode",
		"POST",
		"/nodes/{nodeID}",
		CreateOrUpdateNode,
	},
	Route{
		"DeleteNode",
		"DELETE",
		"/nodes/{nodeID}",
		DeleteNode,
	},
	Route{
		"GetNode",
		"GET",
		"/nodes/{nodeID}",
		GetNode,
	},
	Route{
		"CreateNetwork",
		"POST",
		"/networks/{networkID}",
		CreateNetwork,
	},
	Route{
		"GetNetworkStatus",
		"GET",
		"/networks/{networkID}/status",
		GetNetworkStatus,
	},
	Route{
		"GetNetwork",
		"GET",
		"/networks/{networkID}",
		GetNetwork,
	},
	Route{
		"ReserveIP",
		"POST",
		"/networks/{networkID}/reservations/{reservationID}",
		ReserveIP,
	},
	Route{
		"CreateNetworkContainer",
		"POST",
		"/networks/{networkID}/networkcontainer/{networkContainerID}",
		CreateNetworkContainer,
	},
	Route{
		"CreateNetworkContainer",
		"POST",
		"/networks/{networkID}/networkcontainers/{networkContainerID}",
		CreateNetworkContainer,
	},
	Route{
		"GetNetworkContainer",
		"GET",
		"/networks/{networkID}/networkcontainer/{networkContainerID}",
		GetNetworkContainer,
	},
	Route{
		"GetNetworkContainerStatus",
		"GET",
		"/networks/{networkID}/networkcontainer/{networkContainerID}/status",
		GetNetworkContainerStatus,
	},
	Route{
		"GetNetworkContainerStatus",
		"GET",
		"/networks/{networkID}/networkcontainers/{networkContainerID}/status",
		GetNetworkContainerStatus,
	},

	// Returns the NC entry in the NC table
	Route{
		"GetNetworkContainerConfiguration",
		"GET",
		"/networks/{networkID}/networkcontainer/{networkContainerID}/configuration",
		GetNetworkContainerConfiguration,
	},
	// Route{
	// 	"Reservations",
	// 	"GET",
	// 	"/networks/{networkID}",
	// 	ReservationsBynetworkID,
	// },

	Route{
		"GetReservation",
		"GET",
		"/networks/{networkID}/reservations/{reservationID}",
		GetReservation,
	},
	Route{
		"GetReservationStatus",
		"GET",
		"/networks/{networkID}/reservations/{reservationID}/status",
		GetReservationStatus,
	},
	Route{
		"DeleteReservation",
		"DELETE",
		"/networks/{networkID}/reservations/{reservationID}",
		DeleteReservation,
	},
	Route{
		"DeleteNetworkContainer",
		"DELETE",
		"/networks/{networkID}/networkcontainer/{networkContainerID}",
		DeleteNetworkContainer,
	},
	Route{
		"DeleteNetworkContainer",
		"DELETE",
		"/networks/{networkID}/networkcontainers/{networkContainerID}",
		DeleteNetworkContainer,
	},
	Route{
		"PutSubnet",
		"POST",
		"/networks/{networkID}/subnets/{subnet}",
		PutSubnet,
	},
	Route{
		"GetSubnetStatus",
		"GET",
		"/networks/{networkID}/subnets/{subnet}/status",
		GetSubnetStatus,
	},
	Route{
		"GetSubnetConfiguration",
		"GET",
		"/networks/{networkID}/subnets/{subnet}",
		GetSubnetConfiguration,
	},
	Route{
		"DeleteSubnet",
		"DELETE",
		"/networks/{networkID}/subnets/{subnet}",
		PutSubnet,
	},
	Route{
		"DeleteNetwork",
		"DELETE",
		"/networks/{networkID}",
		DeleteNetwork,
	},
	Route{
		"InitializeAzureStorage",
		"POST",
		"/azurestoragesettings",
		InitializeAzureStorage,
	},
	Route{
		"InitializeAzureStorageV2",
		"POST",
		"/azurestoragesettingsv2",
		InitializeAzureStorageV2,
	},
	Route{
		"InitializeAzureStorageV3",
		"POST",
		"/azurestoragesettingsv3/blob/{blob}/container/{container}",
		InitializeAzureStorageV3,
	},
	Route{
		"RelinquishPrimaryStatus",
		"POST",
		"/relinquishprimarystatus",
		RelinquishPrimaryStatus,
	},
	Route{
		"RefreshMonitorStatus",
		"GET",
		"/refreshmonitorstatus",
		RefreshMonitorStatus,
	},
	Route{
		"PromoteNetworkContainerState",
		"POST",
		"/promotenetworkcontainerstate/networks/{networkID}/networkcontainer/{networkContainerID}",
		PromoteContainerState,
	},
	Route{
		"SkipNodeCleanupForNetworkContainer",
		"POST",
		"/skipnodecleanupfornetworkcontainer/networks/{networkID}/networkcontainer/{networkContainerID}",
		SkipNodeCleanupForNetworkContainer,
	},
	Route{
		"IsWaitingForNodeCleanup",
		"GET",
		"/iswaitingfornodecleanup/networks/{networkID}/networkcontainer/{networkContainerID}",
		IsWaitingForNodeCleanup,
	},
	Route{
		"RefreshNMAgentGoalState",
		"POST",
		"/refreshnmagentgoalstate",
		RefreshNMAgentGoalState,
	},
	Route{
		"DumpStorageState",
		"POST",
		"/dumpstoragestate",
		DumpStorageState,
	},
}
