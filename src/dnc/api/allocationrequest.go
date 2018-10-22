package api

import (
	"dnc/utils"
	"fmt"
	"net"
	"strings"
)

// AllocationRequest - Struct to hold allocation request.
type AllocationRequest struct {
	SubnetName     string `json:"SubnetName"`
	IPConstraint   string `json:"IPConstraint"`
	NodeConstraint string `json:"NodeConstraint"`
}

func (allocationRequest *AllocationRequest) String() string {
	return fmt.Sprintf(
		"AllocationRequest:[IPSubnet:{%s} IPConstraint:{%s} NodeConstraint:{%s}]",
		allocationRequest.SubnetName, allocationRequest.IPConstraint, allocationRequest.NodeConstraint)
}

// Validate - Validates the allocation request
func (allocationRequest *AllocationRequest) Validate() *JsonErr {
	if len(allocationRequest.SubnetName) == 0 {
		return &JsonErr{Code: InvalidIPSubnet, Text: "SubnetName cannot be empty"}
	}

	allocationRequest.SubnetName = strings.ToLower(allocationRequest.SubnetName)

	if len(allocationRequest.IPConstraint) != 0 {
		isIPConstraintASubnet := strings.Contains(allocationRequest.IPConstraint, utils.PrefixDelimiter)
		if !isIPConstraintASubnet {
			ip := net.ParseIP(allocationRequest.IPConstraint)
			if ip == nil {
				return &JsonErr{
					Code: InvalidIPConstraint,
					Text: fmt.Sprintf("Failed to parse IPConstraint[%s]", allocationRequest.IPConstraint)}
			}
		} else {
			_, _, err := net.ParseCIDR(allocationRequest.IPConstraint)
			if err == nil {
				return &JsonErr{
					Code: InvalidIPConstraint,
					Text: fmt.Sprintf("Failed to parse IPConstraint[%s], Error:[%s]", allocationRequest.IPConstraint, err.Error())}
			}
		}
	}

	return nil
}
