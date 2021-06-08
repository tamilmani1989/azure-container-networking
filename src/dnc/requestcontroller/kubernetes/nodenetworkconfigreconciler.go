// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/aiwrapper"
	"dnc/api"
	"dnc/requestcontroller"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns"
	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	processedNNCCountMetricName = "ProcessedNNCCount"
	orchestratorIDDimensionName = "OrchestratorID"
)

// NodeNetworkConfigReconciler is a controller-runtime/pkg/reconcile Reconciler for NodeNetworkConfigs
type NodeNetworkConfigReconciler struct {
	k8sClient                K8sClient
	dncClient                requestcontroller.SynchronousDNCClient
	scalerConfig             requestcontroller.RequestScalerConfig
	subnetInfoProvider       requestcontroller.SubnetInfoProvider
	logger                   logrus.FieldLogger
	nodeIDToNCDetails        *cache.Cache // map name of NNC to NC ID
	registeredNodes          *cache.Cache // Keep a map of nodes and ips to know what has already been registered on DNC
	registeredVnets          *cache.Cache // Keep a set of VNET GUIDs registered on DNC
	registeredSubnets        *cache.Cache // Keep a set of Subnets registered on DNC
	delegationGUIDToVnetGUID *cache.Cache // Keep a map of delegation guid to vnet guid
	reconcileTimeout         time.Duration
	registerVnetMutex        sync.Mutex
	registerSubnetMutex      sync.Mutex
	orchestratorID           string // used for publishing metrics
	skipSubnetJoin           bool
	metricsPublishers        []requestcontroller.MetricsPublisher
	recorder                 record.EventRecorder
}

// NodeNetworkConfigReconcilerOptions options to create NNC Reconciler
type NodeNetworkConfigReconcilerOptions struct {
	K8sClient                   K8sClient
	DNCClient                   requestcontroller.SynchronousDNCClient
	ScalerConfig                requestcontroller.RequestScalerConfig
	SubnetInfoProvider          requestcontroller.SubnetInfoProvider
	Logger                      logrus.FieldLogger
	ReconcileTimeout            time.Duration
	ProcessedNNCMetricsDuration time.Duration
	OrchestratorID              string
	SkipSubnetJoin              bool
	Recorder                    record.EventRecorder
}

type ncDetails struct {
	id       string
	vnetGUID string
}

// NewNodeNetworkConfigReconciler returns a new NNC Reconciler based on the provided options
func NewNodeNetworkConfigReconciler(opts NodeNetworkConfigReconcilerOptions) *NodeNetworkConfigReconciler {
	logger := opts.Logger.WithFields(logrus.Fields{
		"component": "NodeNetworkConfigReconciler",
	})

	// build caches
	nodeIDToNCDetails := cache.New(cache.NoExpiration, cache.NoExpiration)
	registeredNodes := cache.New(cache.NoExpiration, cache.NoExpiration)
	registeredVnets := cache.New(cache.NoExpiration, cache.NoExpiration)
	registeredSubnets := cache.New(cache.NoExpiration, cache.NoExpiration)
	delegationGUIDToVnetGUID := cache.New(cache.NoExpiration, cache.NoExpiration)

	return &NodeNetworkConfigReconciler{
		k8sClient:                opts.K8sClient,
		dncClient:                opts.DNCClient,
		scalerConfig:             opts.ScalerConfig,
		subnetInfoProvider:       opts.SubnetInfoProvider,
		logger:                   logger,
		nodeIDToNCDetails:        nodeIDToNCDetails,
		registeredNodes:          registeredNodes,
		registeredVnets:          registeredVnets,
		registeredSubnets:        registeredSubnets,
		delegationGUIDToVnetGUID: delegationGUIDToVnetGUID,
		reconcileTimeout:         opts.ReconcileTimeout,
		orchestratorID:           opts.OrchestratorID,
		skipSubnetJoin:           opts.SkipSubnetJoin,
		metricsPublishers: []requestcontroller.MetricsPublisher{
			&processedNNCMetricPublisher{
				registeredNodes: registeredNodes,
				orchestratorID:  opts.OrchestratorID,
				duration:        opts.ProcessedNNCMetricsDuration,
				logger:          logger,
			},
		},
		recorder: opts.Recorder,
	}
}

