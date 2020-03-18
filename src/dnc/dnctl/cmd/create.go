package cmd

import (
	"dnc/dnctl/common"

	"github.com/spf13/cobra"
)

// CreateCmd can adds a resource to DNC
func CreateCmd() *cobra.Command {
	var createcmd = &cobra.Command{
		Use:   "create",
		Short: "Adds a resource to DNC",
	}
	createcmd.AddCommand(createNetworkContainerCmd())
	createcmd.AddCommand(createReservationCmd())
	return createcmd
}

func createNetworkContainerCmd() *cobra.Command {
	var (
		nodeID         string
		networkID      string
		ncID           string
		reservationID  string
		subnet         string
		ipConstraint   string
		nodeConstraint string
		ifaceID        string
		containerType  string
		podName        string
		podNamespace   string
		host2NC        bool
		NC2host        bool
	)

	var cmd = &cobra.Command{
		Use:   "nc",
		Short: "Creates a network container",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.AddNc(nodeID, networkID, ncID, reservationID, subnet, ipConstraint, nodeConstraint, ifaceID, containerType, podName, podNamespace, host2NC, NC2host)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	cmd.Flags().StringVar(&networkID, "network-id", "", "Network ID")
	cmd.Flags().StringVar(&ncID, "nc-id", "", "Network Container ID")
	cmd.Flags().StringVar(&reservationID, "reservation-id", "", "Reservation ID")
	cmd.Flags().StringVar(&subnet, "subnet", "", "subnet")
	cmd.Flags().StringVar(&ipConstraint, "ip-constraint", "", "IP constraint")
	cmd.Flags().StringVar(&nodeConstraint, "node-constraint", "", "Node constraint")
	cmd.Flags().StringVar(&ifaceID, "interface-id", "", "Interface ID")
	cmd.Flags().StringVar(&containerType, "container-type", "", "Container type")
	cmd.Flags().StringVar(&podName, "pod-name", "", "Pod name")
	cmd.Flags().StringVar(&podNamespace, "pod-namespace", "", "Pod namespace")
	cmd.Flags().BoolVar(&host2NC, "host-to-nc", false, "Host to nc")
	cmd.Flags().BoolVar(&NC2host, "nc-to-host", false, "NC to host")
	cmd.MarkFlagRequired("node-id")
	cmd.MarkFlagRequired("network-id")
	cmd.MarkFlagRequired("nc-id")
	cmd.MarkFlagRequired("subnet")
	cmd.MarkFlagRequired("interface-id")
	cmd.MarkFlagRequired("container-type")
	cmd.MarkFlagRequired("pod-name")
	cmd.MarkFlagRequired("pod-namespace")

	return cmd
}

func createReservationCmd() *cobra.Command {
	var (
		networkID      string
		subnet         string
		ipConstraint   string
		nodeConstraint string
		reservationID  string
	)

	var cmd = &cobra.Command{
		Use:   "reservation",
		Short: "Creates a reservation",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.AddReservation(networkID, subnet, ipConstraint, nodeConstraint, reservationID)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Aliases = []string{"res"}
	cmd.Flags().StringVar(&networkID, "network-id", "", "Network ID")
	cmd.Flags().StringVar(&subnet, "subnet", "", "Subnet Name")
	cmd.Flags().StringVar(&subnet, "ipConstraint", "", "ipConstraint")
	cmd.Flags().StringVar(&subnet, "nodeConstraint", "", "nodeConstraint")
	cmd.Flags().StringVar(&subnet, "reservation-id", "", "reservation-id")
	cmd.MarkFlagRequired("network-id")
	cmd.MarkFlagRequired("reservation-id")

	return cmd
}
