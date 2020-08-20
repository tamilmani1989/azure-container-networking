package networkcontainer

import "github.com/Azure/azure-container-networking/cns"

// ShouldUpdateVlan indicates whether a vlan should be updated in a node for a given container type
func ShouldUpdateVlan(containerType string) bool {
	switch containerType {
	case cns.AzureContainerInstance:
		fallthrough
	case cns.Docker:
		fallthrough
	case cns.JobObject:
		fallthrough
	case cns.COW:
		return true
	default:
		return false
	}
}
