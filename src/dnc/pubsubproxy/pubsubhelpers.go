// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/api"
	"dnc/datamodel"
	"fmt"
	"strings"
)

const (
	// AzNM related pubsub paths
	containerContextSelectorFmt         = "/containers/%s"
	networkinterfacesContextSelectorFmt = "/networkinterfaces/%s"
	compartmentsConfigPathFmt           = "/compartments/%s/config"
	configPath                          = "/config"
)

func getAzContainerPublishingContextSelector(containerId string) string {
	var splits []string
	splits = strings.Split(containerId, api.ContainerIdDelimiter)
	return fmt.Sprintf(containerContextSelectorFmt, splits[len(splits)-1])
}

func getAzContainerPublishingPath() string {
	return configPath
}

func getNetworkInterfaceContextSelector(interfaceId string) string {
	return fmt.Sprintf(networkinterfacesContextSelectorFmt, interfaceId)
}

func getNetworkInterfaceConfigurationPublishingPath() string {
	return configPath
}

func getNetworkInterfaceCompartmentPublishingPath(interfaceId string) string {
	return fmt.Sprintf(compartmentsConfigPathFmt, interfaceId)
}

func isValidPubSubJob(operationType datamodel.NcOperationType) bool {
	switch operationType {
	case datamodel.NetworkContainerPublishRequest:
	case datamodel.NetworkContainerUnpublishRequest:
	case datamodel.BareMetalNetworkContainerPublishRequest:
	case datamodel.BareMetalNetworkContainerUnpublishRequest:
	default:
		return false
	}

	return true
}

func getJobType(operationType datamodel.NcOperationType) (jobType, *api.JsonErr) {
	switch operationType {
	case datamodel.NetworkContainerPublishRequest:
		return NetworkContainerPublish, nil
	case datamodel.NetworkContainerUnpublishRequest:
		return NetworkContainerUnpublish, nil
	case datamodel.BareMetalNetworkContainerPublishRequest:
		return BareMetalContainerPublish, nil
	case datamodel.BareMetalNetworkContainerUnpublishRequest:
		return BareMetalContainerUnpublish, nil
	default:
		return InvalidJobType, api.BadRequestf(
			api.InvalidPubSubOperationType,
			"Invalid Operation specificed %s",
			operationType.String())
	}
}
