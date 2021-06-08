package requestcontroller

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// maxBackoffSeconds is the maximum backoff between manager restarts
const maxBackoffSeconds = 16

type clusterManagerPool struct {
	clusterManagers map[string]*ClusterManager
	mutex           sync.Mutex
	logger          logrus.FieldLogger
}

// CancellableContext holds a context and its cancel func. While contexts are typically passed to functions to represent
// the scope of a request, in this case we are using a context to represent the scope of a manager. Since a manager
// implementation typically has multiple long-lived functions, we can derive their contexts from this one, and cancel
// this one to propagate the cancellation to all functions.
type CancellableContext struct {
	context.Context
	Cancel context.CancelFunc
}

type ClusterManager struct {
	RunningContext    CancellableContext
	ControllerManager ControllerManager
	OrchestratorID    string
}

// Manager wraps a controller-runtime/pkg/manager and provides some extra functionality
type ControllerManager interface {
	Start(context.Context) error
}

func (c *clusterManagerPool) has(orchestratorID string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, exists := c.clusterManagers[orchestratorID]
	return exists
}

// add adds a manager to the pool and starts it. It is an idempotent call - an existing manager is not restarted
// TODO: support kubeconfigs bytes change
func (c *clusterManagerPool) add(clusterManager *ClusterManager, clusterAuthConfig ClusterAuthConfig) {
	if clusterManager == nil {
		return
	}

	// take the pool mutex
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// add the manager to the pool if it is not present and start it
	if _, found := c.clusterManagers[clusterManager.OrchestratorID]; !found {
		c.clusterManagers[clusterManager.OrchestratorID] = clusterManager
		go c.runClusterManager(clusterManager, clusterAuthConfig)
	}
}

// remove stops a manager and removes it from the pool. This is an idempotent call.
func (c *clusterManagerPool) remove(orchestratorID string) {
	// take the pool mutex
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// stop and remove the manager from the pool if it exists
	if clusterManager, found := c.clusterManagers[orchestratorID]; found {
		clusterManager.stop()
		delete(c.clusterManagers, orchestratorID)
	}
}

// removeAll stops and removes all managers from the pool
func (c *clusterManagerPool) removeAll() {
	// take the pool mutex
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// stop and remove all managers
	for orchestratorID, clusterManager := range c.clusterManagers {
		clusterManager.stop()
		delete(c.clusterManagers, orchestratorID)
	}
}

// run runs the manager, restarting with exponential backoff if there is an error
func (c *clusterManagerPool) runClusterManager(clusterManager *ClusterManager, clusterAuthConfig ClusterAuthConfig) {
	errChan := make(chan error)
	defer close(errChan)

	var backoff time.Duration

	// keep the auth config updated while the manager is running
	go clusterAuthConfig.RefreshForever(clusterManager.RunningContext)

	for {
		// start the manager
		go func(errChan chan error) {
			c.logger.Infof("Starting controller for orchestrator [%s]", clusterManager.OrchestratorID)
			errChan <- clusterManager.ControllerManager.Start(clusterManager.RunningContext)
		}(errChan)

		// wait for exit or error
		err := <-errChan
		if err != nil {
			// since context cancellation is intended to be normal execution flow, don't log a cancellation error
			if !errors.Is(err, context.Canceled) {
				c.logger.Errorf("Tried to start manager for orchestrator %s, but received error: %v", clusterManager.OrchestratorID, err)
				backoff = getNextBackoff(backoff)
			}
		} else {
			// manager exited gracefully, so just return instead of restarting it
			return
		}

		// wait for exit or backoff
		select {
		case <-clusterManager.RunningContext.Done():
			// manager was stopped, so simply return
			return
		case <-time.After(backoff):
			continue
		}
	}
}

// stop stops the manager
func (c *ClusterManager) stop() {
	c.RunningContext.Cancel()
}

// getNextBackoff provides a helper for capped exponential backoff
func getNextBackoff(currentBackoff time.Duration) time.Duration {
	if currentBackoff == 0 {
		return time.Second
	}

	currentBackoff = 2 * currentBackoff
	if currentBackoff > maxBackoffSeconds*time.Second {
		currentBackoff = maxBackoffSeconds * time.Second
	}

	return currentBackoff
}

func NewCancellableContext() CancellableContext {
	ctx, cancel := context.WithCancel(context.Background())
	return CancellableContext{ctx, cancel}
}
