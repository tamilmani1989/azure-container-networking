package cmd

import (
	"encoding/json"
	"io/ioutil"

	"dnc/dnctl/dncclient"

	"github.com/spf13/cobra"
)

// DncClient for customer consume
var DncClient *dncclient.DncClient

// ClientConfigPath used
var ClientConfigPath = ".dncclientconfig.json"

// NewRootCmd returns a root
func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			file, _ := ioutil.ReadFile(ClientConfigPath)
			var dncClient dncclient.DncClient
			_ = json.Unmarshal([]byte(file), &dncClient)
			DncClient = dncclient.NewDncClient(dncClient.DncEP)
		},
	}
	return rootCmd
}