var _ reconcile.Reconciler = &NodeNetworkConfigReconciler{}

// Reconcile is called by controller-runtime when there is an add/update/delete event for a NodeNetworkConfig resource
func (n *NodeNetworkConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	name := request.NamespacedName.String()

	logger := n.logger.WithFields(logrus.Fields{
		"object": name,
	})

	logger.Infof("Reconciling %s", name)
	ctx := context.Background()
	// Add a timeout to the context if desired
	if n.reconcileTimeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, n.reconcileTimeout)
		defer cancel()
	}

	err := n.reconcile(ctx, request)
	if err != nil {
		logger.WithError(err).Infof("Reconcile %s failed with error: %v", name, err)
	} else {
		logger.Infof("Reconcile %s completed", name)
	}

	return reconcile.Result{}, err
}

// reconcile is wrapped by Reconcile to provide logging and context
func (n *NodeNetworkConfigReconciler) reconcile(ctx context.Context, request reconcile.Request) error {
	name := request.NamespacedName.Name

	// Get the NodeNetworkConfig from the cache
	nodeNetworkConfig := acnv1alpha.NodeNetworkConfig{}
	err := n.k8sClient.Get(ctx, request.NamespacedName, &nodeNetworkConfig)
	if err != nil {
		// Ignore NotFound since an immediate requeue won't solve it
		if apierrors.IsNotFound(err) {
			n.logger.Warnf("Error retrieving NNC %s: %v, this will not be retried.", name, err)
		} else {
			n.logger.Errorf("Error retrieving NNC %s: %v", name, err)
		}

		return client.IgnoreNotFound(err)
	}

	// Get the corresponding Node
	node := corev1.Node{}
	err = n.k8sClient.Get(ctx, client.ObjectKey{Name: name}, &node)
	if err != nil && !apierrors.IsNotFound(err) {
		// Requeue due to client error
		err = fmt.Errorf("error getting Node from kubernetes API server: %w", err)
		n.recorder.Event(&nodeNetworkConfig, corev1.EventTypeWarning, "ReconcileError", err.Error())
		return err
	}

	// If the corresponding Node isn't found or is being deleted, clean up this NNC
	if (err != nil && apierrors.IsNotFound(err)) || !node.DeletionTimestamp.IsZero() {
		err = n.cleanUpNNC(ctx, &nodeNetworkConfig, name)
		if err != nil {
			err = fmt.Errorf("error while cleaning up NNC: %w", err)
			n.recorder.Event(&nodeNetworkConfig, corev1.EventTypeWarning, "ReconcileError", err.Error())
		}
		return err
	}

	// Handle a create/update event
	if nodeNetworkConfig.DeletionTimestamp.IsZero() {
		// A zero deletionTimestamp means create/update event
		err := n.publishCreateOrUpdateNC(ctx, &nodeNetworkConfig, &node, name)
		if err != nil {
			err = fmt.Errorf("publishCreateOrUpdateNC error: %w", err)
		}
		return err
	}

	// Handle a deletion event
	if !nodeNetworkConfig.DeletionTimestamp.IsZero() {
		// The Node still exists, so we should let this NNC delete and then re-create it
		err := n.mustRecreateNNC(ctx, &nodeNetworkConfig, &node, name)
		if err != nil {
			return fmt.Errorf("recreate NNC error: %w", err)
		}
	}

	return nil
}

