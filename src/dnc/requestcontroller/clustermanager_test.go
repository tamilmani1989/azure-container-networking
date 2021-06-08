package requestcontroller

import (
	"context"
	"dnc/retry"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

func TestManagerPoolAddRemove(t *testing.T) {
	managerPool := clusterManagerPool{
		clusterManagers: make(map[string]*ClusterManager),
		logger:          logrus.StandardLogger(),
	}

	// Add two managers to the pool
	mgr1 := &mockCtrlManager{}
	mgr2 := &mockCtrlManager{}
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr1,
		OrchestratorID:    "mgr1",
	}, mockClusterAuthConfig{})
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr2,
		OrchestratorID:    "mgr2",
	}, mockClusterAuthConfig{})

	// Add a nil manager to ensure we don't get a panic in case this ever happens
	managerPool.add(nil, nil)

	// Since managers are started in goroutine we can't guarantee that it is started by the time this
	// line is executed, so wrap it in a retrier.
	retrier := retry.Retrier{
		Attempts: 5,
		Delay:    time.Second,
	}

	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); !running {
			return errors.New("mgr1 is not running")
		}

		if running := mgr2.isRunning(); !running {
			return errors.New("mgr2 is not running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be running, but received: %v", err)
	}

	// stop the managers
	managerPool.remove("mgr1")
	managerPool.remove("mgr2")

	// Since managers are stopped by cancelling a context which happens asynchronously,
	// we can't guarantee that it is stopped by the time this line is executed, so wrap it in a retrier.
	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); running {
			return errors.New("mgr1 is running")
		}

		if running := mgr2.isRunning(); running {
			return errors.New("mgr2 is running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be stopped, but received: %v", err)
	}
}

func TestOnlyOneManagerPerOrchestratorWillRun(t *testing.T) {
	managerPool := clusterManagerPool{
		clusterManagers: make(map[string]*ClusterManager),
		logger:          logrus.StandardLogger(),
	}

	// Add two managers to the pool for the same orchestrator ID
	mgr1 := &mockCtrlManager{}
	mgr2 := &mockCtrlManager{}
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr1,
		OrchestratorID:    "orchestrator",
	}, mockClusterAuthConfig{})
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr2,
		OrchestratorID:    "orchestrator",
	}, mockClusterAuthConfig{})

	// Since managers are started in goroutine we can't guarantee that it is started by the time this
	// line is executed, but sleeping for a second should be safe.

	time.Sleep(time.Second)

	retrier := retry.Retrier{
		Attempts: 5,
		Delay:    time.Second,
	}

	if err := retrier.Do(context.Background(), func() error {
		// only one should be running, but we don't know which one
		mgr1Running := mgr1.isRunning()
		mgr2Running := mgr2.isRunning()

		if mgr1Running && mgr2Running {
			return errors.New("both managers are running")
		}

		if !mgr1Running && !mgr2Running {
			return errors.New("neither manager is running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected exactly one manager to be running, but received: %v", err)
	}

	// stop the managers
	managerPool.remove("orchestrator")

	// Since managers are stopped by cancelling a context which happens asynchronously,
	// we can't guarantee that it is stopped by the time this line is executed, so wrap it in a retrier.
	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); running {
			return errors.New("mgr1 is running")
		}

		if running := mgr2.isRunning(); running {
			return errors.New("mgr2 is running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be stopped, but received: %v", err)
	}
}

