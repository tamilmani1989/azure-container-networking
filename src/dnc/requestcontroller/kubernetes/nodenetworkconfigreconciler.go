// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/api"
	"dnc/requestcontroller"
	"fmt"
	"log"
	"net/http"
	"reflect"
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
	Client                  K8sClient
	DncClient               requestcontroller.DNCClient
	DNCClientRequestTimeout time.Duration
	DNCClientAsyncWait      time.Duration
	ScalerConfig            requestcontroller.RequestScalerConfig
	NodeIDToNCID            *cache.Cache // map name of NNC to NC ID
	RegisteredNodes         *cache.Cache // Keep a map of nodes and ips to know what has already been registered on DNC
	VnetGUID                string       //vnet managed by this DNC
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

	err = n.synchronousDeleteNC(n.VnetGUID, nodeID, ncID)
	if err != nil {
		// Could not clean up NC, so requeue
		return err
	}

	// Delete Node from DNC
	err = n.synchronousDeleteNode(nodeID)
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
		ncResponse, _ = n.DncClient.GetNC(n.VnetGUID, ncID)
	}

	nnc := n.getNNCFromNCResponse(ncResponse, name, namespace)

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

	if n.VnetGUID == "" {
		var found bool
		var vnetGUID string
		vnetGUID, found = node.Annotations[VnetGUIDAnnotation]
		if !found {
			return fmt.Errorf("Could not find VNET from annotation %s", VnetGUIDAnnotation)
		}
		n.VnetGUID = vnetGUID
	}

	err := n.synchronousCreateNode(nodeID, internalIPAddress, n.VnetGUID)
	if err != nil {
		// DNC client error, requeue
		return err
	}

	// Create or Update the NC on DNC
	subnetID, found := node.Annotations[SubnetNameAnnotation]
	if !found {
		// Don't know what subnet to use, so requeue
		return fmt.Errorf("Could not find subnet from annotation %s", SubnetNameAnnotation)
	}

	ncResponse, err := n.synchronousCreateOrUpdateNC(n.VnetGUID, nodeID, subnetID, nodeNetworkConfig.Spec.IPsNotInUse, nodeNetworkConfig.Spec.RequestedIPCount)
	if err != nil {
		//DNC client error, requeue
		return err
	}

	// Publish k8s status
	ipAssignments := make([]acnv1alpha.IPAssignment, len(ncResponse.IPConfiguration.SecondaryIPs))
	index := 0
	for id, ipInfo := range ncResponse.IPConfiguration.SecondaryIPs {
		ipAssignments[index] = acnv1alpha.IPAssignment{
			Name: id,
			IP:   ipInfo.IPAddress,
		}
		index++
	}
	status := acnv1alpha.NodeNetworkConfigStatus{
		Scaler: acnv1alpha.Scaler{
			BatchSize:               n.ScalerConfig.BatchSize,
			ReleaseThresholdPercent: n.ScalerConfig.ReleaseThresholdPercent,
			RequestThresholdPercent: n.ScalerConfig.RequestThresholdPercent,
		},
		NetworkContainers: []acnv1alpha.NetworkContainer{
			{
				ID:             ncResponse.NcID,
				PrimaryIP:      ncResponse.IPConfiguration.IPAddress,
				SubnetID:       ncResponse.IPConfiguration.IPSubnet,
				IPAssignments:  ipAssignments,
				DefaultGateway: ncResponse.IPConfiguration.GatewayIP,
				// TODO Netmask?
			},
		},
	}
	nodeNetworkConfig.Status = status
	err = n.Client.Status().Update(context.TODO(), nodeNetworkConfig)
	if err != nil {
		// Unable to write status, requeue
		return err
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
		nodeInfoResponse, apiErr := n.DncClient.GetNode(nodeID)
		if apiErr != nil && apiErr.HTTPStatus != http.StatusOK {
			if apiErr.Code == api.NodeNotFound {
				// Node isn't found, so NC isn't found
				return "", nil
			}
			// return any other api error
			return "", apiErr
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

// TODO: Consider creating a syncronous client in "dnc/client" instead of maintiain the logic everywhere we need it
// Polling logic comes from dnc/runner, when this is moved to client, we can eliminate polling logic here
func (n *NodeNetworkConfigReconciler) synchronousCreateOrUpdateNC(vnetID, nodeID, subnetID string, toBeDeletedIps []string, desiredSecondaryIPCount int64) (*api.NetworkContainerResponse, error) {
	var (
		respErr          *api.JsonErr
		ncStatus         *api.RequestStatus
		expectedNcStatus = &api.RequestStatus{
			State:  "Program",
			Status: "Completed"}
		ncResponse *api.NetworkContainerResponse
	)

	ncID, err := n.getNCIDFromNodeID(nodeID)
	if err != nil {
		return nil, err
	}

	if ncID == "" {
		// If there is no NC registered for the Node, then create a new one
		ncID = uuid.New().String()
	}

	//TODO: validate these parameters
	ncReq := api.NetworkContainerRequestV2{
		AllocationRequest: api.AllocationRequest{
			SubnetName:       subnetID,
			SecondaryIPCount: int(desiredSecondaryIPCount),
		},
		AssociationInfo: api.AssociationInfo{
			NodeID:        nodeID,
			ContainerType: cns.AzureContainerInstance,
		},
		AllowHostToNCCommunication: false,
		AllowNCToHostCommunication: false,
		ToBeDeletedIPIds:           toBeDeletedIps,
	}

	respErr = n.DncClient.UpsertNcV2(vnetID, ncID, ncReq)
	if respErr != nil && respErr.HTTPStatus != http.StatusOK {
		return nil, respErr
	}

	for start := time.Now(); ; {
		ncStatus, respErr = n.DncClient.GetNCStatus(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < n.DNCClientRequestTimeout {
			time.Sleep(n.DNCClientAsyncWait)
			continue
		}

		return nil, fmt.Errorf("Create nc failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
		expectedNcStatus.State = "RefreshAssociation"
		if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
			return nil, fmt.Errorf("Create nc failed;\nreceived %+v\nexpected %+v", ncStatus, expectedNcStatus)
		}
	}

	ncResponse, respErr = n.DncClient.GetNC(vnetID, ncID)
	if respErr != nil && respErr.HTTPStatus != http.StatusOK {
		return nil, respErr
	}

	return ncResponse, nil
}

func (n *NodeNetworkConfigReconciler) synchronousCreateNode(nodeID, ipAddress, vnetID string) error {
	// Check cache to see if Node has already been registered with this IP
	ipCached, found := n.RegisteredNodes.Get(nodeID)
	ip := ""
	if found {
		ip = ipCached.(string)
		if ip == ipAddress {
			// Node already exists on DNC with the correct IP
			return nil
		}
	}

	// Create or Update the node

	respErr := n.DncClient.AddNode(nodeID, ipAddress, cns.Kubernetes, vnetID)
	if respErr != nil && respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("Failed to create node %s with error %+v", nodeID, respErr)
	}

	var nodeCfg *api.NodeInfoResponse

	for start := time.Now(); ; {
		nodeCfg, respErr = n.DncClient.GetNode(nodeID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < n.DNCClientRequestTimeout {
			time.Sleep(n.DNCClientAsyncWait)
			continue
		}

		return fmt.Errorf("Register node (get) failed with error %+v", respErr)
	}

	receivedNodeCfg := nodeCfg.NodeInfo
	expectedNodeCfg := api.NodeInfoRequest{
		IPAddresses:      []string{ipAddress},
		LocationType:     "",
		OrchestratorType: cns.Kubernetes}
	if receivedNodeCfg.String() != expectedNodeCfg.String() {
		return fmt.Errorf("Register node failed;\nreceived %+v\nexpected %+v", receivedNodeCfg, expectedNodeCfg)
	}

	//Add node and ip to cache
	n.RegisteredNodes.Set(nodeID, ipAddress, cache.NoExpiration)

	return nil
}

// synchronousDeleteNC is an idempotent function that calls deleteNC then waits until the NC has been deleted before returning.
func (n *NodeNetworkConfigReconciler) synchronousDeleteNC(vnetID, nodeID, ncID string) error {
	if ncID == "" {
		// No NC ID means nothing to delete
		return nil
	}

	// Remove it from cache if present
	n.NodeIDToNCID.Delete(nodeID)

	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = n.DncClient.DeleteNC(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusNotFound || respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < n.DNCClientRequestTimeout {
			time.Sleep(n.DNCClientAsyncWait)
			continue
		}

		return fmt.Errorf("Delete nc failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		_, respErr = n.DncClient.GetNCStatus(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusNotFound {
			break
		}

		if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < n.DNCClientRequestTimeout {
			time.Sleep(n.DNCClientAsyncWait)
			continue
		}

		return fmt.Errorf("Delete nc failed with error %+v", respErr)
	}

	return nil
}

// synchronousDeleteNode is an idempotent function that calls deleteNode on DNC and waits until the Node has been deleted before returning
func (n *NodeNetworkConfigReconciler) synchronousDeleteNode(nodeID string) error {
	if nodeID == "" {
		// No node to delete
		return nil
	}

	// Remove it from the cache if present
	n.NodeIDToNCID.Delete(nodeID)
	n.RegisteredNodes.Delete(nodeID)

	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = n.DncClient.DeleteNode(nodeID)
		if respErr.HTTPStatus == http.StatusNotFound || respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < n.DNCClientRequestTimeout {
			time.Sleep(n.DNCClientAsyncWait)
			continue
		}

		return fmt.Errorf("Unregister node failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		_, respErr = n.DncClient.GetNode(nodeID)
		if respErr.HTTPStatus == http.StatusNotFound {
			break
		}

		if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < n.DNCClientRequestTimeout {
			time.Sleep(n.DNCClientAsyncWait)
			continue
		}

		return fmt.Errorf("Unregister node failed with error %+v", respErr)
	}

	return nil
}

func (n *NodeNetworkConfigReconciler) getNNCFromNCResponse(nc *api.NetworkContainerResponse, name, namespace string) *acnv1alpha.NodeNetworkConfig {
	// Start with the standard empty NNC
	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: n.ScalerConfig.BatchSize,
		},
		Status: acnv1alpha.NodeNetworkConfigStatus{
			Scaler: acnv1alpha.Scaler{
				BatchSize:               n.ScalerConfig.BatchSize,
				ReleaseThresholdPercent: n.ScalerConfig.ReleaseThresholdPercent,
				RequestThresholdPercent: n.ScalerConfig.RequestThresholdPercent,
			},
		},
	}

	if nc != nil {
		ipAssignments := make([]acnv1alpha.IPAssignment, len(nc.IPConfiguration.SecondaryIPs))
		for id, ipInfo := range nc.IPConfiguration.SecondaryIPs {
			ipAssignments = append(ipAssignments, acnv1alpha.IPAssignment{
				Name: id,
				IP:   ipInfo.IPAddress,
			})
		}

		nnc.Spec.RequestedIPCount = int64(len(nc.IPConfiguration.SecondaryIPs))
		nnc.Status.NetworkContainers = []acnv1alpha.NetworkContainer{
			{
				ID:            nc.NcID,
				PrimaryIP:     nc.IPConfiguration.IPAddress,
				SubnetID:      nc.IPConfiguration.SubnetName,
				IPAssignments: ipAssignments,
			},
		}
	}

	return &nnc
}
