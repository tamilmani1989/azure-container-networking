package simplejson

import (
	"dnc/logger/loggershim"
	"io"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-container-networking/telemetry"
)

type simpleJSON struct {
	entry *logrus.Entry
}

type ExpandableLogger interface {
	loggershim.LoggerInterface
	WithField(key, value string) ExpandableLogger
	io.Writer
}

// using logrus to try and get json out to be captured in AKS svc-cluster
func NewLogger(eventsource string) ExpandableLogger {
	logger := logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.JSONFormatter),
		Level:     logrus.DebugLevel,
	}

	return &simpleJSON{logger.WithField("source", eventsource)}
}

func (d *simpleJSON) Debugf(format string, args ...interface{}) {
	d.entry.Debugf(format, args...)
}

func (d *simpleJSON) Errorf(format string, args ...interface{}) {
	d.entry.Errorf(format, args...)
}

func (_ *simpleJSON) Init(_ bool, _ chan telemetry.DNCReport) {}

func (_ *simpleJSON) Close() {}

func (_ *simpleJSON) SetLogFileLimits(_, _ int) {}

func (d *simpleJSON) Write(p []byte) (n int, err error) {
	d.entry.Debugf(string(p))
	return len(p), nil
}

func (d *simpleJSON) WithField(key, value string) ExpandableLogger {
	return &simpleJSON{d.entry.WithField(key, value)}
}
