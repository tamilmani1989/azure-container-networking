// Copyright 2019 Microsoft. All rights reserved.

package server

import (
	"dnc/api"
	"dnc/controller"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/Azure/azure-container-networking/cns"

	"github.com/gorilla/mux"
)

const (
	// url parameters
	networkIDStr          = "networkID"
	networkContainerIDStr = "networkContainerID"
	nodeIDStr             = "nodeID"
	reservationIDStr      = "reservationID"
	reservationSetIDStr   = "reservationSetID"
	subnetStr             = "subnet"
	infraNetworkIDStr     = "infraNetworkID"
	nodeSetName           = "nodeSetName"

	// query parameters
	apiVersionQueryParam  = "api-version"
	forceDeleteQueryParam = "force-delete"

	// content types
	contentTypeStr     = "Content-Type"
	defaultContentType = "application/json; charset=UTF-8"
)

// todo: remove global. create proper routes struct with controller as dependency
var ctrler *controller.Controller

// SetController - Cache handle to the controller object
func SetController(c *controller.Controller) {
	ctrler = c
}

// Index - Index handler
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, utils.Version)
}

func decodeRequestBody(request *http.Request, decodedBody interface{}) error {
	var err error
	if request.Body != nil {
		err = json.NewDecoder(request.Body).Decode(&decodedBody)
	} else {
		err = fmt.Errorf("Request body is empty")
	}

	return err
}

// CreateOrUpdateNode - Handle to create/update node.
func CreateOrUpdateNode(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	vars := mux.Vars(r)
	nodeID, ok := vars[nodeIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			Text:       api.NodeIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	config := ctrler.GetConfig()
	var nodeInfo api.NodeInfoRequest

	if err := decodeRequestBody(r, &nodeInfo); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	if validationErr := nodeInfo.Validate(config, nodeID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	putNodeInfoErr := ctrler.PersistNodeInfo(nodeID, &nodeInfo, apiVersion)
	if putNodeInfoErr != nil {
		setRequestFailed(w, putNodeInfoErr)
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// GetNode - Handler for querying registered node information
func GetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID, ok := vars[nodeIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			Text:       api.NodeIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	nodeInfo, getErr := ctrler.GetNodeInfo(nodeID)
	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&nodeInfo); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

// GetNodes - Handler for getting all registered nodes
func GetNodes(w http.ResponseWriter, r *http.Request) {
	nodeSet := r.URL.Query().Get("node-set")
	nodes, getErr := ctrler.GetNodes(nodeSet)
	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&nodes); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

// DeleteNode - Handle to delete node configuration
func DeleteNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID, ok := vars[nodeIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			Text:       api.NodeIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	deleteNodeErr := ctrler.DeleteNode(nodeID)
	w.Header().Set(contentTypeStr, defaultContentType)

	if deleteNodeErr == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	setRequestFailed(w, deleteNodeErr)
}

// CreateOrUpdateReservationSet - Handler for creating or updating reservation set
func CreateOrUpdateReservationSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	reservationSetID, ok := vars[reservationSetIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidReservationSetID, api.ReservationSetIDMissing))
		return
	}

	if validationErr := ValidateReservationSetRequest(networkID, reservationSetID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	var request api.ReservationSetRequest
	w.Header().Set(contentTypeStr, defaultContentType)
	if err := decodeRequestBody(r, &request); err != nil {
		jsonErr := api.UnprocessableEntityf(api.JsonBodyUnMarshalFailed, "could not unmarshal request body. error: %v", err)
		setRequestFailed(w, jsonErr)
		return
	}

	if validationErr := request.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	err := ctrler.CreateOrUpdateReservationSetRequest(networkID, reservationSetID, request, apiVersion)
	if err == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	setRequestFailed(w, err)
}

// ReserveIP - Handler for IP reservation
func ReserveIP(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	reservationID, ok := vars[reservationIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			Text:       api.ReservationIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateReservationRequest(networkID, reservationID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	var allocationRequest api.AllocationRequest
	w.Header().Set(contentTypeStr, defaultContentType)
	if err := decodeRequestBody(r, &allocationRequest); err != nil {
		jsonErr := api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       fmt.Sprintf("Failed to unmarshal request body. Error:%s", err.Error()),
			HTTPStatus: http.StatusUnprocessableEntity}

		setRequestFailed(w, &jsonErr)
		return
	}

	if validationErr := allocationRequest.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	reservationErr := ctrler.PersistIPReservationRequest(
		networkID,
		reservationID,
		allocationRequest,
		apiVersion)

	if reservationErr == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	setRequestFailed(w, reservationErr)
}

// GetReservation - Handler for querying reservation info.
func GetReservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	reservationID, ok := vars[reservationIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			Text:       api.ReservationIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateReservationRequest(networkID, reservationID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	reservationCfg, getErr := ctrler.GetReservationConfiguration(
		networkID,
		reservationID)
	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&reservationCfg); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

func GetReservationSetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	reservationSetID, ok := vars[reservationSetIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidReservationSetID, api.ReservationSetIDMissing))
		return
	}

	if validationErr := ValidateReservationSetRequest(networkID, reservationSetID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	status, httpCode, errGet := ctrler.GetReservationSetStatus(networkID, reservationSetID)
	if errGet == nil {
		w.WriteHeader(httpCode)
		if status != nil {
			if err := json.NewEncoder(w).Encode(&status); err != nil {
				// todo: this is copied from other GetStatus code. Revisit this.
				panic(err)
			}
		}
		return
	}

	setRequestFailed(w, errGet)
}

func DeleteReservationSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	reservationSetID, ok := vars[reservationSetIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidReservationSetID, api.ReservationSetIDMissing))
		return
	}

	if validationErr := ValidateReservationSetRequest(networkID, reservationSetID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)
	err := ctrler.DeleteReservationSet(networkID, reservationSetID)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	setRequestFailed(w, err)
}

func GetReservationSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	reservationSetID, ok := vars[reservationSetIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequestf(api.InvalidReservationSetID, api.ReservationSetIDMissing))
		return
	}

	if validationErr := ValidateReservationSetRequest(networkID, reservationSetID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	reservationSet, errGet := ctrler.GetReservationSet(networkID, reservationSetID)
	if errGet == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&reservationSet); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, errGet)
}

// GetReservationStatus - Handler for querying reservation status.
func GetReservationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	reservationID, ok := vars[reservationIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			Text:       api.ReservationIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateReservationRequest(networkID, reservationID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	reservationStatus, retcode, getErr := ctrler.GetReservationStatus(
		networkID,
		reservationID)
	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(retcode)
		if reservationStatus != nil {
			if err := json.NewEncoder(w).Encode(&reservationStatus); err != nil {
				panic(err)
			}
		}
		return
	}

	setRequestFailed(w, getErr)
}

func GetNodeSetStatus(w http.ResponseWriter, r *http.Request) {
	nodeSetName, ok := mux.Vars(r)[nodeSetName]
	if !ok {
		out := api.BadRequest(api.StatusBadRequest, "node set name not found in url")
		setRequestFailed(w, out)
		return
	}

	nse, err := ctrler.GetNodeSet(nodeSetName)
	if err != nil {
		setRequestFailed(w, err)
		return
	}

	if nse.RowKey == "" {
		out := api.NotFoundf(api.EntityNotFound, "node set %s not found", nodeSetName)
		setRequestFailed(w, out)
		return
	}

	status := api.RequestStatus{
		State:         nse.State.String(),
		Status:        nse.Status.String(),
		FailureDetail: nse.FailureDetail,
	}

	w.Header().Set(contentTypeStr, defaultContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		loggershim.LS.Errorf("could not encode node set status: %v", err)
	}
}

