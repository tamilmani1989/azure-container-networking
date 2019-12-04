// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cni/network"
	"github.com/Azure/azure-container-networking/common"
	acn "github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"
	"github.com/containernetworking/cni/pkg/skel"
)

const (
	hostNetAgentURL                 = "http://168.63.129.16/machine/plugins?comp=netagent&type=cnireport"
	ipamQueryURL                    = "http://168.63.129.16/machine/plugins?comp=nmagent&type=getinterfaceinfov1"
	pluginName                      = "CNI"
	telemetryNumRetries             = 5
	telemetryWaitTimeInMilliseconds = 200
	name                            = "azure-vnet"
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

func validateConfig(jsonBytes []byte) error {
	var conf struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(jsonBytes, &conf); err != nil {
		return fmt.Errorf("error reading network config: %s", err)
	}
	if conf.Name == "" {
		return fmt.Errorf("missing network name")
	}
	return nil
}

func getCmdArgsFromEnv() (string, *skel.CmdArgs, error) {
	log.Printf("Going to read from stdin")
	stdinData, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "", nil, fmt.Errorf("error reading from stdin: %v", err)
	}

	cmdArgs := &skel.CmdArgs{
		ContainerID: os.Getenv("CNI_CONTAINERID"),
		Netns:       os.Getenv("CNI_NETNS"),
		IfName:      os.Getenv("CNI_IFNAME"),
		Args:        os.Getenv("CNI_ARGS"),
		Path:        os.Getenv("CNI_PATH"),
		StdinData:   stdinData,
	}

	cmd := os.Getenv("CNI_COMMAND")
	return cmd, cmdArgs, nil
}

func handleIfCniUpdate(update func(*skel.CmdArgs) error) (bool, error) {
	isupdate := true

	if os.Getenv("CNI_COMMAND") != cni.CmdUpdate {
		return false, nil
	}

	log.Printf("CNI UPDATE received.")

	_, cmdArgs, err := getCmdArgsFromEnv()
	if err != nil {
		log.Printf("Received error while retrieving cmds from environment: %+v", err)
		return isupdate, err
	}

	log.Printf("Retrieved command args for update +%v", cmdArgs)
	err = validateConfig(cmdArgs.StdinData)
	if err != nil {
		log.Printf("Failed to handle CNI UPDATE, err:%v.", err)
		return isupdate, err
	}

	err = update(cmdArgs)
	if err != nil {
		log.Printf("Failed to handle CNI UPDATE, err:%v.", err)
		return isupdate, err
	}

	return isupdate, nil
}

// Main is the entry point for CNI network plugin.
func main() {

	// Initialize and parse command line arguments.
	acn.ParseArgs(&args, printVersion)
	vers := acn.GetArg(acn.OptVersion).(bool)

	if vers {
		printVersion()
		os.Exit(0)
	}

	var (
		config common.PluginConfig
		err    error
	)

	log.SetName(name)
	log.SetLevel(log.LevelInfo)
	if err = log.SetTarget(log.TargetLogfile); err != nil {
		fmt.Printf("Failed to setup cni logging: %v\n", err)
		return
	}

	defer log.Close()

	config.Version = version
	reportManager := &telemetry.ReportManager{
		HostNetAgentURL: hostNetAgentURL,
		ContentType:     telemetry.ContentType,
		Report: &telemetry.CNIReport{
			Context:          "AzureCNI",
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
	startTime := time.Now().UnixNano() / int64(time.Millisecond)

	netPlugin, err := network.NewPlugin(name, &config)
	if err != nil {
		log.Printf("Failed to create network plugin, err:%v.\n", err)
		return
	}

	netPlugin.SetCNIReport(cniReport)

	// CNI Acquires lock
	if err = netPlugin.Plugin.InitializeKeyValueStore(&config); err != nil {
		log.Errorf("Failed to initialize key-value store of network plugin, err:%v.\n", err)
		tb := telemetry.NewTelemetryBuffer("")
		if tberr := tb.Connect(); tberr == nil {
			reportPluginError(reportManager, tb, err)
			tb.Close()
		}

		if isSafe, _:= netPlugin.Plugin.IsSafeToRemoveLock(name); isSafe {
			log.Printf("[CNI] Removing lock file as process holding lock exited")
			if errUninit := netPlugin.Plugin.UninitializeKeyValueStore(true); errUninit != nil {
				log.Errorf("Failed to uninitialize key-value store of network plugin, err:%v.\n", errUninit)
			}
		}

		return
	}

	defer func() {
		if errUninit := netPlugin.Plugin.UninitializeKeyValueStore(false); errUninit != nil {
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

	t := time.Now()
	cniReport.Timestamp = t.Format("2006-01-02 15:04:05")

	if err = netPlugin.Start(&config); err != nil {
		log.Errorf("Failed to start network plugin, err:%v.\n", err)
		reportPluginError(reportManager, tb, err)
		panic("network plugin start fatal error")
	}

	handled, err := handleIfCniUpdate(netPlugin.Update)
	if handled == true {
		log.Printf("CNI UPDATE finished.")
	} else if err = netPlugin.Execute(cni.PluginApi(netPlugin)); err != nil {
		log.Errorf("Failed to execute network plugin, err:%v.\n", err)
	}

	endTime := time.Now().UnixNano() / int64(time.Millisecond)
	reflect.ValueOf(reportManager.Report).Elem().FieldByName("OperationDuration").SetInt(int64(endTime - startTime))

	netPlugin.Stop()

	// release cni lock
	if errUninit := netPlugin.Plugin.UninitializeKeyValueStore(false); errUninit != nil {
		log.Errorf("Failed to uninitialize key-value store of network plugin, err:%v.\n", errUninit)
	}

	if err != nil {
		reportPluginError(reportManager, tb, err)
		panic("network plugin execute fatal error")
	}

	// Report CNI successfully finished execution.
	reflect.ValueOf(reportManager.Report).Elem().FieldByName("CniSucceeded").SetBool(true)

	if err = reportManager.SendReport(tb); err != nil {
		log.Errorf("SendReport failed due to %v", err)
	} else {
		log.Printf("Sending report succeeded")
	}

}
