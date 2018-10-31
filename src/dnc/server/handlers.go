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

var ctrler *controller.Controller

// SetController - Cache handle to the controller object
func SetController(c *controller.Controller) {
	ctrler = c
}

// Index - Index handler
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to DNC service!!!!!\n")
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
	apiVersion := r.URL.Query().Get("api-version")
	vars := mux.Vars(r)
	nodeID, ok := vars["nodeID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NodeID is missing in the request"})
		return
	}

	var nodeInfo api.NodeInfoRequest

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := decodeRequestBody(r, &nodeInfo); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	if validationErr := nodeInfo.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
	}

	putNodeInfoErr := ctrler.PersistNodeInfo(nodeID, &nodeInfo, apiVersion)
	if putNodeInfoErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(putNodeInfoErr); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// GetNode - Handler for get node information
func GetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID, ok := vars["nodeID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NodeID is missing"})
		return
	}

	nodeInfo, getErr := ctrler.GetNodeInfo(nodeID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&nodeInfo); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

// DeleteNode - Handle to delete node configuration
func DeleteNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID, ok := vars["nodeID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNodeID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NodeID is missing"})
		return
	}

	deleteNodeErr := ctrler.DeleteNode(nodeID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if deleteNodeErr == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	setRequestFailed(w, deleteNodeErr)
}

// ReserveIP - Handler for IP reservation
func ReserveIP(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get("api-version")
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NodeID is missing"})
		return
	}

	reservationID, ok := vars["reservationID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "Reservation ID is missing"})
		return
	}

	var allocationRequest api.AllocationRequest
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := decodeRequestBody(r, &allocationRequest); err != nil {
		jsonErr := api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       fmt.Sprintf("Failed to unmarshal request body. Error:%s", err.Error()),
			HTTPStatus: http.StatusUnprocessableEntity}

		setRequestFailed(w, &jsonErr)
	}

	if validationErr := allocationRequest.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
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

// GetReservation - Handler for retrieving reservation info.
func GetReservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NetworkID is missing"})
		return
	}

	reservationID, ok := vars["reservationID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "ReservationID is missing"})
		return
	}

	reservationCfg, getErr := ctrler.GetReservationConfiguration(
		networkID,
		reservationID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&reservationCfg); err != nil {
			panic(err)
		}
		return
	}

	setRequestFailed(w, getErr)
}

func GetReservationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NetworkID is missing"})
		return
	}

	reservationID, ok := vars["reservationID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "ReservationID is missing"})
		return
	}

	reservationStatus, getErr := ctrler.GetReservationStatus(
		networkID,
		reservationID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if getErr == nil {
		w.WriteHeader(http.StatusOK)
		if reservationStatus != nil {
			if err := json.NewEncoder(w).Encode(&reservationStatus); err != nil {
				panic(err)
			}
		}
		return
	}

	setRequestFailed(w, getErr)
}

func DeleteReservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NetworkID is missing"})
		return
	}

	reservationID, ok := vars["reservationID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidReservationID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "ReservationID is missing"})
		return
	}

	deleteErr := ctrler.DeleteReservation(
		networkID,
		reservationID)

	if deleteErr == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	setRequestFailed(w, deleteErr)
}

// func ReservationsBynetworkID(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	var networkID string
// 	networkID = vars["networkID"]
// 	reservationIDs := ctrler.GetReservationsBynetworkID(networkID)
// 	if reservationIDs != nil {
// 		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
// 		w.WriteHeader(http.StatusOK)
// 		if err := json.NewEncoder(w).Encode(&reservationIDs); err != nil {
// 			panic(err)
// 		}
// 		return
// 	}

// 	// If we didn't find it, 404
// 	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
// 	w.WriteHeader(http.StatusNotFound)
// 	if err := json.NewEncoder(w).Encode(JsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
// 		panic(err)
// 	}

// }

func GetNetworkContainerStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkID, Text: "Network ID is missing"})
		return
	}
	ncID, ok := vars["networkContainerID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{Code: api.NetworkContainerNotFound, Text: "Network container ID is missing"})
		return
	}

	ncStatus, retCode, getErr := ctrler.GetNetworkContainerStatus(
		networkID,
		ncID)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if getErr == nil {
		w.WriteHeader(retCode)
		if ncStatus != nil {
			if err := json.NewEncoder(w).Encode(&ncStatus); err != nil {
				panic(err)
			}
		}
		return
	}

	setRequestFailed(w, getErr)
}

func DeleteNetworkContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkID, Text: "Network ID is missing"})
		return
	}
	ncID, ok := vars["networkContainerID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{Code: api.NetworkContainerNotFound, Text: "Network container ID is missing"})
		return
	}
	err := ctrler.DeleteNetworkContainer(networkID, ncID)

	if err == nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		return
	}

	setRequestFailed(w, err)
}

