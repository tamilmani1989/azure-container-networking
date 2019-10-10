package customlogger

import (
	"dnc/configuration"
	"dnc/logger"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/telemetry"
)

// CustomLogger :- struct to hold CustomLogger settings.
type CustomLogger struct {
	procEventLog uintptr
	logLevel     int
	logTarget    int
	dllName      string
	fnName       string
	dll          syscall.Handle
	instance     *log.Logger
	reports      chan telemetry.DNCReport
}

// NewCustomLogger :- Create new CustomLogger
func NewCustomLogger(cfg configuration.CustomLoggerSettings) *CustomLogger {
	customLogger := new(CustomLogger)
	customLogger.dllName = cfg.Dll
	customLogger.fnName = cfg.FnName
	return customLogger
}

// Init :- Initialize CustomLogger
func (l *CustomLogger) Init(stderr bool, reports chan telemetry.DNCReport) {
	var err error
	l.dll, err = syscall.LoadLibrary(l.dllName)
	if err != nil {
		fmt.Printf("%s: %s", "Failed to load logging dll", err)
		os.Exit(1)
	}

	l.procEventLog, err = syscall.GetProcAddress(l.dll, l.fnName)
	if err != nil {
		fmt.Printf("%s: %s", "Failed to load logging function", err)
		os.Exit(1)
	}

	if stderr {
		l.logTarget = log.TargetStdOutAndLogFile
	} else {
		l.logTarget = log.TargetLogfile
	}

	l.logLevel = log.LevelDebug
	l.instance = log.NewLogger("azure-dnc", l.logLevel, l.logTarget)
	l.reports = reports
}

// Debugf :- Log debug statements using CustomLogger
func (l *CustomLogger) Debugf(format string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			l.instance.Printf("[Azure DNC] Recovered from crash due to %+v", r)
		}
	}()

	l.instance.Debugf(format, args...)
	if l.reports != nil {
		go func() {
			l.reports <- telemetry.DNCReport{EventMessage: fmt.Sprintf(format, args...)}
		}()
	}

	if ptr, err := convertStrToPass(fmt.Sprintf(format, args...)); err == nil {
		syscall.Syscall(uintptr(l.procEventLog), uintptr(2), uintptr(4), ptr, 0)
	}
}

// Errorf :- Log errors using CustomLogger
func (l *CustomLogger) Errorf(format string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			l.instance.Printf("[Azure DNC] Recovered from crash due to %+v", r)
		}
	}()

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
		}()
	}

	if ptr, err := convertStrToPass(fmt.Sprintf(format, args...)); err == nil {
		syscall.Syscall(uintptr(l.procEventLog), uintptr(2), uintptr(1), ptr, 0)
	}
}

// Close :- Close CustomLogger
func (l *CustomLogger) Close() {
	l.instance.Close()
}

func convertStrToPass(s string) (uintptr, error) {
	arr, err := syscall.UTF16FromString(s)
	if err != nil {
		return 0, err
	}

	return uintptr(unsafe.Pointer(&arr[0])), nil
}

// SetLogFileLimits :- Set the limits for the log file
func (l *CustomLogger) SetLogFileLimits(maxFileSize int, maxFileCount int) {
	l.instance.SetLogFileLimits(maxFileSize, maxFileCount)
}
