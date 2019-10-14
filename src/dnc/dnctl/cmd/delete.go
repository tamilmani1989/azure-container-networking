package cmd

import (
	"dnc/dnctl/common"

	"github.com/spf13/cobra"
)

func DeleteCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "delete",
		Short: "Adds an item to DNC",
	}
	cmd.AddCommand(deleteNetworkContainerCmd())
	cmd.Aliases = []string{"del"}

	return cmd
}

func deleteNetworkContainerCmd() *cobra.Command {

	var (
		vnetId string
		ncId   string
	)

	var cmd = &cobra.Command{
		Use:   "nc",
		Short: "Creates a network container",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.DeleteNC(vnetId, ncId)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().StringVar(&vnetId, "vnet-id", "", "Node ID")
	cmd.Flags().StringVar(&ncId, "nc-id", "", "Network Container ID")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("nc-id")
	return cmd
}
