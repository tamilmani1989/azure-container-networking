package aiwrapper

import (
	"dnc/utils"

	"github.com/Azure/azure-container-networking/aitelemetry"
)

const (
	timeout = 10
)

var (
	aiMetadata       string
	OrchestratorType string
	th               aitelemetry.TelemetryHandle
)

func CreateTelemetryHandle(aiConfig aitelemetry.AIConfig) {
	if aiMetadata == "" {
		return
	}

	aiConfig.AppVersion = utils.Version
	aiConfig.AppName = appName
	th = aitelemetry.NewAITelemetry(aiMetadata, aiConfig)
}

func CloseTelemetryHandle() {
	th.Close(timeout)
}

func SendMetric(metric aitelemetry.Metric) {
	if th == nil {
		return
	}

	metric.CustomDimensions[OrchestratorStr] = OrchestratorType
	th.TrackMetric(metric)
}
