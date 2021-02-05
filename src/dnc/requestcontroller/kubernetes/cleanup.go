package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	"reflect"
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

// Cleaner cleans a DNC partition and kubernetes cluster of all traces of SWIFT state
type Cleaner struct {
	kubeCoreV1Client    corev1.CoreV1Interface
	kubeRESTClient      *rest.RESTClient
	syncDNCClient       requestcontroller.SynchronousDNCClient
	logger              logrus.FieldLogger
	signalCheckInterval time.Duration
	signal              CleanupSignal
	nncNamespace        string
}

// CleanerOpts options to create Cleaner
type CleanerOpts struct {
	KubeCoreV1Client    corev1.CoreV1Interface
	KubeRESTClient      *rest.RESTClient
	SyncDNCClient       requestcontroller.SynchronousDNCClient
	Logger              logrus.FieldLogger
	SignalCheckInterval time.Duration
	Signal              CleanupSignal
	NNCNamespace        string
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

	return &Cleaner{
		kubeCoreV1Client:    opts.KubeCoreV1Client,
		kubeRESTClient:      opts.KubeRESTClient,
		syncDNCClient:       opts.SyncDNCClient,
		logger:              logger,
		signalCheckInterval: opts.SignalCheckInterval,
		signal:              opts.Signal,
		nncNamespace:        opts.NNCNamespace,
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

// Clean cleans the cluster and entire DNC partition
func (c *Cleaner) Clean(ctx context.Context) error {
	// For now, make a best effort to clean everything up.
	// TODO (matlong) Use DNC delete partition API for safer cleanup.

	// Get all NNCs
	c.logger.Info("Cleanup signal found, cleaning.")
	defer c.logger.Info("Leaving cleanup")
	var nncList acnv1alpha.NodeNetworkConfigList
	err := c.kubeRESTClient.Get().
		Namespace(c.nncNamespace).
		Resource(nncPluralName).
		Do(ctx).
		Into(&nncList)

	if err != nil {
		// empty collection still returns 200 (nil err), so an error here means something went wrong and cleanup cannot proceed
		return err
	}

	// Get vnet for cluster from dnc
	vnetID, err := c.getNetworkID(ctx)
	if err != nil {
		// Error trying to get vnet, cleanup cannot proceed
		return err
	}

	// Clean up each NNC
	for _, nnc := range nncList.Items {
		logger := c.logger.WithFields(logrus.Fields{
			"nnc": nnc.Name,
		})

		if vnetID != "" {
			// vnetID is only empty if there is no state to clean up on DNC. If there is state, clean it up.
			logger.Infof("Cleaning up DNC state for NNC %s", nnc.Name)

			// Delete the NCs from DNC
			for _, nc := range nnc.Status.NetworkContainers {
				logger.WithFields(logrus.Fields{
					"nc": nc.ID,
				}).Infof("Deleting NC %s from vnet %s", nc.ID, vnetID)

				err := c.syncDNCClient.DeleteNC(ctx, vnetID, nc.ID)
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

		// Remove finalizer from NNC if present so deletion can proceed on kube apiserver
		if containsString(nnc.Finalizers, NNCFinalizerName) {
			nnc.Finalizers = []string{}
			err = c.kubeRESTClient.Put().
				Resource(nncPluralName).
				Namespace(c.nncNamespace).
				Name(nnc.Name).
				Body(&nnc).
				Do(ctx).
				Error()

			if err != nil {
				logger.WithError(err).Infof("Error removing finalizer from NNC %s: %v", nnc.Name, err)
				return err
			}
		}

		// Delete the NNC from the cluster
		err = c.kubeRESTClient.Delete().
			Resource(nncPluralName).
			Namespace(c.nncNamespace).
			Name(nnc.Name).
			Do(ctx).
			Error()

		if err != nil {
			logger.WithError(err).Infof("Error cleaning up NNC %s: %v", nnc.Name, err)
			return err
		}
	}

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
