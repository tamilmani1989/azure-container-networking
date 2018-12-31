package main

import (
	"dnc/api"
	"dnc/datamodel"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"
	"github.com/google/uuid"
)

const (
	// DNCTelemetryFile - telemetry file path.
	DNCTelemetryFile = platform.DNCRuntimePath + "AzureDNCTelemetry.json"
	// HostNetAgentURL - Host Net Agent url and content type
	HostNetAgentURL = "http://169.254.169.254/machine/plugins?comp=netagent&type=dncreport"
)

func sendTelemetry() {
	retrieveMetadata := true

CONNECT:
	telemetryBuffer, err := telemetry.NewTelemetryBuffer()
	if err == nil {
		go telemetryBuffer.Start(reportToHostInterval)

		heartbeat := time.NewTicker(time.Minute * 30).C
		reportMgr := telemetry.ReportManager{
			HostNetAgentURL: HostNetAgentURL,
			ContentType:     telemetry.ContentType,
			Report: &telemetry.DNCReport{
				PartitionKey: config.DataStoreSettings.AzureStorageInfo.PartitionKey,
			},
		}

		reportMgr.GetReportState(DNCTelemetryFile)
		reportMgr.GetKernelVersion()

		for {
			if replicaInstance.IsReplicaPrimary() {
				// Try to retrieve metadata until successful
				if retrieveMetadata {
					if err := reportMgr.GetHostMetadata(); err != nil {
						reports <- telemetry.DNCReport{EventMessage: fmt.Sprintf("Failed to retrieve host metadata with error: %s", err.Error())}
					} else {
						retrieveMetadata = false
					}
				}

				select {
				case <-heartbeat:
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("EventMessage").SetString("Heartbeat")
					// get all nodes and their configurations
					nodes, err := datamodel.GetAllNodeIDs()
					if err == nil {
						allocations := make(map[string]map[string]map[string][]string)
						for nodeKey := range nodes {
							nodeID := datamodel.GetNodeIDFromKey(nodeKey)
							sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
							sessionManager.BeginReadSession(sessionInfo)
							nodeConfig, jsonerr := datamodel.TryQueryNodeConfigurationByKey(nodeKey)
							sessionManager.EndReadSession(sessionInfo)
							if jsonerr == nil {
								// loop through the deployed network containers in each node
								for networkID, ncMap := range nodeConfig.DeployedNetworkContainerMap {
									reservedIPs, jsonerr := reportNetworkConfiguration(nodeID, networkID, allocations)
									if jsonerr == nil {
										// if a reservationID matches an existing reservation, combine the corresponding ncID and its IP
										for ncID, reservationID := range ncMap {
											for subnet := range allocations[networkID] {
												if ip, exists := reservedIPs[reservationID]; exists {
													allocations[networkID][subnet][nodeID] = append(allocations[networkID][subnet][nodeID], ncID+":"+ip)
													break
												}
											}
										}
									} else {
										err = fmt.Errorf(jsonerr.String())
										goto cleanup
									}
								}
							} else {
								err = fmt.Errorf(jsonerr.String())
								goto cleanup
							}
						}

						allocationsStr, err := json.Marshal(allocations)
						if err == nil {
							reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Allocations").SetString(string(allocationsStr))
						}
					}

				cleanup:
					if err != nil {
						logger.Debugf("[telemetry] %s", err.Error())
					}

					// We can use this upon CLA approval
					// reportMgr.GetSystemDetails()

				case msg := <-reports:
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("EventMessage").SetString(msg.EventMessage)
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Errorcode").SetString(msg.Errorcode)

				case <-telemetryStopProcessing:
					telemetryBuffer.Cancel()
					return
				}

				reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Timestamp").SetString(time.Now().UTC().String())
				if id, err := uuid.NewUUID(); err == nil {
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("UUID").SetString(id.String())
				}

				logger.Debugf("[Telemetry] %+v", reportMgr.Report)
				if !reportMgr.GetReportState(DNCTelemetryFile) {
					reportMgr.SetReportState(DNCTelemetryFile)
				}

				report, err := reportMgr.ReportToBytes()
				if err == nil {
					// If write fails, try to re-establish connections as server/client
					if _, err = telemetryBuffer.Write(report); err != nil {
						telemetryBuffer.Cancel()
						goto CONNECT
					}
				}
			} else {
				time.Sleep(time.Minute * 1)
			}
		}
	} else {
		logger.Debugf("[Telemetry] Failed to establish telemetry buffer connection.")
		time.Sleep(time.Minute * 1)
		goto CONNECT
	}
}

// reportNetworkConfiguration - initialize allocations with networkID subnetName and nodeID and return a map of reserved IPs
func reportNetworkConfiguration(nodeID, networkID string, allocations map[string]map[string]map[string][]string) (map[string]string, *api.JsonErr) {
	reservedIPs := make(map[string]string)
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	sessionManager.BeginReadSession(sessionInfo)
	networkConfig, err := datamodel.TryQueryNetworkConfiguration(networkID)
	sessionManager.EndReadSession(sessionInfo)
	if err == nil {
		allocations[networkID] = make(map[string]map[string][]string)
		for subnetName, subnetKey := range networkConfig.Subnets {
			subnetConfig, err := datamodel.TryQuerySubnetConfigurationByKey(subnetKey)
			if err == nil {
				for reservation, ip := range subnetConfig.Reservations {
					if _, exists := allocations[networkID][subnetName]; !exists {
						allocations[networkID][subnetName] = make(map[string][]string)
					}

					if _, exists := allocations[networkID][subnetName][nodeID]; !exists {
						allocations[networkID][subnetName][nodeID] = make([]string, 1)
					}

					reservedIPs[reservation] = ip
				}
			} else {
				break
			}
		}
	}

	return reservedIPs, err
}
