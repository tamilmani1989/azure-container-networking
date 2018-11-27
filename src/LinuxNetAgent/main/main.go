package main

import (
	"LinuxNetAgent/ExtensionHandler"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-container-networking/log"
)

const (
	DEFAULT_TIMEOUT_IN_SECS = 10
	DEFAULT_MAX_RETRIES     = 5
	DEFAULT_LOGDIR          = "/var/log"
	enable                  = "enable"
	disable                 = "disable"
)

func main() {
	var err error
	var handlerEnv *extensionhandler.HandlerEnvironment
	var handlerConfig *extensionhandler.HandlerConfig
	var logConfig bool

	internalTimeout := 5

	mode := flag.String("mode", "N/A", "mode")
	timeout := flag.Int("timeout", DEFAULT_TIMEOUT_IN_SECS, "retry timeout")
	maxRetries := flag.Int("maxretries", DEFAULT_MAX_RETRIES, "max retry count")
	flag.Parse()

	for i := 0; i < *maxRetries; i++ {
		handlerEnv, err = extensionhandler.ReadHandlerEnvironment("HandlerEnvironment.json")
		if err == nil {
			break
		}

		time.Sleep(time.Duration(internalTimeout) * time.Second)
		fmt.Printf("Reading handlerenv failed with err %v. Retry reading it for %v times\n", err, i)
	}

	if handlerEnv == nil {
		fmt.Printf("Failed reading handler env file.exiting\n")
		os.Exit(1)
	}

	fmt.Printf("Log directory %v\n", handlerEnv.Env.LogFolder)

	for i := 0; i < *maxRetries; i++ {
		var err error
		log.SetLogDirectory(handlerEnv.Env.LogFolder)
		log.SetName("servicetunnel")
		log.SetLevel(log.LevelInfo)
		if err = log.SetTarget(log.TargetLogfile); err == nil {
			fmt.Printf("LogDir configuration succeeded for dir %v\n", handlerEnv.Env.LogFolder)
			logConfig = true
			break
		}

		fmt.Printf("Failed to configure logging, err:%v. Attempt:%v\n", err, i)
		time.Sleep(time.Duration(internalTimeout) * time.Second)
	}

	if !logConfig {
		log.SetLogDirectory(DEFAULT_LOGDIR)
		log.SetName("servicetunnel")
		log.SetLevel(log.LevelInfo)
		if err := log.SetTarget(log.TargetLogfile); err == nil {
			fmt.Printf("LogDir configuration succeeded for dir %v\n", DEFAULT_LOGDIR)
		}
	}

	log.Printf("options passed : mode %v, timeout %v, maxRetries %v", *mode, *timeout, *maxRetries)

	fileNames, err := handlerEnv.ListFilesUnderConfigFolder()
	if err != nil {
		log.Printf("ListFilesUnderConfigFolder failed with error %v", err)
		return
	}

	settingsFiles := extensionhandler.GetSettingsFiles(fileNames)
	log.Printf("setting files %v", settingsFiles)
	latestSetting := extensionhandler.FindLatestSettingNumber(settingsFiles)
	log.Printf("Latest setting %v", latestSetting)

	handlerConfig, err = handlerEnv.ReadHandlerConfig(latestSetting)
	if err != nil {
		log.Printf("ReadHandlerConfig failed with error %v", err)
		return
	}

	serviceTunnelConfig, err := handlerConfig.ReadServiceTunnelConfig()
	if serviceTunnelConfig != nil {
		log.Printf("servicetunnel config %+v", serviceTunnelConfig)
	}

	switch *mode {
	case enable:
		if err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Transistioning, *mode, extensionhandler.PassCode, "Started Enable Operation"); err != nil {
			log.Printf("write status log failed with error %+v", err)
		}

		if CheckIfSTBridgeExists() {
			log.Printf("ServiceTunnelBridge exists. So exiting...")
			os.Exit(0)
		}

		i := 0
		for ; i < *maxRetries; i++ {
			err := CreateServiceTunnel(serviceTunnelConfig)
			if err == nil {
				break
			}

			log.Printf("CreateServiceTunnel Failed. Lets remove and create it again.Attempt:%d", i)
			log.Printf("Going to sleep for %v seconds before calling Remove and createservicetunnel", *timeout)
			time.Sleep(time.Duration(*timeout) * time.Second)

			RemoveServiceTunnel(serviceTunnelConfig)

			if err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Warning, *mode, extensionhandler.WarningCode, "CreateServiceTunnel Failed. Retrying"); err != nil {
				log.Printf("write status log failed with error %+v", err)
			}
		}

		if i < *maxRetries {
			if err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Success, *mode, extensionhandler.PassCode, "Service Tunnel Enable Operation completed successfully!"); err != nil {
				log.Printf("write status log failed with error %+v", err)
			}

			log.Printf("Service Tunnel setup completed successfully!")

		} else {
			if err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Error, *mode, extensionhandler.FailCode, "Service Tunnel Enable Operation failed!"); err != nil {
				log.Printf("write status log failed with error %+v", err)
			}

			log.Printf("Service Tunnel setup failed!")
			os.Exit(2)
		}

		break

	case disable:
		if err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Transistioning, *mode, extensionhandler.PassCode, "Started Disable Operation"); err != nil {
			log.Printf("write status log failed with error %+v", err)
		}

		RemoveServiceTunnel(serviceTunnelConfig)

		err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Success, *mode, extensionhandler.PassCode, "Service Tunnel Disable Operation completed successfully!")
		if err != nil {
			log.Printf("write status log failed with error %+v", err)
		}

		break

	default:
		log.Printf("Unknown option mode %v", *mode)

		if err := handlerEnv.WriteStatusLog(latestSetting, extensionhandler.Error, *mode, extensionhandler.FailCode, "Unknown option mode!"); err != nil {
			log.Printf("write status log failed with error %+v", err)
		}

		os.Exit(3)
	}

	os.Exit(0)
}
