package cmd

import (
	"context"
	"fmt"
	"time"

	dncclient "dnc/client"
	syncdncclient "dnc/requestcontroller/dncclient"
	"dnc/requestcontroller/json"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	pingDNCUse       = "ping-dnc"
	pingDNCShortDesc = "Pings DNC to evaluate RC-to-DNC health"
	pingDNCLongDesc  = "Pings DNC by calling GetVnets endpoint on DNC to evaluate RC-to-DNC health"
)

var (
	pingDNCCommand = &cobra.Command{
		Use:   pingDNCUse,
		Short: pingDNCShortDesc,
		Long:  pingDNCLongDesc,
		Run: func(cmd *cobra.Command, args []string) {
			// If using this command as LivenessProbe in k8s, logs will go to kubelet logs
			configFile, err := cmd.Flags().GetString(configFlag)
			if err != nil {
				logrus.Fatal(err, "[dnc-rc] Error getting config flag value")
			}

			err = pingDNC(configFile)
			if err != nil {
				logrus.Fatalf("[dnc-rc] Error pinging DNC: %v", err)
			}

			logrus.Info("[dnc-rc] Healthy")
		},
	}
)

func pingDNC(configFile string) error {
	// Read the config
	jsonConfigProvider := json.ConfigProvider{
		ConfigFile: configFile,
	}

	config, err := jsonConfigProvider.GetConfig()
	if err != nil {
		return fmt.Errorf("[dnc-rc] Error reading config: %w", err)
	}

	logrus.Infof("[dnc-rc] DNC Client Config: %+v", config.DNCClientConfig)

	// Create the DNC client
	dncClientPollingInterval, err := time.ParseDuration(config.DNCClientConfig.PollingInterval)
	if err != nil {
		return fmt.Errorf("[dnc-rc] Could not parse config.DNCClientConfig.PollingInterval (%s) as duration: %w", config.DNCClientConfig.PollingInterval, err)
	}

	syncDncClient := syncdncclient.NewSynchronousDNCClient(dncclient.NewDncClient(config.DNCClientConfig.DNCEndpoint), dncClientPollingInterval, logrus.StandardLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// TODO: Use DNC health api instead of get VNETs call in future when health API is available
	_, err = syncDncClient.GetVnets(ctx)

	return err
}
