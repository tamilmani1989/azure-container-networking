package cmd

import (
	"dnc/dnctl/common"

	"github.com/spf13/cobra"
)

func CreateCmd() *cobra.Command {
	var createcmd = &cobra.Command{
		Use:   "create",
		Short: "Adds an item to DNC",
	}
	createcmd.AddCommand(createNetworkContainerCmd())
	return createcmd
}

func createNetworkContainerCmd() *cobra.Command {

	var (
		nodeId         string
		networkId      string
		ncId           string
		reservationId  string
		subnet         string
		ipConstraint   string
		nodeConstraint string
		ifaceId        string
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

			err := DncClient.AddNc(nodeId, networkId, ncId, reservationId, subnet, ipConstraint, nodeConstraint, ifaceId, containerType, podName, podNamespace, host2NC, NC2host)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().StringVar(&nodeId, "node-id", "", "Node ID")
	cmd.Flags().StringVar(&networkId, "network-id", "", "Network ID")
	cmd.Flags().StringVar(&ncId, "nc-id", "", "Network Container ID")
	cmd.Flags().StringVar(&reservationId, "reservation-id", "", "Reservation ID")
	cmd.Flags().StringVar(&subnet, "subnet", "", "subnet")
	cmd.Flags().StringVar(&ipConstraint, "ip-constraint", "", "IP constraint")
	cmd.Flags().StringVar(&nodeConstraint, "node-constraint", "", "Node constraint")
	cmd.Flags().StringVar(&ifaceId, "interface-id", "", "Interface ID")
	cmd.Flags().StringVar(&containerType, "container-type", "", "Container type")
	cmd.Flags().StringVar(&podName, "pod-name", "", "Pod name")
	cmd.Flags().StringVar(&podNamespace, "pod-namespace", "", "Pod namespace")
	cmd.Flags().BoolVar(&host2NC, "host-to-nc", false, "Host to nc")
	cmd.Flags().BoolVar(&NC2host, "nc-to-host", false, "NC to host")

	return cmd
}