// cleanUpNNC should be called when the NNC is being deleted and the corresponding Node either doesn't exist, or is being deleted.
// This function will clean up the Node and NC on DNC, and remove the finalizer from the NNC so it can be removed from kubernetes api-server
func (n *NodeNetworkConfigReconciler) cleanUpNNC(ctx context.Context, nodeNetworkConfig *acnv1alpha.NodeNetworkConfig, nodeID string) error {
	// Delete NC from DNC if it exists
	ncInfo, err := n.getNCDetailsFromNodeID(ctx, nodeID)
	if err != nil {
		// Unexpected client error, so requeue
		return err
	}

	ncID := ncInfo.id

	// Delete NC if we have an NC ID to delete. Could be empty if there is no nc associated to this NNC.
	if ncID != "" {
		if err := n.dncClient.DeleteNC(ctx, ncInfo.vnetGUID, ncID); err != nil {
			// Could not clean up NC, so requeue
			return err
		}
	}

	// Remove NC from cache if present
	n.nodeIDToNCDetails.Delete(nodeID)

	// Delete Node from DNC
	err = n.dncClient.DeleteNode(ctx, nodeID)
	if err != nil {
		// Error deleting Node, so requeue
		return err
	}

	// - remove finalizer since we're done with DNC operations for this NNC
	if finalizers, removedFinalizer := removeStringFromSlice(nodeNetworkConfig.Finalizers, NNCFinalizerName); removedFinalizer {
		nodeNetworkConfig.Finalizers = finalizers
		err := n.k8sClient.Update(ctx, nodeNetworkConfig)
		if err != nil {
			// Couldn't remove finalizer from NNC, requeue
			return err
		}
	}

	// Delete the NNC from K8s if not already deleting
	if nodeNetworkConfig.DeletionTimestamp.IsZero() {
		err := n.k8sClient.Delete(ctx, nodeNetworkConfig)
		if err != nil {
			// K8s client error while deleting, requeue
			return err
		}
	}

	return nil
}

// mustRecreateNNC should be called when the NNC is being deleted but the corresponding Node exists and isn't being deleted.
// This function will remove the finalizer from the NNC to let the k8s api-server delete it, and then recreate it from a known good state.
func (n *NodeNetworkConfigReconciler) mustRecreateNNC(ctx context.Context, nodeNetworkConfig *acnv1alpha.NodeNetworkConfig, node *corev1.Node, nodeID string) error {
	name := nodeNetworkConfig.Name
	namespace := nodeNetworkConfig.Namespace

	// Remove the finalizer to allow k8s to finish deleting this NNC
	finalizers, _ := removeStringFromSlice(nodeNetworkConfig.Finalizers, NNCFinalizerName)
	nodeNetworkConfig.Finalizers = finalizers
	err := n.k8sClient.Update(ctx, nodeNetworkConfig)
	if err != nil {
		// Couldn't remove finalizer, so requeue
		return err
	}

	// If we crash at this point, the NodeReconciler will create the NNC on restart because the reconcilers share a manager, so no resources are leaked

	delegatedSubnetID, found := node.Labels[DelegatedSubnetIDLabel]
	if !found {
		return fmt.Errorf("label %s not found for node %s", DelegatedSubnetIDLabel, node.Name)
	}

	subnetName, found := node.Labels[SubnetNameLabel]
	if !found {
		return fmt.Errorf("label %s not found for node %s", SubnetNameLabel, node.Name)
	}

	// Get subnet info, this is a threadsafe call
	subnetInfo, err := n.subnetInfoProvider.GetSubnetInfo(ctx, delegatedSubnetID, subnetName)
	if err != nil {
		return err
	}

	// Recreate the NNC from the NC on DNC
	var ncResponse *api.NetworkContainerResponse
	ncInfo, _ := n.getNCDetailsFromNodeID(ctx, nodeID) // If theres an error, we need to recreate the NC from scratch, so ignore it
	ncID := ncInfo.id
	if ncID != "" {
		if ncResponse, err = n.dncClient.GetNC(ctx, subnetInfo.VnetID, ncID); err != nil {
			return err
		}
	}

	nnc := n.getNewNNCFromNCResponse(ncResponse, name, namespace, node)

	nncCreated := false
	for retry := 0; retry < 30; retry++ {
		err = n.k8sClient.Create(ctx, nnc)
		if err == nil {
			nncCreated = true
			break
		}
		time.Sleep(time.Second)
	}

	if !nncCreated {
		// If we get here, we're in a state where the NNC doesn't exist and we can't recreate it, but the DNC objects (Node and NC) still exist
		// The only safe thing we can do is exit and restart so the Node Reconciler can recreate the NNCs
		// TODO: make this safer so it doesn't depend on orchestrator to restart DNC-RC Pod, instead we should just restart manager
		n.logger.WithFields(logrus.Fields{
			"object": fmt.Sprintf("%s/%s", namespace, name),
		}).Panic("Cannot create NNC but DNC objects still exist. Exiting to trigger Reconcile loops for everything.")
	}

	return n.publishCreateOrUpdateNC(ctx, nnc, node, nodeID)
}

