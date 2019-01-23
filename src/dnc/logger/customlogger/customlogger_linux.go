package customlogger

import (
	"dnc/configuration"
	"fmt"
	"regexp"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/telemetry"
)

const (
	errorcodePrefix = 5
)

// CustomLogger :- struct to hold CustomLogger settings.
type CustomLogger struct {
	instance *log.Logger
	reports  chan telemetry.DNCReport
}

// NewCustomLogger :- Create new CustomLogger
func NewCustomLogger(cfg configuration.CustomLoggerSettings) *CustomLogger {
	customLogger := new(CustomLogger)
	return customLogger
}

// Init :- Initialize CustomLogger
func (l *CustomLogger) Init(stderr bool, reports chan telemetry.DNCReport) {
	logLevel := log.LevelDebug
	logTarget := log.TargetLogfile

	if stderr {
		logTarget = log.TargetStdOutAndLogFile
	}

	l.instance = log.NewLogger("azure-dnc", logLevel, logTarget)
	l.reports = reports
}

// Debugf :- Log debug statements using CustomLogger
func (l *CustomLogger) Debugf(format string, args ...interface{}) {
	l.instance.Debugf(format, args...)
}

// Errorf :- Log errors using CustomLogger
func (l *CustomLogger) Errorf(format string, args ...interface{}) {
	l.instance.Printf(format, args...)
	go func() {
		msg := fmt.Sprintf(format, args...)
		codeStr := regexp.MustCompile(`Code:(\w*)`).FindString(msg)
		if len(codeStr) > errorcodePrefix {
			l.reports <- telemetry.DNCReport{Errorcode: codeStr[errorcodePrefix:], EventMessage: msg}
		} else {
			l.reports <- telemetry.DNCReport{EventMessage: msg}
		}
	}()
}

// Close :- Close CustomLogger
func (l *CustomLogger) Close() {
	l.instance.Close()
}
