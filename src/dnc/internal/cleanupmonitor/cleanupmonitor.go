package cleanupmonitor

import (
	"dnc/datamodel"
	"time"
)

type CleanupMonitor struct {
	pendingCleanupRepo datamodel.PendingResourceCleanupRepo
	IsPrimary          bool
}

func StartCleanupMonitor(monitor *CleanupMonitor) {

	// refine ticker time
	ticker := time.NewTicker(5 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if monitor.IsPrimary {
					monitor.ProcessResourceCleanup()
				}
			}
		}
	}()
}

func (c *CleanupMonitor) ProcessResourceCleanup() {

	// check pubsubproxy
	// check table
	// spawn appropriate jobs
	//cleanupRequests, _ := cleanupRepo.GetAllPendingCleanupRequests()
}
