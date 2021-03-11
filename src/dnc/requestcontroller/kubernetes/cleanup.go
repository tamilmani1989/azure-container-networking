package kubernetes

import (
	"context"
	"dnc/api"
	"dnc/requestcontroller"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	defaultSignalCheckInterval = "30s"
)

const (
	cleanupPending = iota
	cleanupInProgress
)

type cleanupStatus int

type cleanupPool struct {
	items map[string]cleanupItem
	mutex sync.Mutex
}

type cleanupItem struct {
	cleanupStatus int
	nnc           acnv1alpha.NodeNetworkConfig
}

func newCleanupPool() *cleanupPool {
	return &cleanupPool{
		items: make(map[string]cleanupItem),
	}
}

func (c *cleanupPool) upsertItem(item cleanupItem) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items[item.nnc.Name] = item
}

func (c *cleanupPool) deleteItem(item cleanupItem) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, item.nnc.Name)
}

func (c *cleanupPool) upsertItems(items []cleanupItem) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, item := range items {
		c.items[item.nnc.Name] = item
	}
}

func (c *cleanupPool) getPendingItem() (*cleanupItem, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// return true if map is empty
	if len(c.items) == 0 {
		return nil, true
	}

	// find an item in pending state
	for key, item := range c.items {
		if item.cleanupStatus == cleanupPending {
			item.cleanupStatus = cleanupInProgress
			c.items[key] = item

			return &item, false
		}
	}

	// no items found in pending state, but map isn't empty, so return nil, false
	return nil, false
}

// Cleaner cleans a DNC partition and kubernetes cluster of all traces of SWIFT state
type Cleaner struct {
	kubeCoreV1Client     corev1.CoreV1Interface
	kubeRESTClient       *rest.RESTClient
	syncDNCClient        requestcontroller.SynchronousDNCClient
	logger               logrus.FieldLogger
	signalCheckInterval  time.Duration
	signal               CleanupSignal
	nncNamespace         string
	maxConcurrentThreads int
}

// CleanerOpts options to create Cleaner
type CleanerOpts struct {
	KubeCoreV1Client     corev1.CoreV1Interface
	KubeRESTClient       *rest.RESTClient
	SyncDNCClient        requestcontroller.SynchronousDNCClient
	Logger               logrus.FieldLogger
	SignalCheckInterval  time.Duration
	Signal               CleanupSignal
	NNCNamespace         string
	MaxConcurrentThreads int
}

// CleanupSignal configmap settings for cleanup signal
type CleanupSignal struct {
	ConfigMapName        string
	ConfigMapNamespace   string
	ConfigMapSafetyValue string
}

// NewCleaner creates a cleaner from the given options
func NewCleaner(opts CleanerOpts) *Cleaner {
	logger := opts.Logger.WithFields(logrus.Fields{
		"component": "Cleaner",
	})

	// Add default polling interval if not set
	if opts.SignalCheckInterval == 0 {
		duration, err := time.ParseDuration(defaultSignalCheckInterval)
		if err != nil {
			// Couldn't parse duration
			logger.WithError(err).Errorf("Could not parse default signal check duration")
			panic(err)
		}
		opts.SignalCheckInterval = duration
	}

	// Add default maxConcurrentThreads if not set
	if opts.MaxConcurrentThreads == 0 {
		opts.MaxConcurrentThreads = 100 // TODO: make this configurable
	}

	return &Cleaner{
		kubeCoreV1Client:     opts.KubeCoreV1Client,
		kubeRESTClient:       opts.KubeRESTClient,
		syncDNCClient:        opts.SyncDNCClient,
		logger:               logger,
		signalCheckInterval:  opts.SignalCheckInterval,
		signal:               opts.Signal,
		nncNamespace:         opts.NNCNamespace,
		maxConcurrentThreads: opts.MaxConcurrentThreads,
	}
}