func (n *NodeNetworkConfigReconciler) publishCreateOrUpdateNC(ctx context.Context, nodeNetworkConfig *acnv1alpha.NodeNetworkConfig, node *corev1.Node, nodeID string) error {
	// Add finalizer before we start DNC operations if it is not present
	finalizers := appendStringIfNotPresent(nodeNetworkConfig.Finalizers, NNCFinalizerName)
	if len(finalizers) != len(nodeNetworkConfig.Finalizers) {
		nodeNetworkConfig.Finalizers = finalizers
		err := n.k8sClient.Update(ctx, nodeNetworkConfig)
		if err != nil {
			// K8s client error, requeue
			return err
		}
	}

	// Get the InternalIP of the Node
	var internalIPAddress string
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			internalIPAddress = address.Address
			break
		}
	}

	if internalIPAddress == "" {
		return fmt.Errorf("could not find InternalIP for Node %s", node.Name)
	}

	delegatedSubnetID, found := node.Labels[DelegatedSubnetIDLabel]
	if !found {
		return fmt.Errorf("label %s not found for node %s", DelegatedSubnetIDLabel, node.Name)
	}

	subnetName, found := node.Labels[SubnetNameLabel]
	if !found {
		return fmt.Errorf("label %s not found for node %s", SubnetNameLabel, node.Name)
	}

	// Get subnet info, this is a threadsafe call
	subnetInfo, err := n.subnetInfoProvider.GetSubnetInfo(ctx, delegatedSubnetID, subnetName)
	if err != nil {
		return err
	}

	infraVnetId, found := node.Labels[InfraVnetIdLabel]
	if !found {
		n.logger.WithFields(logrus.Fields{
			"vnetID": InfraVnetIdLabel,
			"Node":   node.Name,
		}).Infof("Label %s not found for node %s", InfraVnetIdLabel, node.Name)
		// For now SetInfraVnetId same as DelegatedSubnetId
		infraVnetId = subnetInfo.VnetID
	}

	// Store the delegationGUID -> vnetGUID mapping
	n.delegationGUIDToVnetGUID.Set(delegatedSubnetID, subnetInfo.VnetID, cache.NoExpiration)

	// Register the subnet/vnet if not already registered and not skipped
	if !n.skipSubnetJoin {
		if err := n.ensureSubnetIsRegistered(ctx, subnetInfo.VnetID, subnetName, subnetInfo.PrimaryToken); err != nil {
			return err
		}
	}

	// Register the Node if not already registered using the InfraVnetId
	if err := n.ensureNodeIsRegistered(ctx, nodeID, internalIPAddress, cns.KubernetesCRD, infraVnetId); err != nil {
		return err
	}

	// Create or Update the NC on DNC
	ncInfo, err := n.getNCDetailsFromNodeID(ctx, nodeID)
	if err != nil {
		return err
	}

	ncID := ncInfo.id

	if ncID == "" {
		// If there is no NC registered for the Node, then create a new one
		ncID = uuid.New().String()

		// UUID generation means first NC request for this NNC. Log time delta of NNC creation and NC creation
		// TODO: This should be a metric instead of a log line
		nncCreatedAt := nodeNetworkConfig.CreationTimestamp.Time
		ncCreationDuration := time.Since(nncCreatedAt)
		n.logger.WithFields(logrus.Fields{
			"NNC": nodeNetworkConfig.Name,
		}).Infof("Time to create NC request: %s", ncCreationDuration)
		n.recorder.Eventf(nodeNetworkConfig, corev1.EventTypeNormal, "CreatingNC", "Creating new NC %s for node %s", ncID, nodeID)

		// cache the nc id and vnet for later lookups
		n.nodeIDToNCDetails.Set(nodeID, ncDetails{
			id:       ncID,
			vnetGUID: subnetInfo.VnetID,
		}, cache.NoExpiration)
	}

	ncReq := api.NetworkContainerRequestV2{
		AllocationRequest: api.AllocationRequest{
			SubnetName:       subnetName,
			SecondaryIPCount: int(nodeNetworkConfig.Spec.RequestedIPCount),
		},
		AssociationInfo: api.AssociationInfo{
			NodeID:        nodeID,
			ContainerType: cns.Docker,
		},
		ToBeDeletedIPIds: nodeNetworkConfig.Spec.IPsNotInUse,
	}

	err = n.dncClient.UpsertNcV2(ctx, subnetInfo.VnetID, ncID, ncReq)
	if err != nil {
		//DNC client error, requeue
		if jsonErr, ok := err.(*api.JsonErr); ok {
			// don't push 500s because we don't know what if their contents is sensitive
			if jsonErr.HTTPStatus == http.StatusInternalServerError {
				n.recorder.Eventf(nodeNetworkConfig, corev1.EventTypeWarning, "ErrorUpdatingNC", "Failed to upsert NC %s, InternalServerError", ncID)
			} else {
				n.recorder.Eventf(nodeNetworkConfig, corev1.EventTypeWarning, "ErrorUpdatingNC", "Failed to upsert NC %s, err = %s", ncID, err)
			}
		}
		return err
	}

	ncResponse, err := n.dncClient.GetNC(ctx, subnetInfo.VnetID, ncID)
	if err != nil {
		// DNC client error, requeue
		if jsonErr, ok := err.(*api.JsonErr); ok {
			// don't push 500s because we don't know what if their contents is sensitive
			if jsonErr.HTTPStatus == http.StatusInternalServerError {
				n.recorder.Eventf(nodeNetworkConfig, corev1.EventTypeWarning, "ErrorGettingNC", "Failed to get NC %s, InternalServerError", ncID)
			} else {
				n.recorder.Eventf(nodeNetworkConfig, corev1.EventTypeWarning, "ErrorGettingNC", "Failed to get NC %s, err = %s", ncID, err)
			}
		}
		return err
	}

	n.recorder.Eventf(nodeNetworkConfig, corev1.EventTypeNormal, "UpdatedNC", "Published NC %s", ncID)

	// Publish k8s status
	status := n.getNNCStatusFromNCResponse(ncResponse, node)
	nodeNetworkConfig.Status = status
	err = n.k8sClient.Status().Update(ctx, nodeNetworkConfig)
	if err != nil {
		// Unable to write status, requeue
		return err
	}

	return nil
}

