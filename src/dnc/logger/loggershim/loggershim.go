package loggershim

import (
	"dnc/configuration"
	"dnc/logger/acnlogger"
	"dnc/logger/customlogger"
)

// LS :- Global reference to loggershim
var LS *LoggerShim

// LoggerInterface :- Logging interface
type LoggerInterface interface {
	Init(debug, stderr bool)
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Close()
}

// LoggerShim :- Information to initialize azure-container-networking logger
type LoggerShim struct {
	loggerSettings  configuration.LoggerSettings
	loggerInterface LoggerInterface
}

// NewLoggerShim :- Create a new LoggerShim
func NewLoggerShim(cfg *configuration.Configuration) *LoggerShim {
	LS = new(LoggerShim)
	LS.loggerSettings = cfg.LoggerSettings

	if LS.loggerSettings.CustomLoggerSettings != (configuration.CustomLoggerSettings{}) {
		LS.loggerInterface = customlogger.NewCustomLogger(cfg.LoggerSettings.CustomLoggerSettings)
	} else {
		LS.loggerInterface = acnlogger.NewAcnLogger()
	}

	return LS
}

// Init :- Initialize logger
func (loggerShim *LoggerShim) Init(debug, stderr bool) {
	loggerShim.loggerInterface.Init(debug, stderr)
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
