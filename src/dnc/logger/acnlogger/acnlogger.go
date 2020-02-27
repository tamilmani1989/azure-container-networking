package acnlogger

import (
	"dnc/aiwrapper"
	"dnc/configuration"
	"dnc/logger"
	"fmt"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/telemetry"
)

// AcnLogger :- struct to hold AcnLogger settings.
type AcnLogger struct {
	instance          *log.Logger
	reports           chan telemetry.DNCReport
	telemetrySettings configuration.TelemetrySettings
}

// NewAcnLogger :- Create new AcnLogger
func NewAcnLogger(cfg *configuration.Configuration) *AcnLogger {
	acnLogger := new(AcnLogger)
	acnLogger.telemetrySettings = cfg.TelemetrySettings
	return acnLogger
}

// Init :- Initialize AcnLogger
func (l *AcnLogger) Init(stderr bool, reports chan telemetry.DNCReport) {
	logLevel := log.LevelDebug
	logTarget := log.TargetLogfile

	if stderr {
		logTarget = log.TargetStdOutAndLogFile
	}

	l.instance = log.NewLogger("azure-dnc", logLevel, logTarget, "")
	l.reports = reports
}

// Debugf :- Log debug statements using AcnLogger
func (l *AcnLogger) Debugf(format string, args ...interface{}) {
	l.instance.Debugf(format, args...)
	if l.reports != nil {
		go func() {
			msg := fmt.Sprintf(format, args...)
			l.reports <- telemetry.DNCReport{EventMessage: msg}
			if !l.telemetrySettings.DisableAiLogging {
				logAiMessage("Debug", msg)
			}
		}()
	}
}

// Errorf :- Log errors using AcnLogger
func (l *AcnLogger) Errorf(format string, args ...interface{}) {
	l.instance.Printf(format, args...)
	if l.reports != nil {
		go func() {
			msg := fmt.Sprintf(format, args...)
			codeStr := logger.CompiledRegex.FindString(msg)
			if len(codeStr) > logger.ErrorCodePrefix {
				l.reports <- telemetry.DNCReport{Errorcode: codeStr[logger.ErrorCodePrefix:], EventMessage: msg}
			} else {
				l.reports <- telemetry.DNCReport{EventMessage: msg}
			}

			if !l.telemetrySettings.DisableAiLogging {
				logAiMessage("Error", msg)
			}
		}()
	}
}

// Close :- Close AcnLogger
func (l *AcnLogger) Close() {
	l.instance.Close()
}

// SetLogFileLimits :- Set the limits for the log file
func (l *AcnLogger) SetLogFileLimits(maxFileSize, maxFileCount int) {
	l.instance.SetLogFileLimits(maxFileSize, maxFileCount)
}

// LogAiMessage :- Logs Application insights message.
func logAiMessage(messageType string, msg string) {
	report := aitelemetry.Report{CustomDimensions: make(map[string]string)}
	report.Message = msg

	// Indicates whether this message is error or debug message.
	report.CustomDimensions["MessageType"] = messageType

	codeStr := logger.CompiledRegex.FindString(msg)

	// If there is an error code add it to dimensions.
	if len(codeStr) > logger.ErrorCodePrefix {
		report.CustomDimensions["ErrorCode"] = codeStr
	}

	aiwrapper.Trace(report)
}
