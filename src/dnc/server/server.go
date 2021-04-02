package server

import (
	"crypto/tls"
	"dnc/configuration"
	"dnc/controller"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"log"
	"net/http"
	"net/url"

	localtls "github.com/Azure/azure-container-networking/server/tls"
	"github.com/gorilla/mux"
)

type Server struct {
	ServerSettings      configuration.ServerSettings
	Controller          *controller.Controller
	Endpoint            string
	TlsEndpoint         string
	Router              *mux.Router
	ReverseProxyEnabled bool
	ConnectionTimeout   int
	TlsConfig           *tls.Config
	Logger              *loggershim.LoggerShim
	ProbeRoutes         *ProbeRoutes
	DebugRoutes         *DebugRoutes
}

func (server *Server) Init() error {
	httpEndpoint, err := server.Controller.Replica.GetReplicaIPEndpoint()

	httpUrl, err := url.Parse(httpEndpoint)
	if err != nil {
		utils.OnErrorFail(err, "GetReplicaEndpoint failed")
	}
	server.Endpoint = httpUrl.Host

	if server.ServerSettings.UseHttps {
		tlsEndpoint, err := server.Controller.Replica.GetReplicaTLSEndpoint()
		httpsUrl, err := url.Parse(tlsEndpoint)
		if err != nil {
			utils.OnErrorFail(err, "GetReplicaTlsEndpoint failed")
		}

		server.TlsEndpoint = httpsUrl.Host
	}
	server.Router = NewRouter(server.ReverseProxyEnabled, server.ServerSettings.UseHttps, server.ProbeRoutes, server.DebugRoutes)
	SetController(server.Controller)
	server.Logger.Debugf("Host=%s, Port=%s, ReverseProxyEnabled=%t, UseHttps=%t", server.ServerSettings.IPAddress,
		server.ServerSettings.Port, server.ReverseProxyEnabled, server.ServerSettings.UseHttps)
	return nil
}

func (server *Server) Start() {
	// TODO: remove this after all partners migrate to HTTPS
	// this is to allow HTTPS and HTTP to function
	// at the same time on different ports - this is needed to allow
	// partners such as ADF to deploy HTTPS to ADF pools
	// and have their dependent service then switch to calling HTTPS
	// without building out a seperate cluster
	if server.ServerSettings.UseHttps {
		err := server.SetupTLS()
		if err != nil {
			log.Fatal(err)
		}
		server.StartWithTLS()
	}
	log.Fatal(http.ListenAndServe(server.Endpoint, server.Router))
}

// StartWithTLS : starts a tls server in the background
func (server *Server) StartWithTLS() {
	tlsServer := &http.Server{
		Addr:      server.TlsEndpoint,
		TLSConfig: server.TlsConfig,
		Handler:   server.Router,
	}

	go func() {
		log.Fatal(tlsServer.ListenAndServeTLS("", ""))
	}()
}

func (server *Server) SetupTLS() error {
	tlsSettings := localtls.TlsSettings{
		TLSCertificatePath: server.ServerSettings.TlsCertificateFilePath,
		TLSSubjectName:     server.ServerSettings.TlsCertificateSubjectName,
	}
	tlsCertRetriever, err := localtls.GetTlsCertificateRetriever(tlsSettings)
	if err != nil {
		return fmt.Errorf("Failed to get certificate retriever: %v", err)
	}

	leafCertificate, err := tlsCertRetriever.GetCertificate()

	if err != nil {
		return fmt.Errorf("Failed to get certificate: %v", err)
	}

	if leafCertificate == nil {
		return fmt.Errorf("Certificate retrival returned empty: %v", err)
	}

	server.Logger.Debugf("Retrieved leaf certificate succesfully")

	privateKey, err := tlsCertRetriever.GetPrivateKey()

	if err != nil {
		return fmt.Errorf("Failed to get certificate private key: %v", err)
	}

	server.Logger.Debugf("Retrieved private key succesfully")

	tlsCert := tls.Certificate{
		Certificate: [][]byte{leafCertificate.Raw},
		PrivateKey:  privateKey,
		Leaf:        leafCertificate,
	}

	server.TlsConfig = &tls.Config{
		MaxVersion: tls.VersionTLS12,
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			tlsCert,
		},
	}

	return nil
}
