package api

import (
	"fmt"
)

// IPAllocationData - Struct to hold IP Allocation data
type IPAllocationData struct {
	IPAddress                    string `json:"IPAddress"`
	AssociatedNetworkContainerID string `json:"AssociatedNetworkContainerID"`
	ReservationID                string `json:"ReservationID"`
}

func (ipAllocationData *IPAllocationData) String() string {
	return fmt.Sprintf(
		"IPAllocationData:[IPAddress:%s, AssociatedNetworkContainerID:%s, ReservationID:%s]",
		ipAllocationData.IPAddress, ipAllocationData.AssociatedNetworkContainerID, ipAllocationData.ReservationID)
}

// IPAllocationsData - Array of IP Allocation data
type IPAllocationsData []IPAllocationData

func (ipAllocationsData IPAllocationsData) String() string {
	ipAllocationsDataString := "IPAllocationsData:["
	for _, ipAllocationData := range ipAllocationsData {
		ipAllocationsDataString += ipAllocationData.String()
	}
	ipAllocationsDataString += "]"

	return ipAllocationsDataString
}
