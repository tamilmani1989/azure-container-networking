package main

import (
	"encoding/json"
	"flag"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const defaultAddr = "localhost:10090"

// Small mock of CNS for happy path testing of DNC API
func main() {
	addrFlag := flag.String("address", defaultAddr, "host:port")
	flag.Parse()

	r := mux.NewRouter()
	registerRoutes(r)

	srv := http.Server{Addr: *addrFlag, Handler: r}

	logrus.Infof("cns mock server listening on %s", srv.Addr)
	logrus.Fatal(srv.ListenAndServe())
}

// todo: add routes as needed
func registerRoutes(r *mux.Router) {
	r.HandleFunc("/network/setorchestratortype", setOrchestratorType).Methods(http.MethodPost)
	r.HandleFunc("/network/createorupdatenetworkcontainer", upsertNetworkContainer).Methods(http.MethodPost)
	r.HandleFunc("/network/deletenetworkcontainer", deleteNetworkContainer).Methods(http.MethodPost)
	r.HandleFunc("/hostcpucores", getNumOfCPUCores).Methods(http.MethodGet)
}

func setOrchestratorType(w http.ResponseWriter, _ *http.Request) {
	resp := success()

	respondJSON(w, resp)
}

func getNumOfCPUCores(w http.ResponseWriter, _ *http.Request) {
	resp := cns.NumOfCPUCoresResponse{NumOfCPUCores: 16}

	respondJSON(w, resp)
}

func upsertNetworkContainer(w http.ResponseWriter, _ *http.Request) {
	resp := cns.CreateNetworkContainerResponse{
		Response: success(),
	}

	respondJSON(w, resp)
}

func deleteNetworkContainer(w http.ResponseWriter, _ *http.Request) {
	resp := cns.DeleteNetworkContainerResponse{
		Response: success(),
	}

	respondJSON(w, resp)
}

func success() cns.Response {
	return cns.Response{ReturnCode: 0, Message: "success!"}
}

const (
	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json"
)

func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Add(contentTypeHeader, jsonContentType)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}