func CreateNodeSet(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	nodeSetName, ok := mux.Vars(r)[nodeSetName]
	if !ok {
		setRequestFailed(w, api.BadRequest(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		setRequestFailed(w, api.InternalServerErrorf(api.StatusInternalServerError, "could not consume request body: %v", err))
		return
	}

	// todo: currently there are no required fields in the body of a node set creation request.
	//  until we add any required fields, we want to allow empty body or empty json object.
	if len(bs) != 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(bs, &m); err != nil {
			setRequestFailed(w, api.BadRequest(api.JsonBodyUnMarshalFailed, fmt.Sprintf("could not parse node set request body: %v", err)))
			return
		}
	}

	nse, apiErr := ctrler.CreateNodeSet(nodeSetName, apiVersion)
	if apiErr != nil {
		setRequestFailed(w, apiErr)
		return
	}

	nseResp := api.NodeSetResponse{
		NodeSetName: nse.NodeSetName,
		RequestStatus: api.RequestStatus{
			State:         nse.State.String(),
			Status:        nse.Status.String(),
			FailureDetail: nse.FailureDetail,
		},
	}

	w.Header().Set(contentTypeStr, defaultContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(nseResp); err != nil {
		loggershim.LS.Errorf("could not encode node set response: %v", err)
	}
}

func DeleteNodeSet(w http.ResponseWriter, r *http.Request) {
	nodeSetName, ok := mux.Vars(r)[nodeSetName]
	if !ok {
		out := api.BadRequest(api.StatusBadRequest, "node set name not found in url")
		setRequestFailed(w, out)
		return
	}

	forceDelete, err := extractBoolQueryParam(r.URL, forceDeleteQueryParam)
	if err != nil {
		out := api.BadRequestf(api.StatusBadRequest, "invalid boolean value for force-delete: %v", err)
		setRequestFailed(w, out)
		return
	}

	if forceDelete {
		if err := ctrler.ForceDeleteNodeSet(nodeSetName); err != nil {
			setRequestFailed(w, err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		return
	}

	if err := ctrler.DeleteNodeSet(nodeSetName); err != nil {
		setRequestFailed(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteReservation - Handler for deleting reservation request.
func DeleteReservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(
			w,
			&api.JsonErr{
				Code:       api.InvalidNetworkID,
				Text:       api.NetworkIDMissing,
				HTTPStatus: http.StatusBadRequest})
		return
	}

	reservationID, ok := vars[reservationIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			Text:       api.ReservationIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateReservationRequest(networkID, reservationID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	deleteErr := ctrler.DeleteReservation(
		networkID,
		reservationID)

	if deleteErr == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set(contentTypeStr, defaultContentType)
	setRequestFailed(w, deleteErr)
}

// GetNetworkContainerStatus - Handler for querying network container status
func GetNetworkContainerStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok := vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	ncStatus, retcode, getErr := ctrler.GetNetworkContainerStatus(
		networkID,
		ncID)

	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(retcode)
		if ncStatus != nil {
			if err := json.NewEncoder(w).Encode(&ncStatus); err != nil {
				panic(err)
			}
		}
		return
	}

	setRequestFailed(w, getErr)
}

// DeleteNetworkContainer - Handler for delete network container
func DeleteNetworkContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}
	ncID, ok := vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	if err := ctrler.DeleteNetworkContainer(networkID, ncID); err == nil {
		w.Header().Set(contentTypeStr, defaultContentType)
		w.WriteHeader(http.StatusOK)
		return
	} else {
		setRequestFailed(w, err)
	}
}

// CreateNetworkContainer - Handler for create network container
func CreateNetworkContainer(w http.ResponseWriter, r *http.Request) {
	var ncRequestV2 *api.NetworkContainerRequestV2
	var ncRequestV1 *api.NetworkContainerRequestV1
	var isV1Request = false
	var err error
	var body []byte

	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok := vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	body, err = ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		if err := r.Body.Close(); err != nil {
			panic(err)
		}

		return
	}

	ncRequestV1, err = getNetworkContainerRequestV1(body)
	if err == nil {
		if ncRequestV1 == nil {
			ncRequestV2, err = getNetworkContainerRequestV2(body)
			if ncRequestV2 == nil {
				setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkContainerRequest, Text: "Network Container request is invalid", HTTPStatus: http.StatusBadRequest})
				return
			}
		} else {
			isV1Request = true
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity) // unprocessable entity
		if err = json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	var ncPersistErr *api.JsonErr
	if isV1Request {
		if validationErr := ncRequestV1.Validate(); validationErr != nil {
			setRequestFailed(w, validationErr)
			return
		}

		ncPersistErr = ctrler.PersistNetworkContainerRequestV1(
			networkID,
			ncID,
			ncRequestV1,
			apiVersion)
	} else {
		if validationErr := ncRequestV2.Validate(); validationErr != nil {
			setRequestFailed(w, validationErr)
			return
		}

		ncPersistErr = ctrler.PersistNetworkContainerRequestV2(
			networkID,
			ncID,
			ncRequestV2,
			apiVersion)
	}

	if ncPersistErr != nil {
		setRequestFailed(w, ncPersistErr)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// CreateOrUpdateNetworkContainerV2 - Handler for upsert of network container requests v2
func CreateOrUpdateNetworkContainerV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeStr, defaultContentType)

	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	pathVars := mux.Vars(r)

	networkID, ok := pathVars[networkIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequest(api.InvalidNetworkID, api.NetworkIDMissing))
		return
	}

	ncID, ok := pathVars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, api.BadRequest(api.InvalidNetworkContainerID, api.NetworkContainerIDMissing))
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	var ncRequest api.NetworkContainerRequestV2
	if err := json.NewDecoder(r.Body).Decode(&ncRequest); err != nil {
		setRequestFailed(w, api.BadRequest(api.InvalidNetworkContainerRequest, fmt.Sprintf("Could not parse network container request body: %v", err)))
		return
	}

	if validationErr := ncRequest.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	ncPersistErr := ctrler.PersistNetworkContainerRequestV2(networkID, ncID, &ncRequest, apiVersion)
	if ncPersistErr != nil {
		setRequestFailed(w, ncPersistErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getNetworkContainerRequestV1(body []byte) (*api.NetworkContainerRequestV1, error) {
	var ncRequestV1 api.NetworkContainerRequestV1
	if err := json.Unmarshal(body, &ncRequestV1); err != nil {
		return nil, err
	}

	//todo: revisit and make the logic to check if this is v1 / v2 more robust.
	if !reflect.DeepEqual(ncRequestV1, (api.NetworkContainerRequestV1{})) {
		if len(ncRequestV1.ReservationID) != 0 || len(ncRequestV1.ReservationSetID) != 0 {
			switch ncRequestV1.AssociationInfo.ContainerType {
			case cns.AzureContainerInstance:
				fallthrough
			case cns.WebApps:
				fallthrough
			case cns.Docker:
				fallthrough
			case cns.JobObject:
				fallthrough
			case api.BareMetalContainerType:
				fallthrough
			case cns.COW:
				return &ncRequestV1, nil
			}
		}
	}

	return nil, nil
}

func getNetworkContainerRequestV2(body []byte) (*api.NetworkContainerRequestV2, error) {
	var ncRequestV2 api.NetworkContainerRequestV2
	if err := json.Unmarshal(body, &ncRequestV2); err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(ncRequestV2, (api.NetworkContainerRequestV2{})) {
		switch ncRequestV2.AssociationInfo.ContainerType {
		case cns.AzureContainerInstance:
			fallthrough
		case cns.WebApps:
			fallthrough
		case cns.Docker:
			fallthrough
		case cns.JobObject:
			fallthrough
		case api.BareMetalContainerType:
			fallthrough
		case cns.COW:
			return &ncRequestV2, nil
		}
	}

	return nil, nil
}

// GetNetworkContainer - Query network container configuration
func GetNetworkContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var networkID string
	var ncID string
	var ok bool
	networkID, ok = vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok = vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	ncConfig, getNCConfigErr := ctrler.GetNetworkContainer(networkID, ncID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if getNCConfigErr != nil {
		w.WriteHeader(getNCConfigErr.HTTPStatus)
		if err := json.NewEncoder(w).Encode(getNCConfigErr); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ncConfig); err != nil {
			panic(err)
		}
	}
}

// GetNetworkContainerConfiguration - Query network container configuration
func GetNetworkContainerConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var networkID string
	var ncID string
	var ok bool
	networkID, ok = vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok = vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	ncConfig, getNCConfigErr := ctrler.GetNetworkContainerConfiguration(networkID, ncID)
	w.Header().Set(contentTypeStr, defaultContentType)
	if getNCConfigErr != nil {
		w.WriteHeader(getNCConfigErr.HTTPStatus)
		if err := json.NewEncoder(w).Encode(getNCConfigErr); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ncConfig); err != nil {
			panic(err)
		}
	}
}

