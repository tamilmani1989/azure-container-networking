package main

import "os"

// ExitCallback - ExitCallback API signature
type ExitCallback func()

// ProcessExitMonitor - Monitor process exit and invokes the registered callback
func ProcessExitMonitor(
	osSignals chan os.Signal,
	exitCallback ExitCallback) {
	sig := <-osSignals
	logger.Debugf("received os signal %q. executing exit callback", sig.String())
	exitCallback()
	logger.Debugf("exit callback complete")
	os.Exit(0)
}
