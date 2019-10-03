package cmd

import (
	"fmt"

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
	var addNodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Adds a node to DNC",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Inside delete Run with args: %v\n", args)
			flagname, _ := cmd.Flags().GetString("name")
			fmt.Printf("flags %+v\n", flagname)
		},
	}
	addNodeCmd.Flags().StringP("name", "n", "", "Set your name")
	return addNodeCmd
}
