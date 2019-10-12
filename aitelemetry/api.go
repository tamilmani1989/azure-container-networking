package aitelemetry

import (
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
	metadata              Metadata
	diagListener          appinsights.DiagnosticsMessageListener
	client                appinsights.TelemetryClient
	isMetadataInitialized bool
}

// Metadata retrieved from wireserver
type Metadata struct {
	Location             string `json:"location"`
	VMName               string `json:"name"`
	Offer                string `json:"offer"`
	OsType               string `json:"osType"`
	PlacementGroupID     string `json:"placementGroupId"`
	PlatformFaultDomain  string `json:"platformFaultDomain"`
	PlatformUpdateDomain string `json:"platformUpdateDomain"`
	Publisher            string `json:"publisher"`
	ResourceGroupName    string `json:"resourceGroupName"`
	Sku                  string `json:"sku"`
	SubscriptionID       string `json:"subscriptionId"`
	Tags                 string `json:"tags"`
	OSVersion            string `json:"version"`
	VMID                 string `json:"vmId"`
	VMSize               string `json:"vmSize"`
	KernelVersion        string
}

type metadataWrapper struct {
	Metadata Metadata `json:"compute"`
}
