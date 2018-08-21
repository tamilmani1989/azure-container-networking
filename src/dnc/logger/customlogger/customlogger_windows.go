package customlogger

import (
	"dnc/configuration"
	"dnc/netagentcontracts"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/Azure/azure-container-networking/log"
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
	reports      chan netagentcontracts.DNCReport
}

// NewCustomLogger :- Create new CustomLogger
func NewCustomLogger(cfg configuration.CustomLoggerSettings) *CustomLogger {
	customLogger := new(CustomLogger)
	customLogger.dllName = cfg.Dll
	customLogger.fnName = cfg.FnName
	return customLogger
}

// Init :- Initialize CustomLogger
func (l *CustomLogger) Init(stderr bool, reports chan netagentcontracts.DNCReport) {
	var err error
	l.dll, err = syscall.LoadLibrary(l.dllName)
	if err != nil {
		fmt.Printf("%s: %s\n", "Failed to load logging dll", err)
		os.Exit(1)
	}

	l.procEventLog, err = syscall.GetProcAddress(l.dll, l.fnName)
	if err != nil {
		fmt.Printf("%s: %s\n", "Failed to load logging function", err)
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

// Log :- Log using CustomLogger
func (l *CustomLogger) Debugf(format string, args ...interface{}) {
	syscall.Syscall(uintptr(l.procEventLog), uintptr(2), uintptr(4), convertStrToPass(fmt.Sprintf(format, args...)), 0)
	l.instance.Debugf(format, args...)
}

// Log :- Log using CustomLogger
func (l *CustomLogger) Errorf(format string, args ...interface{}) {
	syscall.Syscall(uintptr(l.procEventLog), uintptr(2), uintptr(1), convertStrToPass(fmt.Sprintf(format, args...)), 0)
	l.instance.Printf(format, args...)
	go func() {
		l.reports <- netagentcontracts.DNCReport{EventMessage: fmt.Sprintf(format, args...)}
	}()
}

// Close :- Close CustomLogger
func (l *CustomLogger) Close() {
	l.instance.Close()
}

func convertStrToPass(s string) uintptr {
	return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s)))
}
