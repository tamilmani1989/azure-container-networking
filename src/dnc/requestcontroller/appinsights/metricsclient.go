// Copyright 2021 Microsoft. All rights reserved.

package appinsights

import (
	"dnc/aiwrapper"
	"dnc/configuration"
	"dnc/requestcontroller"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/sirupsen/logrus"
)

type MetricsClient struct {
	logger               logrus.FieldLogger
	persistentDimensions map[string]string
}

type MetricsClientOptions struct {
	Logger               logrus.FieldLogger
	Config               configuration.TelemetrySettings
	AppName              string
	AppVersion           string
	PersistentDimensions map[string]string
}

func NewMetricsClient(opts MetricsClientOptions) *MetricsClient {
	// Initialize the global AI Wrapper unless specifically requested not to
	if !opts.Config.DisableTelemetry {
		aiConfig := aitelemetry.AIConfig{
			BatchSize:                    opts.Config.TelemetryBatchSizeBytes,
			BatchInterval:                opts.Config.TelemetryBatchIntervalInSecs,
			RefreshTimeout:               opts.Config.RefreshIntervalInSecs,
			DisableMetadataRefreshThread: opts.Config.DisableMetadataRefreshThread,
			DebugMode:                    opts.Config.DebugMode,
			AppVersion:                   opts.AppVersion,
			AppName:                      opts.AppName,
			GetEnvRetryCount:             opts.Config.GetEnvRetryCount,
			GetEnvRetryWaitTimeInSecs:    opts.Config.GetEnvRetryWaitTimeInSecs,
		}
		if err := aiwrapper.CreateTelemetryHandle(opts.Config.AIMetadata, aiConfig); err != nil {
			// just log the error to be consistent with dnc, but maybe we should return this to the caller?
			opts.Logger.WithError(err).Error("error creating AITelemetry handle")
		}
	}
	// Here we return a MetricsClient regardless of error, since that is behavior in dnc today
	return &MetricsClient{
		logger:               opts.Logger,
		persistentDimensions: make(map[string]string),
	}
}

func (m *MetricsClient) SendMetric(metric requestcontroller.Metric) {
	// Merge the persistent dimensions and custom dimensions
	dimensions := make(map[string]string, len(metric.CustomDimensions)+len(m.persistentDimensions))
	// Copy persistent in first, so custom dimensions can override the values
	// read lock not required since m.persistentDimensions is immutable
	for k, v := range m.persistentDimensions {
		dimensions[k] = v
	}
	for k, v := range metric.CustomDimensions {
		dimensions[k] = v
	}

	aiMetric := aitelemetry.Metric{
		Name:             metric.Name,
		Value:            metric.Value,
		CustomDimensions: dimensions,
	}

	go aiwrapper.SendMetric(aiMetric)
}
