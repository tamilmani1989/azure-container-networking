// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package main

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cni/cniv6"
	"github.com/Azure/azure-container-networking/cni/network"
	"github.com/Azure/azure-container-networking/common"
	acn "github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"
)

const (
	hostNetAgentURL                 = "http://168.63.129.16/machine/plugins?comp=netagent&type=cnireport"
	ipamQueryURL                    = "http://168.63.129.16/machine/plugins?comp=nmagent&type=getinterfaceinfov1"
	pluginName                      = "CNIv6"
	telemetryNumRetries             = 5
	telemetryWaitTimeInMilliseconds = 200
	name                            = "azure-vnet"
	logFile                         = name + "v6"
)

// Version is populated by make during build.
var version string

// Command line arguments for CNI plugin.
var args = acn.ArgumentList{
	{
		Name:         acn.OptVersion,
		Shorthand:    acn.OptVersionAlias,
		Description:  "Print version information",
		Type:         "bool",
		DefaultValue: false,
	},
}

// Prints version information.
func printVersion() {
	fmt.Printf("Azure CNI Version %v\n", version)
}

// send error report to hostnetagent if CNI encounters any error.
func reportPluginError(reportManager *telemetry.ReportManager, tb *telemetry.TelemetryBuffer, err error) {
	log.Printf("Report plugin error")
	reflect.ValueOf(reportManager.Report).Elem().FieldByName("ErrorMessage").SetString(err.Error())

	if err := reportManager.SendReport(tb); err != nil {
		log.Errorf("SendReport failed due to %v", err)
	}
}

// Main is the entry point for CNI network plugin.
func main() {

	startTime := time.Now()

	// Initialize and parse command line arguments.
	acn.ParseArgs(&args, printVersion)
	vers := acn.GetArg(acn.OptVersion).(bool)

	if vers {
		printVersion()
		os.Exit(0)
	}

	var (
		config       common.PluginConfig
		err          error
		cnimetric    telemetry.AIMetric
		logDirectory string // This sets empty string i.e. current location
	)

	log.SetName(logFile)
	log.SetLevel(log.LevelInfo)
	if err = log.SetTargetLogDirectory(log.TargetLogfile, logDirectory); err != nil {
		fmt.Printf("Failed to setup cni logging: %v\n", err)
		return
	}

	defer log.Close()

	config.Version = version
	reportManager := &telemetry.ReportManager{
		HostNetAgentURL: hostNetAgentURL,
		ContentType:     telemetry.ContentType,
		Report: &telemetry.CNIReport{
			Context:          "AzureCNIV6",
			SystemDetails:    telemetry.SystemInfo{},
			InterfaceDetails: telemetry.InterfaceInfo{},
			BridgeDetails:    telemetry.BridgeInfo{},
		},
	}

	cniReport := reportManager.Report.(*telemetry.CNIReport)

	upTime, err := platform.GetLastRebootTime()
	if err == nil {
		cniReport.VMUptime = upTime.Format("2006-01-02 15:04:05")
	}

	cniReport.GetReport(pluginName, version, ipamQueryURL)

	cniV6Plugin, err := cniv6.NewPlugin(name, &config)
	if err != nil {
		fmt.Printf("Failed to create IPAM plugin, err:%v.\n", err)
		os.Exit(1)
	}

	// CNI Acquires lock
	if err = cniV6Plugin.Plugin.InitializeKeyValueStore(&config); err != nil {
		log.Errorf("Failed to initialize key-value store of network plugin, err:%v.\n", err)
		tb := telemetry.NewTelemetryBuffer("")
		if tberr := tb.Connect(); tberr == nil {
			reportPluginError(reportManager, tb, err)
			tb.Close()
		}

		if isSafe, _ := cniV6Plugin.Plugin.IsSafeToRemoveLock(name); isSafe {
			log.Printf("[CNI] Removing lock file as process holding lock exited")
			if errUninit := cniV6Plugin.Plugin.UninitializeKeyValueStore(true); errUninit != nil {
				log.Errorf("Failed to uninitialize key-value store of network plugin, err:%v.\n", errUninit)
			}
		}

		return
	}

	defer func() {
		if errUninit := cniV6Plugin.Plugin.UninitializeKeyValueStore(false); errUninit != nil {
			log.Errorf("Failed to uninitialize key-value store of network plugin, err:%v.\n", errUninit)
		}

		if recover() != nil {
			return
		}
	}()

	// Start telemetry process if not already started. This should be done inside lock, otherwise multiple process
	// end up creating/killing telemetry process results in undesired state.
	tb := telemetry.NewTelemetryBuffer("")
	tb.ConnectToTelemetryService(telemetryNumRetries, telemetryWaitTimeInMilliseconds)
	defer tb.Close()

	cniV6Plugin.SetCNIReport(cniReport, tb)

	t := time.Now()
	cniReport.Timestamp = t.Format("2006-01-02 15:04:05")

	if err = cniV6Plugin.Start(&config); err != nil {
		log.Errorf("Failed to start network plugin, err:%v.\n", err)
		reportPluginError(reportManager, tb, err)
		panic("network plugin start fatal error")
	}

	if err = cniV6Plugin.Execute(cni.PluginApi(cniV6Plugin)); err != nil {
		log.Errorf("Failed to execute network plugin, err:%v.\n", err)
	}

	cniV6Plugin.Stop()

	// release cni lock
	if errUninit := cniV6Plugin.Plugin.UninitializeKeyValueStore(false); errUninit != nil {
		log.Errorf("Failed to uninitialize key-value store of network plugin, err:%v.\n", errUninit)
	}

	executionTimeMs := time.Since(startTime).Milliseconds()
	cnimetric.Metric = aitelemetry.Metric{
		Name:             telemetry.CNIV6ExecutimeMetricStr,
		Value:            float64(executionTimeMs),
		CustomDimensions: make(map[string]string),
	}
	network.SetCustomDimensions(&cnimetric, nil, err)
	telemetry.SendCNIMetric(&cnimetric, tb)

	if err != nil {
		reportPluginError(reportManager, tb, err)
		panic("network plugin execute fatal error")
	}

	// Report CNI successfully finished execution.
	reflect.ValueOf(reportManager.Report).Elem().FieldByName("CniSucceeded").SetBool(true)
	reflect.ValueOf(reportManager.Report).Elem().FieldByName("OperationDuration").SetInt(executionTimeMs)

	if err = reportManager.SendReport(tb); err != nil {
		log.Errorf("SendReport failed due to %v", err)
	} else {
		log.Printf("Sending report succeeded")
	}
}
