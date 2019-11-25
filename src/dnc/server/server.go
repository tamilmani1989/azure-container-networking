package server

import (
	"dnc/configuration"
	"dnc/controller"
	"dnc/utils"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	ServerSettings      configuration.ServerSettings
	Controller          *controller.Controller
	Endpoint            string
	Router              *mux.Router
	ReverseProxyEnabled bool
	ConnectionTimeout   int
}

func (server *Server) Init() error {
	var err error
	server.Endpoint, err = server.Controller.Replica.GetReplicaIPEndpoint()
	utils.OnErrorFail(err, "GetReplicaIPEndpoint failed")
	server.Router = NewRouter(server.ReverseProxyEnabled, server.ConnectionTimeout)
	SetController(server.Controller)
	return nil
}

func (server *Server) Start() {
	log.Fatal(http.ListenAndServe(server.Endpoint, server.Router))
}
