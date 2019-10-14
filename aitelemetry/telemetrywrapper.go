package aitelemetry

import (
	"runtime"
	"time"

	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/store"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

const (
	resourceGroupStr  = "ResourceGroup"
	vmSizeStr         = "VMSize"
	osVersionStr      = "OSVersion"
	locationStr       = "Region"
	appVersionStr     = "Appversion"
	subscriptionIDStr = "SubscriptionID"
	defaultTimeout    = 10
)

func messageListener() appinsights.DiagnosticsMessageListener {
	return appinsights.NewDiagnosticsMessageListener(func(msg string) error {
		log.Printf("[%s] %s\n", time.Now().Format(time.UnixDate), msg)
		return nil
	})
}

func getmetadata(th *TelemetryHandle) error {
	// check if metadata in memory otherwise initiate wireserver request
	if !th.isMetadataInitialized {
		var err error
		th.metadata, err = common.GetHostMetadata(metadataFile)
		if err != nil {
			log.Printf("Error getting metadata %v", err)
			return err
		}

		th.isMetadataInitialized = true

		// Save metadata retrieved from wireserver to a file
		kvs, err := store.NewJsonFileStore(metadataFile)
		if err != nil {
			log.Printf("Error acuiring lock for writing metadata file: %v", err)
		}

		kvs.Lock(true)
		err = common.SaveHostMetadata(th.metadata, metadataFile)
		if err != nil {
			log.Printf("saving host metadata failed with :%v", err)
		}
		kvs.Unlock(false)

	}

	return nil
}

// NewAITelemetry creates telemetry handle with user specified appinsights key.
func NewAITelemetry(
	key string,
	appName string,
	appVersion string,
) *TelemetryHandle {

	telemetryConfig := appinsights.NewTelemetryConfiguration(key)
	telemetryConfig.MaxBatchSize = 8192
	telemetryConfig.MaxBatchInterval = time.Duration(1) * time.Second

	telemetryHandle := &TelemetryHandle{
		client:       appinsights.NewTelemetryClientFromConfig(telemetryConfig),
		AppName:      appName,
		AppVersion:   appVersion,
		diagListener: messageListener(),
	}

	return telemetryHandle
}

// TrackLog function sends report (trace) to appinsights resource. It overrides few of the existing columns with app information
// and for rest it uses custom dimesion
func (th *TelemetryHandle) TrackLog(report Report) {
	// Initialize new trace message
	trace := appinsights.NewTraceTelemetry(report.Message, appinsights.Warning)

	//Override few of existing columns with metadata
	trace.Tags.User().SetAuthUserId(runtime.GOOS)
	trace.Tags.Operation().SetId(report.Context)
	trace.Tags.Operation().SetParentId(th.AppName)

	// copy app specified custom dimension
	for key, value := range report.CustomDimensions {
		trace.Properties[key] = value
	}

	trace.Properties[appVersionStr] = th.AppVersion

	var err error
	if err = getmetadata(th); err == nil {
		// copy metadata from wireserver to trace
		trace.Tags.User().SetAccountId(th.metadata.SubscriptionID)
		trace.Tags.User().SetId(th.metadata.VMName)
		trace.Properties[locationStr] = th.metadata.Location
		trace.Properties[resourceGroupStr] = th.metadata.ResourceGroupName
		trace.Properties[vmSizeStr] = th.metadata.VMSize
		trace.Properties[osVersionStr] = th.metadata.OSVersion
	} else {
		log.Printf("Error retrieving metadata from wireserver: %v", err)
	}

	// send to appinsights
	th.client.Track(trace)

}

func (th *TelemetryHandle) TrackMetric(metric Metric) {
	aimetric := appinsights.NewMetricTelemetry(metric.Name, metric.Value)

	var err error
	if err = getmetadata(th); err == nil {
		aimetric.Properties[locationStr] = th.metadata.Location
		aimetric.Properties[subscriptionIDStr] = th.metadata.SubscriptionID
	} else {
		log.Printf("Error retrieving metadata from wireserver: %v", err)
	}

	for key, value := range metric.CustomDimensions {
		aimetric.Properties[key] = value
	}

	th.client.Track(aimetric)
}

func (th *TelemetryHandle) Close(timeout int) {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	// wait for items to be sent otherwise timeout
	<-th.client.Channel().Close(time.Duration(timeout) * time.Second)

	// Remove diganostic message listener
	if th.diagListener != nil {
		th.diagListener.Remove()
		th.diagListener = nil
	}
}
