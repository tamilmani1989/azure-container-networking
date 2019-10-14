package aitelemetry

import (
	"github.com/Azure/azure-container-networking/common"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

// Application trace/log structure
type Report struct {
	Message          string
	Context          string
	CustomDimensions map[string]string
}

// Application metrics structure
type Metric struct {
	Name             string
	Value            float64
	CustomDimensions map[string]string
}

// TelmetryHandle holds appinsight handles and metadata
type TelemetryHandle struct {
	telemetryConfig       *appinsights.TelemetryConfiguration
	AppName               string
	AppVersion            string
	metadata              common.Metadata
	diagListener          appinsights.DiagnosticsMessageListener
	client                appinsights.TelemetryClient
	isMetadataInitialized bool
}
