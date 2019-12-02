// Copyright 2018 Microsoft. All rights reserved.
// MIT License

package logger

import (
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
)

const (
	heartbeatIntervalInMinutes = 30
)

// SendCnsTelemetry - handles cns telemetry reports
func SendCnsTelemetry() {
	heartbeat := time.NewTicker(time.Minute * heartbeatIntervalInMinutes).C
	metric := aitelemetry.Metric{
		Name:             HeartBeatMetricStr,
		Value:            1.0,
		CustomDimensions: make(map[string]string),
	}

	for {
		select {
		case <-heartbeat:
			SendMetric(metric)
		}
	}
}
