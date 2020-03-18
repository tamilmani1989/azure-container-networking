package cmd

import (
	"dnc/dnctl/common"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

// ConfigCmd can manipulate current DNC config
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
			DncClient.DncEP = endpoint
			b, _ := json.MarshalIndent(DncClient, "", "  ")
			_ = ioutil.WriteFile(ClientConfigPath, b, 0644)
			fmt.Print(string(b))
		},
	}
	cmd.Flags().String("endpoint", "", "Set your name")
	cmd.MarkFlagRequired("endpoint")

	return cmd
}

func configViewCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "view",
		Short: "Views the current DNC client config on disk",
		Run: func(cmd *cobra.Command, args []string) {
			common.PrettyPrint(DncClient)
		},
	}
	return cmd
}
