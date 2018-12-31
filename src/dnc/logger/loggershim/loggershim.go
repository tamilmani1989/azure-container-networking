package loggershim

import (
	"dnc/configuration"
	"dnc/logger/acnlogger"
	"dnc/logger/customlogger"

	"github.com/Azure/azure-container-networking/telemetry"
)

// LS :- Global reference to loggershim
var LS *LoggerShim

// LoggerInterface :- Logging interface
type LoggerInterface interface {
	Init(stderr bool, reports chan telemetry.DNCReport)
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Close()
}

// LoggerShim :- Information to initialize azure-container-networking logger
type LoggerShim struct {
	loggerSettings  configuration.CustomLoggerSettings
	loggerInterface LoggerInterface
}

// NewLoggerShim :- Create a new LoggerShim
func NewLoggerShim(cfg *configuration.Configuration) *LoggerShim {
	LS = new(LoggerShim)
	LS.loggerSettings = cfg.CustomLoggerSettings
	if LS.loggerSettings != (configuration.CustomLoggerSettings{}) {
		LS.loggerInterface = customlogger.NewCustomLogger(cfg.CustomLoggerSettings)
	} else {
		LS.loggerInterface = acnlogger.NewAcnLogger()
	}

	return LS
}

// Init :- Initialize logger
func (loggerShim *LoggerShim) Init(stderr bool, reports chan telemetry.DNCReport) {
	loggerShim.loggerInterface.Init(stderr, reports)
}

// Debugf :- Log debug statements using default (azure-container-networking) or custom logger
func (loggerShim *LoggerShim) Debugf(format string, args ...interface{}) {
	loggerShim.loggerInterface.Debugf(format, args...)
}

// Errorf :- Log errors using default (azure-container-networking) or custom logger
func (loggerShim *LoggerShim) Errorf(format string, args ...interface{}) {
	loggerShim.loggerInterface.Errorf(format, args...)
}

// Close :- Close logger
func (loggerShim *LoggerShim) Close() {
	loggerShim.loggerInterface.Close()
}
