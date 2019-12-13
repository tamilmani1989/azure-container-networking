package telemetry

import (
	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/log"
)

var (
	aiMetadata     string
	th             aitelemetry.TelemetryHandle
	gDisableAll    bool
	gDisableTrace  bool
	gDisableMetric bool
)

const (
	waitTimeInSecs = 10
)

func CreateAITelemetryHandle(aiConfig aitelemetry.AIConfig, disableAll, disableMetric, disableTrace bool) {
	th = aitelemetry.NewAITelemetry(aiMetadata, aiConfig)
	gDisableAll = disableAll
	gDisableMetric = disableMetric
	gDisableTrace = disableTrace
}

func sendReport(cnireport CNIReport) {
	var msg string
	if cnireport.ErrorMessage != "" {
		msg = cnireport.ErrorMessage
	} else {
		msg = cnireport.EventMessage
	}

	report := aitelemetry.Report{
		Message:          "CNI:" + msg,
		Context:          cnireport.ContainerName,
		CustomDimensions: make(map[string]string),
	}

	report.CustomDimensions[ContextStr] = cnireport.Context
	report.CustomDimensions[SubContextStr] = cnireport.SubContext
	report.CustomDimensions[VMUptimeStr] = cnireport.VMUptime
	report.CustomDimensions[OperationTypeStr] = cnireport.OperationType
	report.CustomDimensions[VersionStr] = cnireport.Version

	th.TrackLog(report)
}

func SendAITelemetry(cnireport CNIReport) {
	if gDisableAll {
		log.Printf("Sending Telemetry data is disabled")
		return
	}

	if cnireport.Metric.Name != "" && !gDisableMetric {
		th.TrackMetric(cnireport.Metric)
	} else if !gDisableTrace {
		sendReport(cnireport)
	}
}

func CloseAITelemetryHandle() {
	th.Close(waitTimeInSecs)
}
