// Copyright Microsoft. All rights reserved.
package telemetry

import (
	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/log"
)

var (
	aiMetadata     string
	th             aitelemetry.TelemetryHandle
	gDisableTrace  bool
	gDisableMetric bool
)

const (
	// Wait time for AI to gracefully close AI telemetry session
	waitTimeInSecs = 10
)

func CreateAITelemetryHandle(aiConfig aitelemetry.AIConfig, disableAll, disableMetric, disableTrace bool) {
	if disableAll {
		log.Printf("Telemetry is disabled")
		return
	}

	th = aitelemetry.NewAITelemetry(aiMetadata, aiConfig)
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
	if th == nil {
		return
	}

	if cnireport.Metric.Name != "" && !gDisableMetric {
		th.TrackMetric(cnireport.Metric)
	} else if !gDisableTrace {
		sendReport(cnireport)
	}
}

func CloseAITelemetryHandle() {
	if th != nil {
		th.Close(waitTimeInSecs)
	}
}
