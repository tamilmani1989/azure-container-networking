package aiwrapper

import (
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

func CreateTelemetryHandle(aiConfig aitelemetry.AIConfig) error {
	if aiMetadata == "" {
		return nil;
	}

	var err error
	th, err = aitelemetry.NewAITelemetry("", aiMetadata, aiConfig)

	if(err != nil){
		return err;
	}

	return nil;
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

// Trace the report to AI.
func Trace(report aitelemetry.Report) {
	if th == nil {
		return
	}

	report.CustomDimensions[OrchestratorStr] = OrchestratorType
	report.CustomDimensions["Context"] = "AzureDNC"
	report.Context = OrchestratorType
	th.TrackLog(report)
}
