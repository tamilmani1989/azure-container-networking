package server

import (
	"dnc/api"
	"dnc/controller"
	"dnc/logger/loggershim"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

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

	var nodeinfo api.NodeInfoRequest
	body, readErr := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if readErr != nil {
		panic(readErr)
	}

	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.Unmarshal(body, &nodeinfo); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	putNodeInfoErr := ctrler.PersistNodeInfo(nodeID, &nodeinfo, apiVersion)
	if putNodeInfoErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(putNodeInfoErr); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

// GetNode - Handler for get node information
func GetNode(w http.ResponseWriter, r *http.Request) {
	// apiVersion := r.URL.Query().Get("api-version")
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

	if deleteNodeErr == nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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

	body, readErr := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if readErr != nil {
		panic(readErr)
	}

	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	var allocationRequest api.AllocationRequest

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.Unmarshal(body, &allocationRequest); err != nil {
		jsonErr := api.JsonErr{
			Code: api.JsonBodyUnMarshalFailed,
			Text: fmt.Sprintf("Failed to unmarshal request body. Error:%s", err.Error())}

		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(jsonErr); err != nil {
			panic(err)
		}
	}

	reservationErr := ctrler.PersistIPReservationRequest(
		networkID,
		reservationID,
		allocationRequest,
		apiVersion)

	if reservationErr == nil {
		w.WriteHeader(http.StatusCreated)
		return
	}

	setRequestFailed(w, reservationErr)
}

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

func CreateNetworkContainer(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get("api-version")

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

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	var ncRequestV2 *api.NetworkContainerRequestV2
	var ncRequestV1 *api.NetworkContainerRequestV1
	var isV1Request = false

	ncRequestV1, err = getNetworkContainerRequestV1(body)
	if err == nil {
		if ncRequestV1 == nil {
			ncRequestV2, err = getNetworkContainerRequestV2(body)
			if ncRequestV2 == nil {
				setRequestFailed(w, &api.JsonErr{Code: api.InvalidNetworkID, Text: "Network Container request is invalid"})
			}
		} else {
			isV1Request = true
		}
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusUnprocessableEntity) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	var ncPersistErr *api.JsonErr
	if isV1Request {
		ncPersistErr = ctrler.PersistNetworkContainerRequestV1(
			networkID,
			ncID,
			ncRequestV1,
			apiVersion)
	} else {
		ncPersistErr = ctrler.PersistNetworkContainerRequestV2(
			networkID,
			ncID,
			ncRequestV2,
			apiVersion)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if ncPersistErr != nil {
		setRequestFailed(w, ncPersistErr)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

func getNetworkContainerRequestV1(body []byte) (*api.NetworkContainerRequestV1, error) {
	var ncRequestV1 api.NetworkContainerRequestV1
	if err := json.Unmarshal(body, &ncRequestV1); err != nil {
		return nil, err
	}

	if ncRequestV1 != (api.NetworkContainerRequestV1{}) {
		if len(ncRequestV1.ReservationID) != 0 {
			return &ncRequestV1, nil
		}
	}

	return nil, nil
}

func getNetworkContainerRequestV2(body []byte) (*api.NetworkContainerRequestV2, error) {
	var ncRequestV2 api.NetworkContainerRequestV2

	if err := json.Unmarshal(body, &ncRequestV2); err != nil {
		return nil, err
	}

	if ncRequestV2 != (api.NetworkContainerRequestV2{}) {
		return &ncRequestV2, nil
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
		w.WriteHeader(http.StatusBadRequest)
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

func CreateNetwork(w http.ResponseWriter, r *http.Request) {
	apiVersion := r.URL.Query().Get("api-version")

	vars := mux.Vars(r)
	networkID, ok := vars["networkID"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var networkInfo api.NetworkInfoRequest
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &networkInfo); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	if networkInfo.NetworkType != api.AzureNet {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusUnprocessableEntity) // unprocessable entity
		err = fmt.Errorf("Only AzureVnet type is supported")
		err = json.NewEncoder(w).Encode(err)
		if err != nil {
			panic(err)
		}
		return
	}
	t := ctrler.PersistNetworkInfo(networkID, networkInfo, apiVersion)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if t != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(t); err != nil {
		panic(err)
	}
}

func PutSubnet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	networkID := vars["networkID"]
	subnet := vars["subnet"]
	loggershim.LS.Log("Subnet: %s\n", subnet)
	var subnetRequest api.SubnetRequest
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &subnetRequest); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}
	t := ctrler.PersistSubnetInfo(networkID, subnet, subnetRequest)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if t != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(&t); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

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
	loggershim.LS.Log("Subnet: %s\n", subnet)
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			HTTPStatus: http.StatusBadRequest,
			Text:       "SubnetName is missing"})
		return
	}

	subnetStatus, retCode, getErr := ctrler.GetSubnetStatus(
		networkID,
		subnet)

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
	loggershim.LS.Log("Subnet: %s\n", subnet)
	if !ok {
		setRequestFailed(w, &api.JsonErr{
			Code:       api.InvalidSubnetName,
			HTTPStatus: http.StatusBadRequest,
			Text:       "SubnetName is missing"})
		return
	}

	subnetConfiguration, getErr := ctrler.GetSubnetConfiguration(
		networkID,
		subnet)

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
