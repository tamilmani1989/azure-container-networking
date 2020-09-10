// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/api"
	"dnc/requestcontroller"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NodeNetworkConfigReconciler is a controller-runtime/pkg/reconcile Reconciler for NodeNetworkConfigs
type NodeNetworkConfigReconciler struct {
	Client              K8sClient
	DncClient           requestcontroller.SynchronousDNCClient
	ScalerConfig        requestcontroller.RequestScalerConfig
	SubnetInfoProvider  requestcontroller.SubnetInfoProvider
	NodeIDToNCID        *cache.Cache  // map name of NNC to NC ID
	RegisteredNodes     *cache.Cache  // Keep a map of nodes and ips to know what has already been registered on DNC
	RegisteredVnets     *cache.Cache  // Keep a set of VNET GUIDs registered on DNC
	RegisteredSubnets   *cache.Cache  // Keep a set of Subnets registered on DNC
	ReconcileTimeout    time.Duration // TODO: derive a context from this and pass to all functions where appropriate
	vnetGUID            string        // TODO: make this a NC ID -> VnetGUID map to support multiple Vnets
	registerVnetMutex   sync.Mutex
	registerSubnetMutex sync.Mutex
}

var _ reconcile.Reconciler = &NodeNetworkConfigReconciler{}

// Reconcile is called by controller-runtime when there is an add/update/delete event for a NodeNetworkConfig resource
func (n *NodeNetworkConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	name := request.NamespacedName.Name
	log.Printf("Reconciling %s", name)

	// Get the NodeNetworkConfig from the cache
	nodeNetworkConfig := acnv1alpha.NodeNetworkConfig{}
	err := n.Client.Get(context.TODO(), request.NamespacedName, &nodeNetworkConfig)
	if err != nil {
		log.Printf("NodeNetworkConfigReconciler error retrieving NNC %s: %v", name, err)
		// Ignore NotFound since an immediate requeue won't solve it
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Get the corresponding Node
	node := corev1.Node{}
	err = n.Client.Get(context.TODO(), client.ObjectKey{Name: name}, &node)
	if err != nil && !apierrors.IsNotFound(err) {
		// Requeue due to client error
		log.Printf("NodeNetworkConfigReconciler error: %v", err)
		return reconcile.Result{}, err
	}

	// If the corresponding Node isn't found or is being deleted, clean up this NNC
	if (err != nil && apierrors.IsNotFound(err)) || !node.DeletionTimestamp.IsZero() {
		err := n.cleanUpNNC(&nodeNetworkConfig, name)
		if err != nil {
			log.Printf("NodeNetworkConfigReconciler error while cleaning up NNC: %v", err)
		}
		return reconcile.Result{}, err
	}

	// Handle a create/update event
	if nodeNetworkConfig.DeletionTimestamp.IsZero() {
		// A zero deletionTimestamp means create/update event
		err := n.publishCreateOrUpdateNC(&nodeNetworkConfig, &node, name)
		if err != nil {
			log.Printf("NodeNetworkConfigReconciler publishCreateOrUpdateNC error: %v", err)
		}
		return reconcile.Result{}, err
	}

	// Handle a deletion event
	if !nodeNetworkConfig.DeletionTimestamp.IsZero() {
		// The Node still exists, so we should let this NNC delete and then re-create it
		err := n.mustRecreateNNC(&nodeNetworkConfig, &node, name)
		if err != nil {
			log.Printf("NodeNetworkConfigReconciler recreate NNC error: %v", err)
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// cleanUpNNC should be called when the NNC is being deleted and the corresponding Node either doesn't exist, or is being deleted.
// This function will clean up the Node and NC on DNC, and remove the finalizer from the NNC so it can be removed from kubernetes api-server
func (n *NodeNetworkConfigReconciler) cleanUpNNC(nodeNetworkConfig *acnv1alpha.NodeNetworkConfig, nodeID string) error {
	// Delete NC from DNC if it exists
	ncID, err := n.getNCIDFromNodeID(nodeID)
	if err != nil {
		// Unexpected client error, so requeue
		return err
	}

	// Remove NC from cache if present
	n.NodeIDToNCID.Delete(nodeID)

	err = n.DncClient.DeleteNC(context.TODO(), n.vnetGUID, ncID)
	if err != nil {
		// Could not clean up NC, so requeue
		return err
	}

	// Delete Node from DNC
	err = n.DncClient.DeleteNode(context.TODO(), nodeID)
	if err != nil {
		// Error deleting Node, so requeue
		return err
	}

	// - remove finalizer since we're done with DNC operations for this NNC
	if finalizers, removedFinalizer := removeStringFromSlice(nodeNetworkConfig.Finalizers, NNCFinalizerName); removedFinalizer {
		nodeNetworkConfig.Finalizers = finalizers
		err := n.Client.Update(context.TODO(), nodeNetworkConfig)
		if err != nil {
			// Couldn't remove finalizer from NNC, requeue
			return err
		}
	}

	// Delete the NNC from K8s if not already deleting
	if nodeNetworkConfig.DeletionTimestamp.IsZero() {
		err := n.Client.Delete(context.TODO(), nodeNetworkConfig)
		if err != nil {
			// K8s client error while deleting, requeue
			return err
		}
	}

	return nil
}

// mustRecreateNNC should be called when the NNC is being deleted but the corresponding Node exists and isn't being deleted.
// This function will remove the finalizer from the NNC to let the k8s api-server delete it, and then recreate it from a known good state.
func (n *NodeNetworkConfigReconciler) mustRecreateNNC(nodeNetworkConfig *acnv1alpha.NodeNetworkConfig, node *corev1.Node, nodeID string) error {
	name := nodeNetworkConfig.Name
	namespace := nodeNetworkConfig.Namespace

	// Remove the finalizer to allow k8s to finish deleting this NNC
	finalizers, _ := removeStringFromSlice(nodeNetworkConfig.Finalizers, NNCFinalizerName)
	nodeNetworkConfig.Finalizers = finalizers
	err := n.Client.Update(context.TODO(), nodeNetworkConfig)
	if err != nil {
		// Couldn't remove finalizer, so requeue
		return err
	}

	// If we crash at this point, the NodeReconciler will create the NNC on restart because the reconcilers share a manager, so no resources are leaked

	// Recreate the NNC from the NC on DNC
	var ncResponse *api.NetworkContainerResponse
	ncID, _ := n.getNCIDFromNodeID(nodeID) // If theres an error, we need to recreate the NC from scratch, so ignore it
	if ncID != "" {
		ncResponse, _ = n.DncClient.GetNC(context.TODO(), n.vnetGUID, ncID)
	}

	nnc := n.getNewNNCFromNCResponse(ncResponse, name, namespace)

	nncCreated := false
	for retry := 0; retry < 30; retry++ {
		err = n.Client.Create(context.TODO(), nnc)
		if err == nil {
			nncCreated = true
			break
		}
		time.Sleep(time.Second)
	}

	if !nncCreated {
		// If we get here, we're in a state where the NNC doesn't exist and we can't recreate it, but the DNC objects (Node and NC) still exist
		// The only safe thing we can do is exit and restart so the Node Reconciler can recreate the NNCs
		log.Printf("Cannot create NNC but DNC objects still exist. Exiting to trigger Reconcile loops for everything.")
		panic("Could not create NNC in mustRecreateNNC")
	}

	return n.publishCreateOrUpdateNC(nnc, node, nodeID)
}

func (n *NodeNetworkConfigReconciler) publishCreateOrUpdateNC(nodeNetworkConfig *acnv1alpha.NodeNetworkConfig, node *corev1.Node, nodeID string) error {
	// Add finalizer before we start DNC operations if it is not present
	finalizers := appendStringIfNotPresent(nodeNetworkConfig.Finalizers, NNCFinalizerName)
	if len(finalizers) != len(nodeNetworkConfig.Finalizers) {
		nodeNetworkConfig.Finalizers = finalizers
		err := n.Client.Update(context.TODO(), nodeNetworkConfig)
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
		return fmt.Errorf("Could not find InternalIP for Node %s", node.Name)
	}

	delegatedSubnetID, found := node.Labels[DelegatedSubnetIDLabel]
	if !found {
		return fmt.Errorf("Label %s not found for node %s", DelegatedSubnetIDLabel, node.Name)
	}

	subnetName, found := node.Labels[SubnetNameLabel]
	if !found {
		return fmt.Errorf("Label %s not found for node %s", SubnetNameLabel, node.Name)
	}

	// Get subnet info, this is a threadsafe call
	subnetInfo, err := n.SubnetInfoProvider.GetSubnetInfo(context.TODO(), delegatedSubnetID, subnetName)
	if err != nil {
		return err
	}

	// Register the vnet if not already registered
	if err := n.ensureVnetIsRegistered(context.TODO(), subnetInfo.VnetID); err != nil {
		return err
	}

	//Register the subnet if not already registered
	if err := n.ensureSubnetIsRegistered(context.TODO(), subnetInfo.VnetID, subnetName, subnetInfo.PrimaryToken); err != nil {
		return err
	}

	err = n.DncClient.UpsertNode(context.TODO(), nodeID, internalIPAddress, cns.KubernetesCRD, n.vnetGUID)
	if err != nil {
		// DNC client error, requeue
		return err
	}

	// Create or Update the NC on DNC
	ncID, err := n.getNCIDFromNodeID(nodeID)
	if err != nil {
		return err
	}

	if ncID == "" {
		// If there is no NC registered for the Node, then create a new one
		ncID = uuid.New().String()
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

	err = n.DncClient.UpsertNcV2(context.TODO(), n.vnetGUID, ncID, ncReq)
	if err != nil {
		//DNC client error, requeue
		return err
	}

	ncResponse, err := n.DncClient.GetNC(context.TODO(), n.vnetGUID, ncID)

	// Publish k8s status
	status := n.getNNCStatusFromNCResponse(ncResponse)
	nodeNetworkConfig.Status = status
	err = n.Client.Status().Update(context.TODO(), nodeNetworkConfig)
	if err != nil {
		// Unable to write status, requeue
		return err
	}

	return nil
}

func (n *NodeNetworkConfigReconciler) ensureVnetIsRegistered(ctx context.Context, vnetID string) error {
	// Take lock on vnet so one goroutine registers and the rest wait
	n.registerVnetMutex.Lock()
	defer n.registerVnetMutex.Unlock()
	n.vnetGUID = vnetID
	if _, found := n.RegisteredVnets.Get(n.vnetGUID); !found {
		err := n.DncClient.RegisterVnet(ctx, n.vnetGUID)
		if err != nil {
			return err
		}
		n.RegisteredVnets.Set(n.vnetGUID, nil, cache.NoExpiration)
	}

	return nil
}

func (n *NodeNetworkConfigReconciler) ensureSubnetIsRegistered(ctx context.Context, vnetID, subnetName, authToken string) error {
	// Take lock on subnet so one goroutine registers and the rest wait
	n.registerSubnetMutex.Lock()
	defer n.registerSubnetMutex.Unlock()
	if _, found := n.RegisteredSubnets.Get(subnetName); !found {
		err := n.DncClient.RegisterSubnet(context.TODO(), vnetID, subnetName, authToken)
		if err != nil {
			return err
		}
		n.RegisteredSubnets.Set(subnetName, nil, cache.NoExpiration)
	}
	return nil
}

// get the NC for a given Node, either from cache or DNC. Caches the result if necessary. If the NC is not found, this returns and empty string and nil error.
// This function only returns an error when there is a DNC client errror, such as an http transport error
func (n *NodeNetworkConfigReconciler) getNCIDFromNodeID(nodeID string) (string, error) {
	ncIDCached, foundInCache := n.NodeIDToNCID.Get(nodeID)
	ncID := ""
	if foundInCache {
		ncID = ncIDCached.(string)
	} else {
		// NC ID not found in cache, try to query DNC for the node to get its NC (assuming one NC per node)
		nodeInfoResponse, err := n.DncClient.GetNode(context.TODO(), nodeID)
		if err != nil {
			var respErr *api.JsonErr
			if errors.As(err, &respErr) {
				if respErr.Code == api.NodeNotFound {
					// Node isn't found, so NC isn't found
					return "", nil
				}
			}
			// return any other api error
			return "", err
		}

		ncs := nodeInfoResponse.NetworkContainers
		if len(ncs) == 0 {
			// Node has no NCs, so NC not found
			return "", nil
		} else if len(ncs) > 1 {
			/* Node has too many NCs, so choose the first and log a warning. This is a warning and not an error because requeuing will not solve the problem,
			   so instead we'll make a best effort of using what is available */
			log.Printf("WARN: Node %s should have 1 NC but has %d: %v", nodeID, len(ncs), ncs)
		}
		ncID = ncs[0].NcID
		n.NodeIDToNCID.Set(nodeID, ncID, cache.NoExpiration)
	}

	return ncID, nil
}

func (n *NodeNetworkConfigReconciler) getNewNNCFromNCResponse(nc *api.NetworkContainerResponse, name, namespace string) *acnv1alpha.NodeNetworkConfig {
	// Start with the standard empty NNC
	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: n.ScalerConfig.BatchSize,
		},
		Status: n.getNNCStatusFromNCResponse(nc),
	}

	return &nnc
}

// getNNCStatusFromNCResponse returns a NNC Status representing the NC response
func (n *NodeNetworkConfigReconciler) getNNCStatusFromNCResponse(nc *api.NetworkContainerResponse) acnv1alpha.NodeNetworkConfigStatus {
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
		},
	}

	// Populate scaler config with scaler config from n
	scaler := acnv1alpha.Scaler{
		BatchSize:               n.ScalerConfig.BatchSize,
		ReleaseThresholdPercent: n.ScalerConfig.ReleaseThresholdPercent,
		RequestThresholdPercent: n.ScalerConfig.RequestThresholdPercent,
	}
	nncStatus.Scaler = scaler

	return nncStatus
}
