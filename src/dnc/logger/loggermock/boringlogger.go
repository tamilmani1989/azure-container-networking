package loggermock

import (
	"log"

	"github.com/Azure/azure-container-networking/telemetry"
)

type boringlogger struct{}

// Newboringlogger fullfills the logging shim interface but just uses golangs log pakcage.
func NewBoringLogger() *boringlogger {
	return &boringlogger{}
}

func (d *boringlogger) Debugf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d *boringlogger) Errorf(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}

func (_ *boringlogger) Init(_ bool, _ chan telemetry.DNCReport) {}

func (_ *boringlogger) Close() {}

func (_ *boringlogger) SetLogFileLimits(_, _ int) {}
