package acnlogger

import (
	"github.com/Azure/azure-container-networking/log"
)

// AcnLogger :- struct to hold AcnLogger settings.
type AcnLogger struct {
	l *log.Logger
}

// NewAcnLogger :- Create new AcnLogger
func NewAcnLogger() *AcnLogger {
	acnLogger := new(AcnLogger)
	return acnLogger
}

// Init :- Initialize AcnLogger
func (l *AcnLogger) Init() {
	l.l = log.NewLogger("azure-dnc", log.LevelInfo, log.TargetLogfile)
}

// Log :- Log using AcnLogger
func (l *AcnLogger) Log(format string, args ...interface{}) {
	l.l.Printf(format, args...)
}

// Close :- Close AcnLogger
func (l *AcnLogger) Close() {
	l.l.Close()
}
