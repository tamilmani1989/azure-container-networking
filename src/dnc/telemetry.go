package main

import (
	"dnc/aiwrapper"
	"dnc/api"
	"dnc/datamodel"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"
)

const (
	// DNCTelemetryFile - telemetry file path.
	DNCTelemetryFile = platform.DNCRuntimePath + "AzureDNCTelemetry.json"
	// HeartBeatInterval for reporting to mds
	heartBeatIntervalInMinutes = 30
	// Retrywaittime before trying to attempt connection again
	retryWaitTimeInSeconds          = 60
	telemetryNumRetries             = 5
	telemetryWaitTimeInMilliseconds = 200
)

func sendTelemetry() {
	orchestrator := ""
CONNECT:
	tb := telemetry.NewTelemetryBuffer("")
	tb.ConnectToTelemetryService(telemetryNumRetries, telemetryWaitTimeInMilliseconds)
	if tb.Connected {
		heartbeat := time.NewTicker(time.Minute * heartBeatIntervalInMinutes).C
		reportMgr := telemetry.ReportManager{
			ContentType: telemetry.ContentType,
			Report: &telemetry.DNCReport{
				PartitionKey: config.DataStoreSettings.AzureStorageInfo.PartitionKey,
			},
		}

		reportMgr.GetReportState(DNCTelemetryFile)
		reportMgr.GetKernelVersion()

		for {
			if replicaInstance.IsReplicaPrimary() {
				select {
				case <-heartbeat:
					numberOfNodes := 0
					numberOfNCs := 0
					var containerTypesSlice []string
					// get all nodes and their configurations
					nodes, err := datamodel.GetAllNodeIDs()
					if err == nil {
						numberOfNodes = len(nodes)
						// allocations - mapping of networkID to subnet to nodeID to ncID:ip
						allocations := make(map[string]map[string]map[string][]string)
						for nodeKey := range nodes {
							nodeID := datamodel.GetNodeIDFromKey(nodeKey)
							sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
							sessionManager.BeginReadSession(sessionInfo)
							nodeConfig, jsonerr := datamodel.TryQueryNodeConfigurationByKey(nodeKey)

							sessionManager.EndReadSession(sessionInfo)
							if jsonerr == nil {
								if nodeConfig != nil {
									// Currently we will only have one orchestrator per DNC ring, so we can record any orchestrator from nodeinfo
									orchestrator = nodeConfig.NodeInfo.OrchestratorType
									// loop through the deployed network containers in each node
									for networkID, ncMap := range nodeConfig.DeployedNetworkContainerMap {
										reservedIPs, jsonerr := reportNetworkConfiguration(nodeID, networkID, allocations)
										if jsonerr == nil {
											// if a reservationID matches an existing reservation, combine the corresponding ncID and its IP
											numberOfNCs += len(ncMap)
											for ncID, reservationID := range ncMap {
												for subnet := range allocations[networkID] {
													if ip, exists := reservedIPs[reservationID]; exists {
														allocations[networkID][subnet][nodeID] = append(allocations[networkID][subnet][nodeID], ncID+":"+ip)
														// Log to event snapshots
														aiwrapper.LogNCSnapsnot(ncID, networkID, subnet, nodeID)
														break
													}
												}
											}
										} else {
											err = fmt.Errorf(jsonerr.String())
											goto cleanup
										}
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

						containerTypes.Range(func(key interface{}, value interface{}) bool {
							containerTypesSlice = append(containerTypesSlice, key.(string))
							return true
						})

						containerTypesStr, err := json.Marshal(containerTypesSlice)
						if err == nil {
							reflect.ValueOf(reportMgr.Report).Elem().FieldByName("ContainerType").SetString(string(containerTypesStr))
						}
					}

				cleanup:
					if err != nil {
						logger.Debugf("[telemetry] %s", err.Error())
					}

					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("EventMessage").SetString("Heartbeat")
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("NumberOfNodes").SetInt(int64(numberOfNodes))
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("NumberOfNCs").SetInt(int64(numberOfNCs))
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Orchestrator").SetString(orchestrator)

					// We can use this upon CLA approval
					// reportMgr.GetSystemDetails()

				case msg := <-reports:
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("EventMessage").SetString(msg.EventMessage)
					reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Errorcode").SetString(msg.Errorcode)

				case <-telemetryStopProcessing:
					tb.Close()
					return
				}

				reflect.ValueOf(reportMgr.Report).Elem().FieldByName("Timestamp").SetString(time.Now().UTC().String())

				if !reportMgr.GetReportState(DNCTelemetryFile) {
					reportMgr.SetReportState(DNCTelemetryFile)
				}

				report, err := reportMgr.ReportToBytes()
				if err == nil {
					// If write fails, try to re-establish connections as server/client
					if _, err = tb.Write(report); err != nil {
						tb.Close()
						goto CONNECT
					}
				}
			} else {
				time.Sleep(time.Minute * 1)
			}
		}
	}
}

// reportNetworkConfiguration - initialize allocations with networkID subnetName and nodeID and return a map of reserved IPs
func reportNetworkConfiguration(nodeID, networkID string, allocations map[string]map[string]map[string][]string) (map[string]string, *api.JsonErr) {
	reservedIPs := make(map[string]string)
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	sessionManager.BeginReadSession(sessionInfo)
	networkConfig, err := datamodel.TryQueryNetworkConfiguration(networkID)
	sessionManager.EndReadSession(sessionInfo)
	if err == nil && networkConfig != nil {
		allocations[networkID] = make(map[string]map[string][]string)
		for subnetName, subnetKey := range networkConfig.Subnets {
			subnetConfig, err := datamodel.TryQuerySubnetConfigurationByKey(subnetKey)
			if err == nil && subnetConfig != nil {
				for reservation, ip := range subnetConfig.Reservations {
					if _, exists := allocations[networkID][subnetName]; !exists {
						allocations[networkID][subnetName] = make(map[string][]string)
					}

					if _, exists := allocations[networkID][subnetName][nodeID]; !exists {
						allocations[networkID][subnetName][nodeID] = make([]string, 0)
					}

					reservedIPs[reservation] = ip
				}
			} else {
				if subnetConfig == nil {
					logger.Errorf("UNEXPECTED ERROR: subnet configuration not found for subnet: %s", subnetKey)
				}

				break
			}
		}
	}

	if networkConfig == nil {
		logger.Errorf("UNEXPECTED ERROR: network configuration not found for network: %s", networkID)
	}

	return reservedIPs, err
}

func setOrchestratorForTelemetry() {
	var nodeID string
	nodes, err := datamodel.GetAllNodeIDs()
	if err == nil {
		for nodeKey := range nodes {
			nodeID = datamodel.GetNodeIDFromKey(nodeKey)
			break
		}
	}

	if nodeID == "" {
		logger.Debugf("No nodes added for DNC")
		return
	}

	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	sessionManager.BeginReadSession(sessionInfo)
	nodeConfig, jsonerr := datamodel.QueryNodeConfiguration(nodeID)
	sessionManager.EndReadSession(sessionInfo)

	if err != nil {
		logger.Errorf("QueryNodeConfiguration returned error %v", jsonerr)
		return
	}

	logger.Debugf("Orchestrator type: %v", nodeConfig.NodeInfo.OrchestratorType)
	aiwrapper.OrchestratorType = nodeConfig.NodeInfo.OrchestratorType
}
