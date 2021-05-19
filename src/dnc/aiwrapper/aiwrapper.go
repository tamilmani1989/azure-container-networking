package aiwrapper

import (
	"sync"

	"github.com/Azure/azure-container-networking/aitelemetry"
)

var (
	// todo: assign through config, not at build time
	aiMetadata string

	// todo: implement proper wrapper type around the handle, so that callers don't depend on a global.
	//  may need mutex as well.
	th aitelemetry.TelemetryHandle

	orchTypeLock     sync.RWMutex
	orchestratorType string
)

// CreateTelemetryHandle creates a telemetry handle using param iKey or global variable aiMetadata, which is usually set at build time
func CreateTelemetryHandle(iKey string, aiConfig aitelemetry.AIConfig) error {
	if aiMetadata == "" && iKey == "" {
		return nil
	}

	// if ikey was passed in, use it instead of build-time value stored in aiMetadata
	if iKey != "" {
		aiMetadata = iKey
	}

	var err error
	th, err = aitelemetry.NewAITelemetry("", aiMetadata, aiConfig)

	return err
}

func SetOrchestratorType(orchType string) {
	orchTypeLock.Lock()
	orchestratorType = orchType
	orchTypeLock.Unlock()
}

func SendMetric(metric aitelemetry.Metric) {
	if th == nil {
		return
	}

	orchTypeLock.RLock()
	metric.CustomDimensions[OrchestratorStr] = orchestratorType
	orchTypeLock.RUnlock()

	th.TrackMetric(metric)
}

// Trace the report to AI.
func Trace(report aitelemetry.Report) {
	if th == nil {
		return
	}

	orchTypeLock.RLock()
	report.CustomDimensions[OrchestratorStr] = orchestratorType
	report.Context = orchestratorType
	orchTypeLock.RUnlock()

	report.CustomDimensions["Context"] = "AzureDNC"
	th.TrackLog(report)
}

func Event(event aitelemetry.Event) {
	if th == nil {
		return
	}

	orchTypeLock.RLock()
	event.Properties[OrchestratorStr] = orchestratorType
	orchTypeLock.RUnlock()

	event.Properties["Context"] = "AzureDNC"
	th.TrackEvent(event)
}

// Logs NC snapshots to AI telemetry.
func LogNCSnapsnot(networkContainerId string, networkId string, subnetName string, nodeId string) {
	// Log NC Snapshot
	aiEvent := aitelemetry.Event{
		EventName:  DNCNCSnapshotStr,
		Properties: make(map[string]string),
		ResourceID: networkContainerId,
	}

	aiEvent.Properties[NetworkIDStr] = networkId
	aiEvent.Properties[SubNetStr] = subnetName
	aiEvent.Properties[NodeIDStr] = nodeId
	Event(aiEvent)
}
