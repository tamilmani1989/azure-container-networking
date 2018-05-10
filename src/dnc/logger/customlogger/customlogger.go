package customlogger

import (
	"dnc/configuration"
	"fmt"

	"github.com/Azure/azure-container-networking/log"
)

// CustomLogger :- struct to hold CustomLogger settings.
type CustomLogger struct {
	name      string
	logLevel  int
	logTarget int
}

// NewCustomLogger :- Create new CustomLogger
func NewCustomLogger(cfg configuration.CustomLoggerSettings) *CustomLogger {
	customLogger := new(CustomLogger)
	customLogger.name = cfg.Name
	customLogger.logLevel = cfg.LogLevel
	customLogger.logTarget = cfg.LogTarget
	return customLogger
}

// Init :- Initialize CustomLogger
func (l *CustomLogger) Init() {
	log.SetName(l.name)
	log.SetLevel(l.logLevel)

	if err := log.SetTarget(l.logTarget); err != nil {
		fmt.Printf("Failed to configure custom logger: %s\n", err.Error())
		panic(err)
	}
}

// Log :- Log using CustomLogger
func (l *CustomLogger) Log(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Close :- Close CustomLogger
func (l *CustomLogger) Close() {
	log.Close()
}
