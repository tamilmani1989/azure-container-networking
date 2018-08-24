package customlogger

import (
	"dnc/configuration"
	"dnc/netagentcontracts"
	"fmt"

	"github.com/Azure/azure-container-networking/log"
)

// CustomLogger :- struct to hold CustomLogger settings.
type CustomLogger struct {
	instance *log.Logger
	reports  chan netagentcontracts.DNCReport
}

// NewCustomLogger :- Create new CustomLogger
func NewCustomLogger(cfg configuration.CustomLoggerSettings) *CustomLogger {
	customLogger := new(CustomLogger)
	return customLogger
}

// Init :- Initialize CustomLogger
func (l *CustomLogger) Init(stderr bool, reports chan netagentcontracts.DNCReport) {
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
		l.reports <- netagentcontracts.DNCReport{EventMessage: fmt.Sprintf(format, args...)}
	}()
}

// Close :- Close CustomLogger
func (l *CustomLogger) Close() {
	l.instance.Close()
}
