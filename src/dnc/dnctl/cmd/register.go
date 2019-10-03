package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterCmd() *cobra.Command {
	var registercmd = &cobra.Command{
		Use:   "register",
		Short: "Registers an object with DNC",
	}
	registercmd.AddCommand(registerNodeCmd())
	registercmd.AddCommand(registerVnetCmd())
	registercmd.AddCommand(registerNetworkProfileCmd())
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
			fmt.Println(err)
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
			fmt.Printf("Inside delete Run with args: %v\n", args)
			flagname, _ := cmd.Flags().GetString("name")
			fmt.Printf("flags %+v\n", flagname)
		},
	}
	cmd.Flags().StringP("name", "n", "", "Set your name")
	return cmd
}

func registerNetworkProfileCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "networkprofile",
		Short: "Registers a Network Profile",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Inside delete Run with args: %v\n", args)
			flagname, _ := cmd.Flags().GetString("name")
			fmt.Printf("flags %+v\n", flagname)
		},
	}
	cmd.Aliases = []string{"netprof"}
	cmd.Flags().StringP("name", "n", "", "Set your name")
	return cmd
}