func (n *NodeNetworkConfigReconciler) ensureVnetIsRegistered(ctx context.Context, vnetID string) error {
	// Take lock on vnet so one goroutine registers and the rest wait
	// TODO: This lock will prevent concurrent vnet joins when there are multiple vnets
	n.registerVnetMutex.Lock()
	defer n.registerVnetMutex.Unlock()
	if _, found := n.registeredVnets.Get(vnetID); !found {
		n.logger.WithFields(logrus.Fields{
			"vnetID": vnetID,
		}).Infof("Vnet %s not found in local cache, registering on DNC", vnetID)
		err := n.dncClient.RegisterVnet(ctx, vnetID)
		if err != nil {
			return err
		}
		n.registeredVnets.Set(vnetID, nil, cache.NoExpiration)
	}

	return nil
}

func (n *NodeNetworkConfigReconciler) ensureSubnetIsRegistered(ctx context.Context, vnetID, subnetName, authToken string) error {
	/// First, register the vnet
	if err := n.ensureVnetIsRegistered(ctx, vnetID); err != nil {
		return err
	}

	// Second, register the subnet
	// Take lock on subnet so one goroutine registers and the rest wait
	n.registerSubnetMutex.Lock()
	defer n.registerSubnetMutex.Unlock()
	if _, found := n.registeredSubnets.Get(subnetName); !found {
		n.logger.WithFields(logrus.Fields{
			"vnetID":     vnetID,
			"subnetName": subnetName,
		}).Infof("Subnet %s in vnet %s not found in local cache, registering on DNC", subnetName, vnetID)
		err := n.dncClient.RegisterSubnet(ctx, vnetID, subnetName, authToken)
		if err != nil {
			return err
		}
		n.registeredSubnets.Set(subnetName, nil, cache.NoExpiration)
	}
	return nil
}

