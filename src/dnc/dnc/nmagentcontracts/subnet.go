package nmagentcontracts

import (
	"fmt"
)

// Subnet - Defines the subnet struct.
type Subnet struct {
	SubnetName     string `json:"name"`
	AddressPrefix  string `json:"addressPrefix"`
	Tags           []Tag  `json:"tags"`
	DefaultGateway string `json:"defaultGateway"`
}

func (subnet *Subnet) String() string {
	tagsString := "null"
	if subnet.Tags != nil {
		for index, tag := range subnet.Tags {
			if index > 0 && index < len(subnet.Tags)-1 {
				tagsString += ","
			}
			tagsString += tag.String()
		}
	}

	return fmt.Sprintf(
		"Subnet:[SubnetName:%s, AddressPrefix:%s, DefaultGateway:%s Tags:%s]",
		subnet.SubnetName,
		subnet.AddressPrefix,
		subnet.DefaultGateway,
		tagsString)
}
