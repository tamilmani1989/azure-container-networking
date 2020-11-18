package main

import (
	"crypto/tls"
	localtls "github.com/Azure/azure-container-networking/server/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const defaultAddr = "localhost:10090"

// Small mock of CNS for happy path testing of DNC API
func main() {
	addrFlag := flag.String("address", defaultAddr, "host:port")
	tlsCertificatePath := flag.String("tlsCertificatePath", "", "Directory to PFX")
	flag.Parse()

	r := mux.NewRouter()
	r.SkipClean(true)
	registerRoutes(r)

	srv := http.Server{Addr: *addrFlag, Handler: r}

	if *tlsCertificatePath != ""{
		logrus.Infof("cns mock tls server listening on %s", srv.Addr)
		logrus.Fatal(StartTlsServer(*tlsCertificatePath,*addrFlag, r))
	}else{
		logrus.Infof("cns mock server listening on %s", srv.Addr)
		logrus.Fatal(srv.ListenAndServe())
	}
}

func StartTlsServer(tlsCertificatePath, address string, router *mux.Router) error {
	settings := localtls.TlsSettings{
		TLSCertificatePath: tlsCertificatePath,
	}
	tlsCertRetriever, err := localtls.GetTlsCertificateRetriever(settings)
	if err != nil {
		return fmt.Errorf("Failed to get certificate retriever", err)
	}

	leafCertificate, err := tlsCertRetriever.GetCertificate()

	if err != nil {
		return fmt.Errorf("Failed to get certificate", err)
	}

	logrus.Infof("Retrieved Leaf certificate")

	privateKey, err := tlsCertRetriever.GetPrivateKey()

	if err != nil {
		return fmt.Errorf("Failed to get certificate private key", err)
	}

	logrus.Infof("Retrieved Private Key")
	tlsCert := tls.Certificate{
		Certificate: [][]byte{leafCertificate.Raw},
		PrivateKey:  privateKey,
		Leaf:        leafCertificate,
	}

	tlsConfig := &tls.Config{
		MaxVersion: tls.VersionTLS12,
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			tlsCert,
		},
	}

	tlsServer :=  &http.Server{
		Addr:      address,
		TLSConfig: tlsConfig,
		Handler: router,
	}

	return tlsServer.ListenAndServeTLS("", "")
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
