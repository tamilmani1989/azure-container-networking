package main

import (
	"flag"
	"fmt"
	"os"
	"servicetunnel/configuration"
	"time"

	"github.com/Azure/azure-container-networking/log"
)

const (
	DEFAULT_TIMEOUT_IN_SECS = 30
	DEFAULT_MAX_RETRIES     = 20
	DEFAULT_LOGDIR          = "/var/log"
	enable                  = "enable"
	disable                 = "disable"
	configFileName          = "stconfig.json"
)

var version string

// Prints description and version information.
func printVersion() {
	fmt.Printf("Azure Linux ServiceTunnel\n")
	fmt.Printf("Version %v\n", version)
}

func main() {
	var err error
	var mode *string

	mode = flag.String("m", "N/A", "mode")
	timeout := flag.Int("t", DEFAULT_TIMEOUT_IN_SECS, "retry timeout")
	maxRetries := flag.Int("rc", DEFAULT_MAX_RETRIES, "max retry count")
	logDir := flag.String("l", DEFAULT_LOGDIR, "logdir")
	vers := flag.Bool("v", false, "-v")
	flag.Parse()

	if *vers {
		printVersion()
		os.Exit(0)
	}

	log.SetName("servicetunnel")
	log.SetLevel(log.LevelInfo)
	if err = log.SetTargetLogDirectory(log.TargetLogfile, *logDir); err == nil {
		fmt.Printf("LogDir configuration succeeded for dir %v\n", logDir)
	}

	defer log.Close()

	log.Printf("options passed : mode %v, timeout %v, maxRetries %v", *mode, *timeout, *maxRetries)

	serviceTunnelConfig, err := configuration.ReadConfigFile(configFileName)
	if err != nil {
		log.Errorf("Error reading servicetunnel config: %v", err)
	}
	configuration.SetConfigDefaults(&serviceTunnelConfig)

	log.Printf("servicetunnel config %+v", serviceTunnelConfig)

	switch *mode {
	case enable:
		if CheckIfSTBridgeExists(serviceTunnelConfig.BridgeName) {
			log.Printf("ServiceTunnelBridge exists. So exiting...")
			os.Exit(0)
		}

		i := 0
		for ; i < *maxRetries; i++ {
			if err = CreateServiceTunnel(serviceTunnelConfig); err == nil {
				break
			}

			log.Printf("CreateServiceTunnel Failed. Lets remove and create it again.Attempt:%d", i)
			log.Printf("Going to sleep for %v seconds before calling Remove and createservicetunnel", *timeout)
			time.Sleep(time.Duration(*timeout) * time.Second)

			RemoveServiceTunnel(serviceTunnelConfig)
		}

		if i < *maxRetries {
			log.Printf("Service Tunnel setup completed successfully!")
		} else {
			log.Printf("Service Tunnel setup failed!")
			os.Exit(3)
		}

		break
	case disable:
		RemoveServiceTunnel(serviceTunnelConfig)
		break
	default:
		log.Printf("Unknown option mode %v", *mode)
	}

	os.Exit(0)
}
