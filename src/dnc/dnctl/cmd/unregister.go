package cmd

import (
	"dnc/dnctl/common"

	"github.com/spf13/cobra"
)

func UnregisterCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "unregister",
		Short: "Unregisters an object with DNC",
	}
	cmd.AddCommand(unregisterNodeCmd())
	cmd.AddCommand(unregisterVnetCmd())
	cmd.AddCommand(unregisterSubnetCmd())
	cmd.Aliases = []string{"unreg"}
	return cmd
}

func unregisterVnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "vnet",
		Short: "Unregisters a VNET",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("id")

			err := DncClient.DeleteNetwork(vnetid)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("id", "", "The VNET ID for delegation")
	cmd.MarkFlagRequired("id")

	return cmd
}

func unregisterNodeCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "node",
		Short: "Unregisters a node",
		Run: func(cmd *cobra.Command, args []string) {
			nodeid, _ := cmd.Flags().GetString("id")

			err := DncClient.DeleteNode(nodeid)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("id", "", "The registered node id")
	cmd.MarkFlagRequired("id")

	return cmd
}

func unregisterSubnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "subnet",
		Short: "Unregisters a subnet",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			subnetid, _ := cmd.Flags().GetString("vnet-id")

			err := DncClient.DeleteSubnet(vnetid, subnetid)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("vnet-id", "", "Vnet ID")
	cmd.Flags().String("subnet-id", "", "Subnet ID")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("subnet-id")

	return cmd
}
