package cmd

import (
	"encoding/json"
	"io/ioutil"

	"dnc/dnctl/dncclient"

	"github.com/spf13/cobra"
)

var DncClient *dncclient.DncClient

var ClientConfigPath = ".dncclientconfig.json"

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			file, _ := ioutil.ReadFile(ClientConfigPath)
			var dncClient dncclient.DncClient
			_ = json.Unmarshal([]byte(file), &dncClient)
			DncClient = dncclient.NewDncClient(dncClient.DncEP, dncClient.Blob, dncClient.Container, dncClient.PartitionKey)
		},
	}
	return rootCmd
}