// HasCleanupSignal checks the cluster for the cleanup signal and returns bool
func (c *Cleaner) HasCleanupSignal(ctx context.Context) bool {
	c.logger.WithFields(logrus.Fields{
		"configMapName":      c.signal.ConfigMapName,
		"configMapNamespace": c.signal.ConfigMapNamespace,
	}).Infof("Checking for cleanup signal")
	_, err := c.kubeCoreV1Client.ConfigMaps(c.signal.ConfigMapNamespace).Get(ctx, c.signal.ConfigMapName, metav1.GetOptions{})
	if err == nil {
		// If err is not nil, then cleanup signal is present
		return true
	}

	/* Check err to see if we're detecting properly. Nothing we can really do at this point if it is not a 404 but
	   we can at least log the error if it is anything else, in case of DRI investigation
	*/
	if !apierrors.IsNotFound(err) {
		c.logger.WithError(err).Warnf("Expected a 404 Not Found, but received a different error: %v", err)
	}

	return false
}

// WaitForCleanupSignal returns a channel that is closed when the cleanup signal is found or context is cancelled
func (c *Cleaner) WaitForCleanupSignal(ctx context.Context) chan struct{} {
	signalChan := make(chan struct{})

	go func() {
		defer close(signalChan)
		if c.HasCleanupSignal(ctx) {
			return
		}

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled, so exit
				return
			case <-time.After(c.signalCheckInterval):
				// Check for cleanup signal
				if c.HasCleanupSignal(ctx) {
					return
				}
			}
		}
	}()

	return signalChan
}

func (c *Cleaner) getAllNNCs(ctx context.Context) (acnv1alpha.NodeNetworkConfigList, error) {
	var nncs acnv1alpha.NodeNetworkConfigList

	err := c.kubeRESTClient.Get().
		Namespace(c.nncNamespace).
		Resource(nncPluralName).
		Do(ctx).
		Into(&nncs)

	return nncs, err
}

// Clean cleans the cluster and entire DNC partition
func (c *Cleaner) Clean(ctx context.Context) error {
	c.logger.Info("Cleanup signal found, cleaning.")
	defer c.logger.Info("Leaving cleanup")

	// For now, make a best effort to clean everything up.
	// TODO (matlong) Use DNC delete partition API for safer cleanup.

	// Get all NNCs
	nncs, err := c.getAllNNCs(ctx)
	if err != nil {
		return fmt.Errorf("Error getting all NNCs for cluster: %w", err)
	}

	c.logger.Infof("Cleaner found %d nncs to clean", len(nncs.Items))

	// Create a cleanup pool with all nncs set to cleanupPending
	cleanupPool := newCleanupPool()
	items := make([]cleanupItem, len(nncs.Items))
	for i, nnc := range nncs.Items {
		items[i] = cleanupItem{
			cleanupStatus: cleanupPending,
			nnc:           nnc,
		}
	}
	cleanupPool.upsertItems(items)

	// Get vnet for cluster from dnc
	vnetID, err := c.getNetworkID(ctx)
	if err != nil {
		// Error trying to get vnet, cleanup cannot proceed
		return err
	}

	if vnetID == "" {
		c.logger.Info("No vnet was found on DNC, but DNC did not return error. Thus, there is no DNC state to clean.")
	} else {
		c.logger.Infof("Found vnetID %s on DNC", vnetID)
	}

	// Clean up each NNC, spawning as many goroutines as there are config.maxConcurrentThreads
	var wg sync.WaitGroup
	for i := 0; i < c.maxConcurrentThreads; i++ {
		wg.Add(1)
		go c.runCleanerWorker(ctx, cleanupPool, &wg, vnetID, i)
	}

	// wait for NNCs to be cleaned up before proceeding with vnet deletion
	wg.Wait()

	// if vnetID is empty, there is nothing to delete on DNC. If it isn't empty, delete it.
	if vnetID != "" {
		c.logger.Infof("Cleaning up vnet %s on DNC", vnetID)
		// Delete Vnet
		if err := c.syncDNCClient.UnregisterVnet(ctx, vnetID); err != nil {
			return err
		}
	}

	err = c.reportCleanupDone(ctx)
	if err != nil {
		c.logger.WithError(err).Infof("Error reporting cleanup as done: %v", err)
	}

	return err
}