// GetNetworkContainerCleanupStatus returns nc cleanup status
func GetNetworkContainerCleanupStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok := vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	cleanupStatus, respErr := ctrler.GetNetworkContainerCleanupStatus(networkID, ncID)

	if respErr != nil {
		w.WriteHeader(respErr.HTTPStatus)
		if err := json.NewEncoder(w).Encode(respErr); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(cleanupStatus); err != nil {
			panic(err)
		}
	}
}

// RegisterNode - mark a node as registered
func RegisterNode(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	vars := mux.Vars(r)

	nodeID, ok := vars[nodeIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			Text:       api.NodeIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	infraNetworkID, ok := vars[infraNetworkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	nodeConfig, err := ctrler.GetNodeConfiguration(nodeID)
	if err != nil {
		w.WriteHeader(err.HTTPStatus)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	if nodeConfig == nil {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       fmt.Sprintf("Node configuration for node %v not found.", nodeID),
			HTTPStatus: http.StatusNotFound})
		return
	}

	if nodeConfig.NodeInfo.InfrastructureNetwork != infraNetworkID {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       "Invalid node ID and infrastructure network combination",
			HTTPStatus: http.StatusUnauthorized})
		return
	}

	var nodeRegistrationRequest api.NodeRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&nodeRegistrationRequest); err != nil {
		setRequestFailed(w, api.BadRequest(api.InvalidNodeRegistrationRequest,
			fmt.Sprintf("Could not parse node registration request body: %v", err)))
		return
	}

	if validationErr := nodeRegistrationRequest.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	if err = ctrler.RegisterNode(nodeID, apiVersion, &nodeRegistrationRequest); err != nil {
		w.WriteHeader(err.HTTPStatus)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	info := &cns.SetOrchestratorTypeRequest{
		OrchestratorType: nodeConfig.NodeInfo.OrchestratorType,
		DncPartitionKey:  ctrler.GetConfig().DataStoreSettings.AzureStorageInfo.PartitionKey,
		NodeID:           nodeID}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(info); err != nil {
		panic(err)
	}
}

