package main

import (
	"dnc/datamodel"
	"dnc/netagentcontracts"
	"encoding/json"
	"reflect"
	"time"

	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"
)

// DNCReport structure.
type DNCReport struct {
	IsNewInstance bool
	CPUUsage      string
	MemoryUsage   string
	Processes     string
	EventMessage  string
	PartitionKey  string
	Allocations   string
	Metadata      telemetry.Metadata `json:"compute"`
}

const (
	// DNCTelemetryFile Path.
	DNCTelemetryFile = platform.DNCRuntimePath + "AzureDNCTelemetry.json"
	HostNetAgentUrl  = "http://169.254.169.254/machine/plugins?comp=netagent&type=dncreport"
)

func sendTelemetry() {
	heartbeat := time.NewTicker(time.Minute * 30).C
	reportMgr := telemetry.ReportManager{
		HostNetAgentURL: HostNetAgentUrl,
		ContentType:     telemetry.ContentType,
		Report: &DNCReport{
			PartitionKey: config.DataStoreSettings.AzureStorageInfo.PartitionKey,
		},
	}

	if err := reportMgr.GetHostMetadata(); err != nil {
		reports <- netagentcontracts.DNCReport{EventMessage: "Failed to retrieve host metadata."}
	}

	reportMgr.GetReportState(DNCTelemetryFile)
	reportMgr.GetKernelVersion()

	for {
		if replicaInstance.IsReplicaPrimary() {
			select {
			case <-heartbeat:
				reflect.ValueOf(reportMgr.Report).Elem().FieldByName("EventMessage").SetString("Heartbeat")
				// report updated network configurations
				networksMap, err := datamodel.GetAllNetworkIDs()
				if err == nil {
					allocations := make(map[string]map[string][]string)
					for networkKey := range networksMap {
						networkID := datamodel.GetNetworkIdFromKey(networkKey)
						allocations[networkID] = reportNetworkConfiguration(networkID, networkKey)
					}

					allocationsStr, err := json.Marshal(allocations)
					if err == nil {
						reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Allocations").SetString(string(allocationsStr))
					}
				}

				if err != nil {
					logger.Debugf("[telemetry] %s", err.Error())
				}

				// We can use this upon CLA approval
				// reportMgr.GetSystemDetails()

			case msg := <-reports:
				reflect.ValueOf(reportMgr.Report).Elem().FieldByName("EventMessage").SetString(msg.EventMessage)
			}

			if err := reportMgr.SendReport(); err == nil {
				reportMgr.SetReportState(DNCTelemetryFile)
			} else {
				logger.Debugf("[telemetry] %s", err.Error())
			}
		} else {
			time.Sleep(time.Minute * 1)
		}
	}
}

func reportNetworkConfiguration(networkID, networkKey string) map[string][]string {
	allocations := make(map[string][]string)
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	sessionManager.BeginReadSession(sessionInfo)

	networkConfig, err := datamodel.TryQueryNetworkConfigurationByKey(networkKey)
	if err == nil {
		for subnetName, subnetKey := range networkConfig.Subnets {
			subnetConfig, err := datamodel.TryQuerySubnetConfigurationByKey(subnetKey)
			if err == nil {
				for _, reservation := range subnetConfig.Reservations {
					allocations[subnetName] = append(allocations[subnetName], reservation)
				}
			} else {
				logger.Debugf("[telemetry] %s", err.String())
			}
		}
	} else {
		logger.Debugf("[telemetry] %s", err.String())
	}

	sessionManager.EndReadSession(sessionInfo)
	return allocations
}
