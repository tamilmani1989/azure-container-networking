package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func UnregisterCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "register",
		Short: "Registers an object with DNC",
	}
	cmd.AddCommand(registerNodeCmd())

	cmd.AddCommand(unregisterVnetCmd())
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
				fmt.Println(err)
			}
		},
	}
	cmd.Flags().String("id", "", "The VNET ID for delegation")
	cmd.MarkFlagRequired("id")

	return cmd
}
