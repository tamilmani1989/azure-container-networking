package acnlogger

import (
	"github.com/Azure/azure-container-networking/log"
)

// AcnLogger :- struct to hold AcnLogger settings.
type AcnLogger struct {
	instance *log.Logger
}

// NewAcnLogger :- Create new AcnLogger
func NewAcnLogger() *AcnLogger {
	acnLogger := new(AcnLogger)
	return acnLogger
}

// Init :- Initialize AcnLogger
func (l *AcnLogger) Init(debug, stderr bool) {
	logLevel := log.LevelInfo
	logTarget := log.TargetLogfile

	if debug {
		logLevel = log.LevelDebug
	}

	if stderr {
		logTarget = log.TargetStderr
	}

	l.instance = log.NewLogger("azure-dnc", logLevel, logTarget)
}

// Debugf :- Log debug statements using AcnLogger
func (l *AcnLogger) Debugf(format string, args ...interface{}) {
	l.instance.Debugf(format, args...)
}

// Errorf :- Log errors using AcnLogger
func (l *AcnLogger) Errorf(format string, args ...interface{}) {
	l.instance.Printf(format, args...)
}

// Close :- Close AcnLogger
func (l *AcnLogger) Close() {
	l.instance.Close()
}
