package nmagentcontracts

const (
	// JoinNetworkURLFmt - URL format to join a network.
	JoinNetworkURLFmt = "http://169.254.169.254/machine/plugins/?comp=nmagent&type=NetworkManagement/joinedVirtualNetworks/%s/api-version/1"

	// GetNetworkConfigurationURLFmt - URL format to get network configuration.
	GetNetworkConfigurationURLFmt = "http://169.254.169.254/machine/plugins/?comp=nmagent&type=NetworkManagement/joinedVirtualNetworks/%s/api-version/1"

	// PutNetworkContainerURLFmt - URL format to create a network container.
	PutNetworkContainerURLFmt = "http://169.254.169.254/machine/plugins/?comp=nmagent&type=NetworkManagement/interfaces/%s/networkContainers/%s/authenticationToken/%s/api-version/1"

	// DeleteNetworkContainerURLFmt - URL format to delete a network container.
	DeleteNetworkContainerURLFmt = "http://169.254.169.254/machine/plugins/?comp=nmagent&type=NetworkManagement/interfaces/%s/networkContainers/%s/api-version/1/method/DELETE"
)
