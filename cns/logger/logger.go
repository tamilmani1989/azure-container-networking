package logger

import (
	"fmt"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/log"
)

const (
	waitTimeInSecs = 10
)

var (
	Log        *CNSLogger
	aiMetadata string
)

type CNSLogger struct {
	logger               *log.Logger
	th                   aitelemetry.TelemetryHandle
	Orchestrartor        string
	NodeID               string
	DisableTraceLogging  bool
	DisableMetricLogging bool
}

func InitLogger(fileName string, logLevel, logTarget int) {
	Log = &CNSLogger{
		logger: log.NewLogger(fileName, logLevel, logTarget),
	}
}

func InitAI(aiConfig aitelemetry.AIConfig, disableTraceLogging, disableMetricLogging bool) {
	Log.th = aitelemetry.NewAITelemetry(aiMetadata, aiConfig)
	Log.DisableMetricLogging = disableMetricLogging
	Log.DisableTraceLogging = disableTraceLogging
}

func Close() {
	Log.logger.Close()
	if Log.th != nil {
		Log.th.Close(waitTimeInSecs)
	}
}

func SetLogDirectory(dir string) {
	Log.logger.SetLogDirectory(dir)
}

func SetContextDetails(
	orchestrartor string,
	nodeID string,
) {
	Printf("SetContext details called with: %v orchestrator nodeID %v", orchestrartor, nodeID)
	Log.Orchestrartor = orchestrartor
	Log.NodeID = nodeID
}

func sendTraceInternal(msg string) {
	report := aitelemetry.Report{CustomDimensions: make(map[string]string)}
	report.Message = msg
	report.CustomDimensions[OrchestratorTypeStr] = Log.Orchestrartor
	report.CustomDimensions[NodeIDStr] = Log.NodeID
	report.Context = Log.NodeID
	Log.th.TrackLog(report)
}

func Printf(format string, args ...interface{}) {
	Log.logger.Printf(format, args...)

	if Log.th == nil || Log.DisableTraceLogging {
		return
	}

	msg := fmt.Sprintf(format, args...)
	sendTraceInternal(msg)
}

func Debugf(format string, args ...interface{}) {
	Log.logger.Debugf(format, args...)

	if Log.th == nil || Log.DisableTraceLogging {
		return
	}

	msg := fmt.Sprintf(format, args...)
	sendTraceInternal(msg)
}

func Errorf(format string, args ...interface{}) {
	Log.logger.Errorf(format, args...)

	if Log.th == nil || Log.DisableTraceLogging {
		return
	}

	msg := fmt.Sprintf(format, args...)
	sendTraceInternal(msg)
}

func SendMetric(metric aitelemetry.Metric) {
	if Log.th == nil || Log.DisableMetricLogging {
		return
	}

	metric.CustomDimensions[OrchestratorTypeStr] = Log.Orchestrartor
	metric.CustomDimensions[NodeIDStr] = Log.NodeID
	Log.th.TrackMetric(metric)
}