func (c *Cleaner) runCleanerWorker(ctx context.Context, pool *cleanupPool, wg *sync.WaitGroup, vnetID string, workerID int) {
	defer wg.Done()

	var delay time.Duration

	item, isEmpty := pool.getPendingItem()
	if isEmpty {
		// All NNCs have been cleaned up. Exit this worker.
		c.logger.Infof("status map is emtpy. Exiting worker. %d", workerID)
		return
	}

	// if item is nil, there is no work to do right now. Check again after a delay.
	if item == nil {
		c.logger.Infof("no work to do for this worker %d", workerID)
		delay = 5 * time.Second
	} else {
		// Item to clean exists, so clean it.
		c.logger.WithField("nnc", item.nnc.Name).Infof("Cleaning up nnc with worker %d", workerID)
		if err := c.cleanupNNC(ctx, item.nnc, vnetID); err != nil {
			// There was an error cleaning up this nnc. Put it back into pending state.
			c.logger.WithField("nnc", item.nnc.Name).WithError(err).Error("Error cleaning up NNC. Will retry.")
			item.cleanupStatus = cleanupPending
			pool.upsertItem(*item)
		} else {
			// cleanup was successful, so remove the NNC from the pool and move on
			pool.deleteItem(*item)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			// reset delay to 0 in case it was changed
			delay = 0
			item, isEmpty := pool.getPendingItem()
			if isEmpty {
				// All NNCs have been cleaned up. Exit this worker.
				c.logger.Infof("status map is emtpy. Exiting worker. %d", workerID)
				return
			}

			if item == nil {
				c.logger.Infof("no work to do for this worker %d", workerID)
				// There is no work for this worker to do right now, but that could change. Check again after a delay.
				delay = 5 * time.Second
				continue
			}

			if err := c.cleanupNNC(ctx, item.nnc, vnetID); err != nil {
				// There was an error cleaning up this nnc. Put it back into pending state.
				c.logger.WithField("nnc", item.nnc.Name).WithError(err).Error("Error cleaning up NNC, will retry")
				item.cleanupStatus = cleanupPending
				pool.upsertItem(*item)
				continue
			}

			// cleanup was successful, so remove the NNC from the map and move on
			pool.deleteItem(*item)
		}
	}
}

func (c *Cleaner) cleanupNNC(ctx context.Context, nnc acnv1alpha.NodeNetworkConfig, vnetID string) error {
	logger := c.logger.WithFields(logrus.Fields{
		"nnc": nnc.Name,
	})

	if vnetID != "" {
		// vnetID is only empty if there is no state to clean up on DNC. If there is state, clean it up.
		logger.Infof("Cleaning up DNC state for NNC %s", nnc.Name)

		// Delete the NCs from DNC
		ncs, err := c.getNCsForNNC(ctx, nnc)
		if err != nil {
			return err
		}

		for _, nc := range ncs {
			logger.WithFields(logrus.Fields{
				"nc": nc,
			}).Infof("Deleting NC %s from vnet %s", nc, vnetID)

			err := c.syncDNCClient.DeleteNC(ctx, vnetID, nc)
			if err != nil {
				return err
			}
		}

		logger.WithFields(logrus.Fields{
			"node": nnc.Name,
		}).Infof("Cleaning up Node %s", nnc.Name)

		// Delete the Node from DNC
		if err := c.syncDNCClient.DeleteNode(ctx, nnc.Name); err != nil {
			return err
		}
	}

	// Delete the NNC from the kube apiserver
	logger.WithFields(logrus.Fields{
		"nnc": nnc.Name,
	}).Infof("Cleaning up NNC %s", nnc.Name)

	// Get the latest NNC from the apiserver so we don't run into resourceVersion issues
	res := c.kubeRESTClient.Get().Resource(nncPluralName).Namespace(c.nncNamespace).Name(nnc.Name).Do(ctx)
	var statusCode int
	if res.StatusCode(&statusCode); statusCode == http.StatusNotFound {
		// NNC is already gone, nothing to do
		return nil
	}

	if err := res.Into(&nnc); err != nil {
		logger.WithError(err).Error("Unable to get latest NNC version from apiserver.")
		return err
	}

	// Remove finalizer from NNC if present so deletion can proceed on kube apiserver
	if containsString(nnc.Finalizers, NNCFinalizerName) {
		nnc.Finalizers = []string{}
		if err := c.kubeRESTClient.Put().
			Resource(nncPluralName).
			Namespace(c.nncNamespace).
			Name(nnc.Name).
			Body(&nnc).
			Do(ctx).
			Error(); err != nil {
			logger.WithError(err).Infof("Error removing finalizer from NNC %s: %v", nnc.Name, err)
			return err
		}
	}

	// Delete the NNC from the cluster
	if err := c.kubeRESTClient.Delete().
		Resource(nncPluralName).
		Namespace(c.nncNamespace).
		Name(nnc.Name).
		Do(ctx).
		Error(); err != nil {
		logger.WithError(err).Infof("Error cleaning up NNC %s: %v", nnc.Name, err)
		return err
	}

	return nil
}

