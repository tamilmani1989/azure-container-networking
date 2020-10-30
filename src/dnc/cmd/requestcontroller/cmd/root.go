package cmd

import (
	"dnc/requestcontroller/buildinfo"
	"dnc/requestcontroller/json"
	"dnc/requestcontroller/kubernetes"
	"dnc/utils"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

const (
	shortDesc  = "Watches K8s api-server for SWIFT-based CNI deployment"
	longDesc   = "Watches K8s api-server for Nodes, creates/deletes NodeNetworkConfigs, and responds to NodeNetworkConfig changes"
	configFlag = "config"
)

var (
	rootCmd = &cobra.Command{
		Use:   buildinfo.ExecutableName,
		Short: shortDesc,
		Long:  longDesc,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, err := cmd.Flags().GetString(configFlag)
			if err != nil {
				log.Fatal(err, "Error getting config flag value")
			}
			runRequestController(configFile)
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(utils.Version)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringP(configFlag, "c", "", "Configuration file")
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func runRequestController(configFile string) {
	// Create the config provider
	jsonConfigProvider := json.ConfigProvider{
		ConfigFile: configFile,
	}

	// Create the RequestController
	requestController, err := kubernetes.NewRequestControllerWithConfigProvider(&jsonConfigProvider)
	if err != nil {
		log.Fatalf("could not create request controller: %v", err)
	}

	// Start the RequestController
	requestController.Start() // blocks
}
