package acnlogger

import (
	"fmt"
	"regexp"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/telemetry"
)

const (
	errorcodePrefix = 5
)

// AcnLogger :- struct to hold AcnLogger settings.
type AcnLogger struct {
	instance *log.Logger
	reports  chan telemetry.DNCReport
}

// NewAcnLogger :- Create new AcnLogger
func NewAcnLogger() *AcnLogger {
	acnLogger := new(AcnLogger)
	return acnLogger
}

// Init :- Initialize AcnLogger
func (l *AcnLogger) Init(stderr bool, reports chan telemetry.DNCReport) {
	logLevel := log.LevelDebug
	logTarget := log.TargetLogfile

	if stderr {
		logTarget = log.TargetStdOutAndLogFile
	}

	l.instance = log.NewLogger("azure-dnc", logLevel, logTarget)
	l.reports = reports
}

// Debugf :- Log debug statements using AcnLogger
func (l *AcnLogger) Debugf(format string, args ...interface{}) {
	l.instance.Debugf(format, args...)
}

// Errorf :- Log errors using AcnLogger
func (l *AcnLogger) Errorf(format string, args ...interface{}) {
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

// Close :- Close AcnLogger
func (l *AcnLogger) Close() {
	l.instance.Close()
}
