package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

func ConfigCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "config",
		Short: "Can manipulate the current DNC config",
	}
	cmd.AddCommand(configViewCmd())
	cmd.AddCommand(configSet())
	return cmd
}

func configSet() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "set",
		Short: "Set the current DNC client config on disk",
		Run: func(cmd *cobra.Command, args []string) {
			endpoint, _ := cmd.Flags().GetString("endpoint")
			blob, _ := cmd.Flags().GetString("blob")
			container, _ := cmd.Flags().GetString("container")
			partitionkey, _ := cmd.Flags().GetString("partition-key")
			DncClient.DncEP = endpoint
			DncClient.Blob = blob
			DncClient.Container = container
			DncClient.PartitionKey = partitionkey
			b, _ := json.MarshalIndent(DncClient, "", "  ")
			_ = ioutil.WriteFile(ClientConfigPath, b, 0644)
			fmt.Print(string(b))
		},
	}
	cmd.Flags().String("endpoint", "", "Set your name")
	cmd.Flags().String("blob", "", "Set your name")
	cmd.Flags().String("container", "", "Set your name")
	cmd.Flags().String("partition-key", "", "Set your name")
	cmd.MarkFlagRequired("endpoint")
	cmd.MarkFlagRequired("blob")
	cmd.MarkFlagRequired("container")
	cmd.MarkFlagRequired("partition-key")

	return cmd
}

func configViewCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "view",
		Short: "Views the current DNC client config on disk",
		Run: func(cmd *cobra.Command, args []string) {
			b, _ := json.MarshalIndent(DncClient, "", "  ")
			fmt.Print(string(b))
		},
	}
	return cmd
}