func (c *Cleaner) reportCleanupDone(ctx context.Context) error {
	c.logger.Info("Reporting succesful cleanup")
	successData := map[string]string{
		"cleanup": "success",
		"key":     c.signal.ConfigMapSafetyValue,
	}

	configMap, err := c.kubeCoreV1Client.ConfigMaps(c.signal.ConfigMapNamespace).Get(ctx, c.signal.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		// If configmap isn't found, recreate it
		if apierrors.IsNotFound(err) {
			// This shouldn't happen, but if it does, this means that cx was probably messing with the configmap
			c.logger.Warn("Configmap not found, but cleanup is done. Recreating it to signal AKS-RP.")
			_, err = c.kubeCoreV1Client.ConfigMaps(c.signal.ConfigMapNamespace).Create(ctx, &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.signal.ConfigMapName,
					Namespace: c.signal.ConfigMapNamespace,
				},
				Data: successData,
			}, metav1.CreateOptions{})
		}

		return err
	}

	c.logger.WithFields(logrus.Fields{
		"configmapName":      configMap.Name,
		"configmapNamespace": configMap.Namespace,
	}).Infof("Reporting cleanup success to configmap %s in namespace %s", configMap.Name, configMap.Namespace)

	// Update the configmap to say that cleanup is done
	if reflect.DeepEqual(successData, configMap.Data) {
		// Cleanup already reported
		c.logger.Info("Cleanup already reported. Not updating configmap.")
		return nil
	}

	configMap.Data = successData

	_, err = c.kubeCoreV1Client.ConfigMaps(c.signal.ConfigMapNamespace).Update(ctx, configMap, metav1.UpdateOptions{})

	return err
}

// getNetworkID gets the network ID for the cluster. This assumes one vnet per cluster. Empty vnet and nil error means no vnet found.
func (c *Cleaner) getNetworkID(ctx context.Context) (string, error) {
	vnets, err := c.syncDNCClient.GetVnets(ctx)
	if err != nil {
		return "", err
	}

	var vnetID string

	if len(vnets.NetworkInfos) != 0 {
		vnetID = vnets.NetworkInfos[0].GUID
	}

	return vnetID, nil
}

func (c *Cleaner) getNCsForNNC(ctx context.Context, nnc acnv1alpha.NodeNetworkConfig) ([]string, error) {
	ncs := make(map[string]struct{})
	// get ncs from nnc status
	for _, nc := range nnc.Status.NetworkContainers {
		ncs[nc.ID] = struct{}{}
	}

	// get ncs from dnc getNode
	nodeInfoResponse, err := c.syncDNCClient.GetNode(ctx, nnc.Name)
	if ignoreNotFound(err) != nil {
		// Some unexpected error happened, so return it instead of proceeding
		return nil, err
	}

	// nodeInfoResponse might be nil if the node wasn't found, so check it
	if nodeInfoResponse != nil {
		// Get all the NCs from the node
		for _, ncResp := range nodeInfoResponse.NetworkContainers {
			ncs[ncResp.NcID] = struct{}{}
		}
	}

	// Put the NC IDs into a slice to return
	res := make([]string, len(ncs))
	i := 0
	for ncID := range ncs {
		res[i] = ncID
		i++
	}

	return res, nil

}

func ignoreNotFound(err error) error {
	var respErr *api.JsonErr
	if errors.As(err, &respErr) {
		if respErr.HTTPStatus == http.StatusNotFound {
			return nil
		}
	}

	return err
}
