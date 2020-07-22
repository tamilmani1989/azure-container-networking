package cmd

import (
	"dnc/api"
	"dnc/client"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/spf13/cobra"
)

//UpsertCmd creates or updates a resource on DNC
func UpsertCmd() *cobra.Command {
	var upsertcmd = &cobra.Command{
		Use:   "upsert",
		Short: "Creates or updates a resource on DNC",
	}
	upsertcmd.AddCommand(upsertNetworkContainerCmd())
	return upsertcmd
}

func upsertNetworkContainerCmd() *cobra.Command {
	var (
		nodeID           string
		networkID        string
		ncID             string
		subnet           string
		ipConstraint     string
		nodeConstraint   string
		ifaceID          string
		containerType    string
		host2NC          bool
		NC2host          bool
		secondaryIPCount int
		ipsNotInUse      *[]string
	)

	var cmd = &cobra.Command{
		Use:   "ncv2",
		Short: "Creates or updates a network container with v2 request",
		Run: func(cmd *cobra.Command, args []string) {
			var toBeDeletedIPs []string
			if ipsNotInUse != nil {
				toBeDeletedIPs = *ipsNotInUse
			}
			ncReqV2 := api.NetworkContainerRequestV2{
				AllocationRequest: api.AllocationRequest{
					SubnetName:       subnet,
					IPConstraint:     ipConstraint,
					NodeConstraint:   nodeConstraint,
					SecondaryIPCount: secondaryIPCount,
				},
				AssociationInfo: api.AssociationInfo{
					NodeID:        nodeID,
					InterfaceID:   ifaceID,
					ContainerType: containerType,
				},
				AllowHostToNCCommunication: host2NC,
				AllowNCToHostCommunication: NC2host,
				EndpointPolicies:           []cns.NetworkContainerRequestPolicies{},
				ToBeDeletedIPIds:           toBeDeletedIPs,
			}

			err := DncClient.UpsertNcV2(networkID, ncID, ncReqV2)
			if err != nil {
				client.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	cmd.Flags().StringVar(&networkID, "network-id", "", "Network ID")
	cmd.Flags().StringVar(&ncID, "nc-id", "", "Network Container ID")
	cmd.Flags().StringVar(&subnet, "subnet", "", "subnet")
	cmd.Flags().StringVar(&ipConstraint, "ip-constraint", "", "IP constraint")
	cmd.Flags().StringVar(&nodeConstraint, "node-constraint", "", "Node constraint")
	cmd.Flags().StringVar(&ifaceID, "interface-id", "", "Interface ID")
	cmd.Flags().StringVar(&containerType, "container-type", "", "Container type")
	cmd.Flags().BoolVar(&host2NC, "host-to-nc", false, "Host to nc")
	cmd.Flags().BoolVar(&NC2host, "nc-to-host", false, "NC to host")
	cmd.Flags().IntVar(&secondaryIPCount, "secondary-ip-count", 0, "Desired number of secondary IPs")
	ipsNotInUse = cmd.Flags().StringSlice("ips-not-in-use", nil, "IPs to deallocate (comma separated)")
	cmd.MarkFlagRequired("node-id")
	cmd.MarkFlagRequired("network-id")
	cmd.MarkFlagRequired("nc-id")
	cmd.MarkFlagRequired("subnet")
	cmd.MarkFlagRequired("container-type")

	return cmd
}