// CreateNetworkContainer - Handler for create network container
func CreateNetworkContainer(w http.ResponseWriter, r *http.Request) {
	var ncRequestV2 *api.NetworkContainerRequestV2
	var ncRequestV1 *api.NetworkContainerRequestV1
	var isV1Request = false
	var err error
	var body []byte

	apiVersion := r.URL.Query().Get("api-version")
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkID, Text: "Network ID is missing", HTTPStatus: http.StatusBadRequest})
		return
	}

	ncID, ok := vars["networkContainerID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{Code: api.NetworkContainerNotFound, Text: "Network container ID is missing", HTTPStatus: http.StatusBadRequest})
		return
	}

	if !utils.IsValidUUID(ncID) {
		setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkContainerID, Text: "Network container ID must be a UUID", HTTPStatus: http.StatusBadRequest})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

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
				setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkContainerRequest, Text: "Network Container request is invalid"})
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
		}

		ncPersistErr = ctrler.PersistNetworkContainerRequestV1(
			networkID,
			ncID,
			ncRequestV1,
			apiVersion)
	} else {
		if validationErr := ncRequestV2.Validate(); validationErr != nil {
			setRequestFailed(w, validationErr)
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

func getNetworkContainerRequestV1(body []byte) (*api.NetworkContainerRequestV1, error) {
	var ncRequestV1 api.NetworkContainerRequestV1
	if err := json.Unmarshal(body, &ncRequestV1); err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(ncRequestV1, (api.NetworkContainerRequestV1{})) {
		if len(ncRequestV1.ReservationID) != 0 {
			switch ncRequestV1.AssociationInfo.ContainerType {
			case cns.AzureContainerInstance:
				fallthrough
			case cns.WebApps:
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
	networkID, ok = vars["networkID"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ncID, ok = vars["networkContainerID"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ncConfig, getNCConfigErr := ctrler.GetNetworkContainerConfiguration(networkID, ncID)
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

// CreateNetwork - Handler for create network
func CreateNetwork(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get("api-version")
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var networkInfo api.NetworkInfoRequest
	var err error
	if err = decodeRequestBody(r, &networkInfo); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err = json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	if validationErr := networkInfo.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
		return
	}

	t := ctrler.PersistNetworkJoinRequest(networkID, networkInfo, apiVersion)
	if t != nil {
		setRequestFailed(w, t)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(t); err != nil {
		panic(err)
	}
}

// PutSubnet - Handler for put delgate subnet
func PutSubnet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID := vars["networkID"]
	subnet := vars["subnet"]
	loggershim.LS.Debugf("Subnet: %s\n", subnet)
	var subnetRequest api.SubnetRequest
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err := decodeRequestBody(r, &subnetRequest); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
		return
	}

	if validationErr := subnetRequest.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
	}

	t := ctrler.PersistSubnetDelegationRequest(networkID, strings.ToLower(subnet), subnetRequest)
	if t != nil {
		w.WriteHeader(t.HTTPStatus)
		if err := json.NewEncoder(w).Encode(&t); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// GetSubnetStatus - Handler for retrieving the subnet status
func GetSubnetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NetworkID is missing"})
		return
	}

	subnet, ok := vars["subnet"]
	loggershim.LS.Debugf("Subnet: %s\n", subnet)
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			HTTPStatus: http.StatusBadRequest,
			Text:       "SubnetName is missing"})
		return
	}

	subnetStatus, retCode, getErr := ctrler.GetSubnetStatus(
		networkID,
		strings.ToLower(subnet))

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NetworkID is missing"})
		return
	}

	subnet, ok := vars["subnet"]
	loggershim.LS.Debugf("Subnet: %s\n", subnet)
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			HTTPStatus: http.StatusBadRequest,
			Text:       "SubnetName is missing"})
		return
	}

	subnetConfiguration, getErr := ctrler.GetSubnetConfiguration(
		networkID,
		strings.ToLower(subnet))

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
	networkID = vars["networkID"]
	t := ctrler.DeleteNetwork(networkID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
	networkID, ok := vars["networkID"]
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidNetworkID,
			HTTPStatus: http.StatusBadRequest,
			Text:       "NetworkID is missing"})
		return
	}

	networkStatus, retCode, getErr := ctrler.GetNetworkStatus(
		networkID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if getErr == nil {
		w.WriteHeader(retCode)
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
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err := decodeRequestBody(r, &request); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}

		return
	}

	if validationErr := request.Validate(); validationErr != nil {
		setRequestFailed(w, validationErr)
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
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

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

// RelinquishPrimaryStatus - Relinquish replica primary status
func RelinquishPrimaryStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	ctrler.RelinquishPrimaryStatus()
	w.WriteHeader(http.StatusOK)
}