func (n *NodeNetworkConfigReconciler) ensureNodeIsRegistered(ctx context.Context, nodeID, ipAddress, orchestratorType, vnetGUID string) error {
	ipInCache, found := n.registeredNodes.Get(nodeID)
	ip := ""
	if found {
		ip = ipInCache.(string)
	}

	if !found || ip != ipAddress {
		n.logger.WithFields(logrus.Fields{
			"vnetID": vnetGUID,
			"nodeID": nodeID,
		}).Infof("Node %s not found in cache or IP doesn't match, registering on DNC", nodeID)
		err := n.dncClient.UpsertNode(ctx, nodeID, ipAddress, cns.KubernetesCRD, vnetGUID)
		if err != nil {
			// DNC client error, requeue
			return err
		}
		n.registeredNodes.Set(nodeID, ipAddress, cache.NoExpiration)
	}

	return nil
}

// get the NC for a given Node, either from cache or DNC. Caches the result if necessary. If the NC is not found, this returns and empty string and nil error.
// This function only returns an error when there is a DNC client errror, such as an http transport error
func (n *NodeNetworkConfigReconciler) getNCDetailsFromNodeID(ctx context.Context, nodeID string) (ncDetails, error) {
	if ncInfo, found := n.nodeIDToNCDetails.Get(nodeID); found {
		return ncInfo.(ncDetails), nil
	}

	// NC ID not found in cache (or was empty), try to query DNC for the node to get its NC (assuming one NC per node)
	nodeInfoResponse, err := n.dncClient.GetNode(ctx, nodeID)
	if err != nil {
		var respErr *api.JsonErr
		if errors.As(err, &respErr) {
			if respErr.HTTPStatus == http.StatusNotFound {
				// Node isn't found, so NC isn't found
				return ncDetails{}, nil
			}
		}
		// return any other api error
		return ncDetails{}, err
	}

	ncs := nodeInfoResponse.NetworkContainers
	if len(ncs) == 0 {
		// Node has no NCs, so NC not found
		return ncDetails{}, nil
	}

	if len(ncs) > 1 {
		// Node has too many NCs, so choose the first and log a warning.
		// This is a warning and not an error because requeuing will not solve the problem,
		// so instead we'll make a best effort of using what is available
		n.logger.WithFields(logrus.Fields{
			"nodeID":  nodeID,
			"ncCount": strconv.Itoa(len(ncs)),
		}).Warnf("WARN: Node %s should have 1 NC but has %d: %v", nodeID, len(ncs), ncs)
	}

	ncInfo := ncDetails{
		id:       ncs[0].NcID,
		vnetGUID: ncs[0].IPConfiguration.NetworkID,
	}

	// cache it for later lookups
	n.nodeIDToNCDetails.Set(nodeID, ncInfo, cache.NoExpiration)

	return ncInfo, nil
}

func (n *NodeNetworkConfigReconciler) getNewNNCFromNCResponse(nc *api.NetworkContainerResponse, name, namespace string, node *corev1.Node) *acnv1alpha.NodeNetworkConfig {
	// Start with the standard empty NNC
	return &acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				DelegatedSubnetIDLabel: nc.IPConfiguration.NetworkID, // TODO: will this always be the case?
				SubnetNameLabel:        nc.IPConfiguration.SubnetName,
			},
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: n.getBatchSize(*node),
		},
		Status: n.getNNCStatusFromNCResponse(nc, node),
	}
}

