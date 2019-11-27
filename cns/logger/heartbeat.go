// Copyright 2018 Microsoft. All rights reserved.
// MIT License

package logger

import (
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/restserver"

	"github.com/Azure/azure-container-networking/platform"
)

const (
	// CNSTelemetryFile - telemetry file path.
	CNSTelemetryFile                = platform.CNSRuntimePath + "AzureCNSTelemetry.json"
	errorcodePrefix                 = 5
	heartbeatIntervalInMinutes      = 30
	retryWaitTimeInSeconds          = 60
	telemetryNumRetries             = 5
	telemetryWaitTimeInMilliseconds = 200
)

// SendCnsTelemetry - handles cns telemetry reports
func SendCnsTelemetry(reports chan interface{}, service *restserver.HTTPRestService, telemetryStopProcessing chan bool) {
	heartbeat := time.NewTicker(time.Minute * heartbeatIntervalInMinutes).C
	for {
		select {
		case <-heartbeat:
			metric := aitelemetry.Metric{
				Name:             HeartBeatMetricStr,
				Value:            1.0,
				CustomDimensions: make(map[string]string),
			}

			logger.SendMetric(metric)
		}
	}
}
