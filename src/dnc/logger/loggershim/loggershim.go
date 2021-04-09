package loggershim

import (
	"dnc/configuration"
	"dnc/logger/acnlogger"
	"dnc/logger/customlogger"
	"fmt"

	"github.com/Azure/azure-container-networking/telemetry"
)

// LS :- Global reference to loggershim
var LS LoggerInterface

// LoggerInterface :- Logging interface
type LoggerInterface interface {
	Init(stderr bool, reports chan telemetry.DNCReport)
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Close()
	SetLogFileLimits(maxFileSize, maxFileCount int)
}

// LoggerShim :- Information to initialize azure-container-networking logger
type LoggerShim struct {
	loggerInterface LoggerInterface
	debugFunc       func(format string, args ...interface{})
	errorFunc       func(format string, args ...interface{})
}

// NewLoggerShim :- Create a new LoggerShim
func NewLoggerShim(cfg *configuration.Configuration) *LoggerShim {
	ls := new(LoggerShim)

	if cfg.CustomLoggerSettings != (configuration.CustomLoggerSettings{}) {
		ls.loggerInterface = customlogger.NewCustomLogger(cfg.CustomLoggerSettings)
	} else {
		ls.loggerInterface = acnlogger.NewAcnLogger(cfg)
	}

	LS = ls

	return ls
}

// Init :- Initialize logger
func (loggerShim *LoggerShim) Init(stderr bool, reports chan telemetry.DNCReport) {
	loggerShim.loggerInterface.Init(stderr, reports)
}

// Debugf :- Log debug statements using default (azure-container-networking) or custom logger
func (loggerShim *LoggerShim) Debugf(format string, args ...interface{}) {
	if loggerShim.debugFunc != nil {
		loggerShim.debugFunc(format, args...)
		return
	}
	loggerShim.loggerInterface.Debugf(format, args...)
}

// Errorf :- Log errors using default (azure-container-networking) or custom logger
func (loggerShim *LoggerShim) Errorf(format string, args ...interface{}) {
	if loggerShim.errorFunc != nil {
		loggerShim.errorFunc(format, args...)
		return
	}
	loggerShim.loggerInterface.Errorf(format, args...)
}

// Close :- Close logger
func (loggerShim *LoggerShim) Close() {
	loggerShim.loggerInterface.Close()
}

// SetLogFileLimits :- Set the limits for the log file
func (loggerShim *LoggerShim) SetLogFileLimits(maxFileSize, maxFileCount int) {
	loggerShim.loggerInterface.SetLogFileLimits(maxFileSize, maxFileCount)
}

// WithPrefix returns a LoggerInterface which logs using the provided prefix.
func WithPrefix(l LoggerInterface, prefix string) LoggerInterface {
	ls := LoggerShim{
		loggerInterface: l,
		debugFunc: func(format string, args ...interface{}) {
			l.Debugf("%s: %s", prefix, fmt.Sprintf(format, args...))
		},
		errorFunc: func(format string, args ...interface{}) {
			l.Errorf("%s: %s", prefix, fmt.Sprintf(format, args...))
		},
	}

	return &ls
}

// WithPrefixf returns a LoggerInterface which logs using the provided prefix format and args.
func WithPrefixf(l LoggerInterface, prefixFmt string, prefixArgs ...interface{}) LoggerInterface {
	prefix := fmt.Sprintf(prefixFmt, prefixArgs...)
	return WithPrefix(l, prefix)
}