func TestManagerPoolRemoveAll(t *testing.T) {
	managerPool := clusterManagerPool{
		clusterManagers: make(map[string]*ClusterManager),
		logger:          logrus.StandardLogger(),
	}

	// Add two managers to the pool
	mgr1 := &mockCtrlManager{}
	mgr2 := &mockCtrlManager{}
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr1,
		OrchestratorID:    "mgr1",
	}, mockClusterAuthConfig{})
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr2,
		OrchestratorID:    "mgr2",
	}, mockClusterAuthConfig{})

	// Since managers are started in goroutine we can't guarantee that it is started by the time this
	// line is executed, so wrap it in a retrier.
	retrier := retry.Retrier{
		Attempts: 5,
		Delay:    time.Second,
	}

	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); !running {
			return errors.New("mgr1 is not running")
		}

		if running := mgr2.isRunning(); !running {
			return errors.New("mgr2 is not running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be running, but received: %v", err)
	}

	// stop the managers with remove all
	managerPool.removeAll()

	// Since managers are stopped by cancelling a context which happens asynchronously,
	// we can't guarantee that it is stopped by the time this line is executed, so wrap it in a retrier.
	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); running {
			return errors.New("mgr1 is running")
		}

		if running := mgr2.isRunning(); running {
			return errors.New("mgr2 is running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be stopped, but received: %v", err)
	}
}

type mockCtrlManager struct {
	running bool
	mutex   sync.Mutex
}

func (m *mockCtrlManager) Start(ctx context.Context) error {
	m.mutex.Lock()
	m.running = true
	m.mutex.Unlock()

	<-ctx.Done()

	m.mutex.Lock()
	m.running = false
	m.mutex.Unlock()

	return ctx.Err()
}

func (m *mockCtrlManager) isRunning() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.running
}

type mockFailingCtrlManager struct {
	running    bool
	failedOnce bool
	mutex      sync.Mutex
}

func (m *mockFailingCtrlManager) Start(ctx context.Context) error {
	m.mutex.Lock()
	if !m.failedOnce {
		m.failedOnce = true
		m.mutex.Unlock()
		return errors.New("mgr failed for some reason")
	}
	m.running = true
	m.mutex.Unlock()

	<-ctx.Done()

	m.mutex.Lock()
	m.running = false
	m.mutex.Unlock()

	return ctx.Err()
}

func (m *mockFailingCtrlManager) isRunning() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.running
}

func TestFailingManagersAreRestarted(t *testing.T) {
	managerPool := clusterManagerPool{
		clusterManagers: make(map[string]*ClusterManager),
		logger:          logrus.StandardLogger(),
	}

	// Add a failing manager to the pool
	mgr1 := &mockFailingCtrlManager{}
	managerPool.add(&ClusterManager{
		RunningContext:    NewCancellableContext(),
		ControllerManager: mgr1,
		OrchestratorID:    "mgr1",
	}, mockClusterAuthConfig{})

	// Since managers are started in goroutine, and this manager is eventually restarted after some backoff,
	// we can't guarantee that it is started by the time this line is executed, so wrap it in a retrier.
	retrier := retry.Retrier{
		Attempts: 5,
		Delay:    time.Second,
	}

	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); !running {
			return errors.New("mgr1 is not running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be running, but received: %v", err)
	}

	// stop the manager
	managerPool.removeAll()

	// Since managers are stopped by cancelling a context which happens asynchronously,
	// we can't guarantee that it is stopped by the time this line is executed, so wrap it in a retrier.
	if err := retrier.Do(context.Background(), func() error {
		if running := mgr1.isRunning(); running {
			return errors.New("mgr1 is running")
		}

		return nil
	}); err != nil {
		t.Fatalf("expected both managers to be stopped, but received: %v", err)
	}
}

func TestGetNextBackoff(t *testing.T) {
	var currentBackoff time.Duration = 0

	expected := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 16 * time.Second}
	got := make([]time.Duration, 6)

	for i := range got {
		currentBackoff = getNextBackoff(currentBackoff)
		got[i] = currentBackoff
	}

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

type mockClusterAuthConfig struct{}

func (mockClusterAuthConfig) Refresh(context.Context) error {
	return nil
}

func (mockClusterAuthConfig) RefreshForever(ctx context.Context) {
	<-ctx.Done()
}

func (mockClusterAuthConfig) GetConfig() *rest.Config {
	return &rest.Config{}
}