// SyncNodeNetworkContainers - compile and retrieve network container requests
func SyncNodeNetworkContainers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	nodeID, ok := vars[nodeIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			Text:       api.NodeIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	infraNetworkID, ok := vars[infraNetworkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	nodeConfig, err := ctrler.GetNodeConfiguration(nodeID)
	if err != nil {
		w.WriteHeader(err.HTTPStatus)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	if !nodeConfig.Registered {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.UnregisteredNode,
			Text:       "Node must be registered before sync",
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if nodeConfig.NodeInfo.InfrastructureNetwork != infraNetworkID {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       "Invalid node ID and infrastructure network combination",
			HTTPStatus: http.StatusUnauthorized})
		return
	}

	var syncNodeInfo cns.NodeInfoResponse

	for networkID, ncTuple := range nodeConfig.DeployedNetworkContainerMap {
		for ncID := range ncTuple {
			ncRequest, err := ctrler.GetNetworkContainerRequest(networkID, ncID)
			if err != nil {
				w.WriteHeader(err.HTTPStatus)
				if err := json.NewEncoder(w).Encode(err); err != nil {
					panic(err)
				}
				return
			}

			syncNodeInfo.NetworkContainers = append(syncNodeInfo.NetworkContainers, *ncRequest)
		}
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(syncNodeInfo); err != nil {
		panic(err)
	}
}

// CreateNetwork - Handler for create network
func CreateNetwork(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	var networkInfo api.NetworkInfoRequest
	var err error
	if err = decodeRequestBody(r, &networkInfo); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err = json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	if validationErr := networkInfo.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	t := ctrler.PersistNetworkJoinRequest(networkID, networkInfo, apiVersion)
	if t != nil {
		setRequestFailed(w, t)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(t); err != nil {
		panic(err)
	}
}

// GetNetwork - Handler for retrieving the network configuration (e.g. delegated subnets)
func GetNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	networkConfiguration, getErr := ctrler.GetNetwork(networkID)
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(networkConfiguration); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

// GetNetworks - Handler for retrieving the vnets that are managed by this DNC
func GetNetworks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeStr, defaultContentType)

	networks, getErr := ctrler.GetNetworks()
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(networks); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

func CleanupAndDeleteSubnet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	subnet, ok := vars[subnetStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			Text:       api.SubnetNameMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	w.Header().Set(contentTypeStr, defaultContentType)

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	if err := ctrler.CleanupSubnet(networkID, strings.ToLower(subnet)); err != nil {
		setRequestFailed(w, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// PutSubnet - Handler for put delegated subnet
func PutSubnet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	subnet, ok := vars[subnetStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			Text:       api.SubnetNameMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	isDelete := r.Method == http.MethodDelete
	w.Header().Set(contentTypeStr, defaultContentType)

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	var subnetRequest api.SubnetRequest
	// Decode the request body if the request is not a delete request
	if !isDelete {
		if err := decodeRequestBody(r, &subnetRequest); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
			return
		}

		if validationErr := subnetRequest.Validate(); validationErr != nil {
			setRequestFailed(w, validationErr)
			return
		}
	}

	apiVersion := r.URL.Query().Get(apiVersionQueryParam)
	forceDelete := r.URL.Query().Get(forceDeleteQueryParam)

	switch apiVersion {
	case "2018-03-01":
		// Set the Auth Token to empty if the request is of type Delete
		if isDelete {
			subnetRequest.AuthenticationToken = ""
		}

		if err := ctrler.PersistSubnetDelegationRequest(networkID, strings.ToLower(subnet), subnetRequest, isDelete, apiVersion); err != nil {
			setRequestFailed(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)

	case "2021-02-24":
		if isDelete {
			if forceDelete == "true" {
				if err := ctrler.CleanupSubnet(networkID, strings.ToLower(subnet)); err != nil {
					setRequestFailed(w, err)
					return
				}
			} else {
				if err := ctrler.DeleteSubnetV2(networkID, strings.ToLower(subnet), apiVersion); err != nil {
					setRequestFailed(w, err)
					return
				}
			}
		} else {
			if err := ctrler.PersistSubnetDelegationRequestV2(networkID, strings.ToLower(subnet), subnetRequest, apiVersion); err != nil {
				setRequestFailed(w, err)
				return
			}
		}

		w.WriteHeader(http.StatusAccepted)

	case "2021-06-11":
		if isDelete {
			if err := ctrler.DeleteSubnetV3(networkID, strings.ToLower(subnet), apiVersion); err != nil {
				setRequestFailed(w, err)
				return
			}
		} else {
			if err := ctrler.PersistSubnetDelegationRequestV3(networkID, strings.ToLower(subnet), subnetRequest, apiVersion); err != nil {
				setRequestFailed(w, err)
				return
			}
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

// GetSubnetStatus - Handler for retrieving the subnet status
func GetSubnetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	subnet, ok := vars[subnetStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			Text:       api.SubnetNameMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	subnetStatus, retCode, getErr := ctrler.GetSubnetStatus(networkID, strings.ToLower(subnet))

	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(retCode)
		if subnetStatus != nil {
			if err := json.NewEncoder(w).Encode(&subnetStatus); err != nil {
				panic(err)
			}
		}
		return
	}

	setRequestFailed(w, getErr)
}

// GetSubnetConfiguration - Handler for retrieving the subnet configuration
func GetSubnetConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	subnet, ok := vars[subnetStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			Text:       api.SubnetNameMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	subnetConfiguration, getErr := ctrler.GetSubnetConfiguration(networkID, strings.ToLower(subnet))

	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&subnetConfiguration); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

// DeleteNetwork - Handler for deleting the network.
func DeleteNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var networkID string
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	t := ctrler.DeleteNetwork(networkID)
	w.Header().Set(contentTypeStr, defaultContentType)
	if t != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	if err := json.NewEncoder(w).Encode(t); err != nil {
		panic(err)
	}
}

// GetNetworkStatus - Handler for retrieving the network status.
func GetNetworkStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	networkStatus, retcode, getErr := ctrler.GetNetworkStatus(networkID)
	w.Header().Set(contentTypeStr, defaultContentType)
	if getErr == nil {
		w.WriteHeader(retcode)
		if networkStatus != nil {
			if err := json.NewEncoder(w).Encode(&networkStatus); err != nil {
				panic(err)
			}
		}

		return
	}

	setRequestFailed(w, getErr)
}

// InitializeAzureStorage - Handler for initializing datastore settings to establish a connection.
func InitializeAzureStorage(w http.ResponseWriter, r *http.Request) {
	var request api.AzureStorageRequest
	w.Header().Set(contentTypeStr, defaultContentType)

	if err := decodeRequestBody(r, &request); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	if validationErr := request.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	t := ctrler.InitializeAzureStorage(request)
	if t != nil {
		w.WriteHeader(t.HTTPStatus)
		if err := json.NewEncoder(w).Encode(&t); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// InitializeAzureStorageV2 - Handler for initializing datastore settings to establish a connection.
func InitializeAzureStorageV2(w http.ResponseWriter, r *http.Request) {
	var request api.AzureStorageRequestV2
	w.Header().Set(contentTypeStr, defaultContentType)

	if err := decodeRequestBody(r, &request); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	if validationErr := request.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	t := ctrler.InitializeAzureStorageV2(request)
	if t != nil {
		w.WriteHeader(t.HTTPStatus)
		if err := json.NewEncoder(w).Encode(&t); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// InitializeAzureStorageV3 - Handler for initializing datastore settings to establish a connection.
func InitializeAzureStorageV3(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blobName, ok := vars["blob"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidBlobName,
			HTTPStatus: http.StatusBadRequest,
			Text:       "BlobName is missing"})
		return
	}

	containerName, ok := vars["container"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidContainerName,
			HTTPStatus: http.StatusBadRequest,
			Text:       "ContainerName is missing"})
		return
	}

	var request api.AzureStorageRequest
	w.Header().Set(contentTypeStr, defaultContentType)
	if err := decodeRequestBody(r, &request); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	if validationErr := request.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	t := ctrler.InitializeAzureStorageV3(request, blobName, containerName)
	if t != nil {
		w.WriteHeader(t.HTTPStatus)
		if err := json.NewEncoder(w).Encode(&t); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// RelinquishPrimaryStatus - Relinquish replica primary status
func RelinquishPrimaryStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeStr, defaultContentType)
	ctrler.RelinquishPrimaryStatus()
	w.WriteHeader(http.StatusOK)
}

// RefreshMonitorStatus - Retrieve refresh monitor status
func RefreshMonitorStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeStr, defaultContentType)
	refreshMonitorStatus := ctrler.GetRefreshMonitorStatus()
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&refreshMonitorStatus); err != nil {
		panic(err)
	}
}

// PromoteContainerState - Promotes container to the next state (if it is stuck)
func PromoteContainerState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var networkID string
	var ncID string
	var ok bool
	networkID, ok = vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok = vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	if err := ctrler.PromoteContainerState(networkID, ncID); err != nil {
		setRequestFailed(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// SkipNodeCleanupForNetworkContainer - Promotes container to the next state (if connection to node has failed during disassociation)
func SkipNodeCleanupForNetworkContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var networkID string
	var ncID string
	var ok bool

	w.Header().Set(contentTypeStr, defaultContentType)
	networkID, ok = vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       api.NetworkIDMissing})
		return
	}

	ncID, ok = vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	if err := ctrler.SkipNodeCleanupForNetworkContainer(networkID, ncID); err != nil {
		setRequestFailed(w, err)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// IsWaitingForNodeCleanup - Check to see if NC is waiting for node cleanup and in DisAssociationFailed state
func IsWaitingForNodeCleanup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var networkID string
	var ncID string
	var ok bool
	networkID, ok = vars[networkIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			Text:       api.NetworkIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok = vars[networkContainerIDStr]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       api.NetworkContainerIDMissing,
			HTTPStatus: http.StatusBadRequest})
		return
	}

	if validationErr := ValidateNetworkContainerRequest(networkID, ncID); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	ncConfig, err := ctrler.GetNetworkContainer(networkID, ncID)
	if err != nil {
		setRequestFailed(w, err)
	} else {
		failureDetail := ncConfig.RequestStatus.FailureDetail
		response := api.IsWaitingForNodeCleanupResponse{IsWaitingForNodeCleanup: false}
		if failureDetail.Code == api.DisAssociationFailed &&
			strings.Contains(failureDetail.Text, "dial tcp") {
			response.IsWaitingForNodeCleanup = true
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&response); err != nil {
			panic(err)
		}
	}
}

// RefreshNMAgentGoalState - Refresh NMAgent goal state
func RefreshNMAgentGoalState(w http.ResponseWriter, r *http.Request) {
	ctrler.RefreshNMAgentGoalState()
	w.WriteHeader(http.StatusOK)
}

// DumpStorageState - dumps DNC storage state
func DumpStorageState(w http.ResponseWriter, r *http.Request) {
	if err := ctrler.DumpStorageState(); err != nil {
		setRequestFailed(w, &api.JsonErr{
			Text:       err.Error(),
			HTTPStatus: http.StatusInternalServerError})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
