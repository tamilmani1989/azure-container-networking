package logger

import (
	"fmt"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/log"
)

const (
	waitTimeInSecs = 10
)

var Log *CNSLogger

type CNSLogger struct {
	logger               *log.logger
	th                   *telemetryHandle
	DisableAILogging     bool
	DisableTraceLogging  bool
	DisableMetricLogging bool
}

func InitLogger(fileName string, aiConfig aitelmetry.AIConfig, disableAILogging, disableTraceLogging, disableMetricLogging bool) {
	logLevel := log.LevelDebug
	logTarget := log.TargetLogfile

	Log = &CNSLogger{
		logger:               log.NewLogger(fileName, logLevel, logTarget),
		disableAILogging:     disableAILogging,
		DisableTraceLogging:  disableTraceLogging,
		DisableMetricLogging: disableMetricLogging,
	}

	if !disableAILogging {
		Log.th = aitelemetry.NewAITelemetry(aiMetadata, aiConfig)
	}
}

func Close() {
	Log.logger.Close()
	Log.th.Close(waitTimeInSecs)
}

func SetLogDirectory(dir string) {
	Log.logger.SetLogDirectory(dir)
}

func SetReportDetails(
	orchestrartor string,
	nodeID string,
) {
	Log.Orchestrartor = orchestrartor
	Log.NodeID = nodeID
}

func Printf(format string, args ...interface{}) {
	Log.logger.Printf(format, args...)
	if !l.disableAILogging && !l.disableTraceLogging {
		report = aitelemetry.Report{CustomDimensions: make(map[string]string)}
		report.Message = fmt.Sprintf(format, args...)
		report.CustomDimensions[OrchestrartorTypeStr] = Log.Orchestrartor
		report.CustomDimensions[NodeIDStr] = Log.NodeID
		report.Context = nodeID
		Log.th.TrackLog(report)
	}
}

func Errorf(format string, args ...interface{}) {
	Log.logger.Errorf(format, args...)
	if !l.disableAILogging && !l.disableTraceLogging {
		report = aitelemetry.Report{CustomDimensions: make(map[string]string)}
		report.Message = fmt.Sprintf(format, args...)
		report.CustomDimensions[OrchestrartorTypeStr] = Log.Orchestrartor
		report.CustomDimensions[NodeIDStr] = Log.NodeID
		report.Context = nodeID
		Log.th.TrackLog(report)
	}
}


func SendMetric(metric aitelemetry.metric) {
	if !l.disableAILogging && !l.disableMetric {
		metric.CustomDimensions[OrchestrartorTypeStr] = Log.Orchestrartor
		metric.CustomDimensions[NodeIDStr] = Log.NodeID
		Log.th.TrackMetric(metric)
	}
}
