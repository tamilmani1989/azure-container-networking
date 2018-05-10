package main

import "os"

// ExitCallback - ExitCallback API signature
type ExitCallback func()

// ProcessExitMonitor - Monitor process exit and invokes the registered callback
func ProcessExitMonitor(
	osSignals chan os.Signal,
	exitCallback ExitCallback) {
	<-osSignals
	exitCallback()
	os.Exit(0)
}
