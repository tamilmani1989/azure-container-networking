package cmd

import (
	"dnc/dnctl/common"
	"net/http"

	"github.com/spf13/cobra"
)

// RegisterCmd can register an object
func RegisterCmd() *cobra.Command {
	var registercmd = &cobra.Command{
		Use:   "register",
		Short: "Registers an object with DNC",
	}
	registercmd.AddCommand(registerNodeCmd())
	registercmd.AddCommand(registerVnetCmd())
	registercmd.AddCommand(registerSubnet())
	registercmd.Aliases = []string{"reg"}
	return registercmd
}

func registerNodeCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "node",
		Short: "Registers a node",
		Run: func(cmd *cobra.Command, args []string) {
			nodeid, _ := cmd.Flags().GetString("id")
			ipaddr, _ := cmd.Flags().GetString("ip-address")
			orchtype, _ := cmd.Flags().GetString("orchestrator-type")

			err := DncClient.AddNode(nodeid, ipaddr, orchtype)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("id", "", "Name of the node")
	cmd.Flags().String("ip-address", "", "IP Address of the node")
	cmd.Flags().String("orchestrator-type", "", "The orchestrator type of the node")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("ip-address")
	cmd.MarkFlagRequired("orchestrator-type")

	return cmd
}

func registerVnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "vnet",
		Short: "Registers a VNET",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("id")

			err := DncClient.AddNetwork(vnetid)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("id", "", "The VNET ID for delegation")
	cmd.MarkFlagRequired("id")

	return cmd
}

func registerSubnet() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "subnet",
		Short: "Registers a subnet",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			subnetid, _ := cmd.Flags().GetString("subnet-id")
			authtoken, _ := cmd.Flags().GetString("auth-token")

			err := DncClient.AddSubnet(vnetid, subnetid, authtoken)
			if err != nil && err.HTTPStatus != http.StatusOK {
				common.PrettyPrint(err)
				return
			}
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("subnet-id", "", "subnet-id")
	cmd.Flags().String("auth-token", "", "auth-token")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("subnet-id")
	cmd.MarkFlagRequired("auth-id")
	return cmd
}