// getNNCStatusFromNCResponse returns a NNC Status representing the NC response
func (n *NodeNetworkConfigReconciler) getNNCStatusFromNCResponse(nc *api.NetworkContainerResponse, node *corev1.Node) acnv1alpha.NodeNetworkConfigStatus {
	nncStatus := acnv1alpha.NodeNetworkConfigStatus{}
	if nc == nil {
		return nncStatus
	}

	// Populate NetworkContainers list with nc response
	ipAssignments := make([]acnv1alpha.IPAssignment, len(nc.IPConfiguration.SecondaryIPs))
	index := 0
	for id, ipInfo := range nc.IPConfiguration.SecondaryIPs {
		ipAssignments[index] = acnv1alpha.IPAssignment{
			Name: id,
			IP:   strings.Split(ipInfo.IPAddress, "/")[0], //chop off the trailing /32
		}
		index++
	}
	nncStatus.NetworkContainers = []acnv1alpha.NetworkContainer{
		{
			ID:                 nc.NcID,
			PrimaryIP:          strings.Split(nc.IPConfiguration.IPAddress, "/")[0], //chop off the /32
			SubnetName:         nc.IPConfiguration.SubnetName,
			DefaultGateway:     nc.IPConfiguration.GatewayIP,
			SubnetAddressSpace: nc.IPConfiguration.IPSubnet,
			IPAssignments:      ipAssignments,
			Version:            int64(nc.NCVersion),
		},
	}

	// Populate scaler config with scaler config from n
	scaler := acnv1alpha.Scaler{
		BatchSize:               n.getBatchSize(*node),
		ReleaseThresholdPercent: n.scalerConfig.ReleaseThresholdPercent,
		RequestThresholdPercent: n.scalerConfig.RequestThresholdPercent,
		MaxIPCount:              n.getMaxIPCountOfNode(*node),
	}
	nncStatus.Scaler = scaler

	return nncStatus
}

// getMaxIPCountOfNode gets the max ip count for a node
// returns default max ip count if label is either not found or 0
func (n *NodeNetworkConfigReconciler) getMaxIPCountOfNode(node corev1.Node) int64 {
	// Get node capacity from node object
	// Returns 0 if label is not present
	nodeCapacity := node.Status.Capacity.Pods().Value()
	if nodeCapacity <= 0 {
		// if node capacity not present or < 0, set it to system max
		nodeCapacity = swiftIPCountSystemMax
	}

	// Get max desired swift pods from label
	// Start with system max, overwrite if label is found and parsed successfully
	var maxDesiredPods int64 = swiftIPCountSystemMax
	maxDesiredPodsString := node.Labels[MaxVnetPodsLabel]
	if parsedNum, err := strconv.ParseInt(maxDesiredPodsString, 10, 0); err == nil {
		// don't allow value <= 0
		if parsedNum > 0 {
			maxDesiredPods = parsedNum
		}
	}

	// Take the min of nodeCapacity, maxDesiredPods, and systemMax
	return minInt64(minInt64(nodeCapacity, maxDesiredPods), swiftIPCountSystemMax)
}

// getBatchSize gets the batch size
// If the batch size is > than maxIPCount, make batch size = maxIPCount
func (n *NodeNetworkConfigReconciler) getBatchSize(node corev1.Node) int64 {
	maxIPCount := n.getMaxIPCountOfNode(node)

	batchSize := n.scalerConfig.BatchSize
	if batchSize > maxIPCount {
		batchSize = maxIPCount
	}

	return batchSize
}

// Return all metrics publishers for this reconciler
func (n *NodeNetworkConfigReconciler) getMetricsPublishers() []requestcontroller.MetricsPublisher {
	return n.metricsPublishers
}

type processedNNCMetricPublisher struct {
	registeredNodes *cache.Cache
	orchestratorID  string
	duration        time.Duration
	logger          logrus.FieldLogger
}

// This is mainly a copy/paste of dnc's sendDatastoreMetric. When we revamp metrics, this should not use global, and should use contexts
func (p *processedNNCMetricPublisher) SendMetric() {
	registeredNodesCount := p.registeredNodes.ItemCount()

	p.logger.WithFields(logrus.Fields{
		"MetricName":     processedNNCCountMetricName,
		"Value":          fmt.Sprintf("%d", registeredNodesCount),
		"OrchestratorID": p.orchestratorID,
	}).Infof("Sending metric to AI")

	metric := aitelemetry.Metric{
		Name:             processedNNCCountMetricName,
		Value:            float64(p.registeredNodes.ItemCount()),
		CustomDimensions: make(map[string]string),
	}

	metric.CustomDimensions[orchestratorIDDimensionName] = p.orchestratorID
	go aiwrapper.SendMetric(metric)
}

func (p *processedNNCMetricPublisher) GetDuration() time.Duration {
	return p.duration
}

func minInt64(a, b int64) int64 {
	if a <= b {
		return a
	}

	return b
}
