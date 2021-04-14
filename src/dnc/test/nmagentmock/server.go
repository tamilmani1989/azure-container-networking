package nmagentmock

import (
	"dnc/nmagentcontracts"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const defaultAddr = "localhost:9001"

type Server struct {
	joinedVNets map[string]nmagentcontracts.VirtualNetworkResponse
	ncs         map[string]nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	m           sync.RWMutex
	httpS       *http.Server
}

func NewServer(address string) *Server {
	if address == "" {
		address = defaultAddr
	}

	s := Server{
		joinedVNets: make(map[string]nmagentcontracts.VirtualNetworkResponse),
		ncs:         make(map[string]nmagentcontracts.CreateOrUpdateNetworkContainerRequest),
		m:           sync.RWMutex{},
	}

	r := mux.NewRouter()
	r.SkipClean(true)
	s.registerRoutes(r)
	s.httpS = &http.Server{Addr: address, Handler: r}

	return &s
}

func (s *Server) Start() chan error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.httpS.ListenAndServe()
	}()

	logrus.Infof("mock nmagent started on %s", s.httpS.Addr)
	return errChan
}

func (s *Server) Close() error {
	return s.httpS.Close()
}

func (s *Server) GetVnetResponse(vnetID string) (nmagentcontracts.VirtualNetworkResponse, bool) {
	s.m.RLock()
	resp, ok := s.joinedVNets[vnetID]
	s.m.RUnlock()
	return resp, ok
}

func (s *Server) StoreVnetResponses(responses map[string]nmagentcontracts.VirtualNetworkResponse) {
	s.m.Lock()
	for vnetID, response := range responses {
		s.joinedVNets[vnetID] = response
	}
	s.m.Unlock()
}

func (s *Server) StoreNCRequests(requests map[string]nmagentcontracts.CreateOrUpdateNetworkContainerRequest) {
	s.m.Lock()
	for ncID, request := range requests {
		s.ncs[ncID] = request
	}
	s.m.Unlock()
}

const machinePluginsRoute = "/machine/plugins/"

func (s *Server) registerRoutes(r *mux.Router) {
	r.HandleFunc(machinePluginsRoute, s.MachinePluginsGET).Methods(http.MethodGet)
	r.HandleFunc(machinePluginsRoute, s.MachinePluginsPOST).Methods(http.MethodPost)
}

var (
	joinedNetworksRegex   = regexp.MustCompile(`NetworkManagement/joinedVirtualNetworks/(.*?)/api-version/1`)
	networkContainerRegex = regexp.MustCompile(`NetworkManagement/interfaces/(.*?)/networkContainers/(.*?)/authenticationToken/(.*?)/api-version/1`)
)

func (s *Server) MachinePluginsGET(w http.ResponseWriter, r *http.Request) {
	nmagentRoute := r.URL.Query().Get("type")
	if nmagentRoute == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	matches := joinedNetworksRegex.FindStringSubmatch(nmagentRoute)
	if len(matches) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vnetID := matches[1]

	resp, found := s.GetVnetResponse(vnetID)
	if !found {
		logrus.Errorf("could not find response for vnet id %v", vnetID)
		w.WriteHeader(http.StatusNotFound) // todo: not sure what the actual response would be
		return
	}

	respondJSON(w, resp)
}

func (s *Server) MachinePluginsPOST(w http.ResponseWriter, r *http.Request) {
	nmagentRoute := r.URL.Query().Get("type")
	if nmagentRoute == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	postNCMatches := networkContainerRegex.FindStringSubmatch(nmagentRoute)
	if len(postNCMatches) == 4 { // todo: nothing done with interface (matches[1]) id and auth token (matches[3]) at the moment
		ncID := postNCMatches[2]
		s.postNetworkContainer(ncID, w, r)
		return
	}

	joinVnetMatches := joinedNetworksRegex.FindStringSubmatch(nmagentRoute)
	if len(joinVnetMatches) == 2 { // todo: nothing done with vnet ID (matches[1]) at the moment
		s.joinVirtualNetwork(w, r)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	return
}

func (s *Server) postNetworkContainer(ncID string, w http.ResponseWriter, r *http.Request) {
	var payload nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.StoreNCRequests(map[string]nmagentcontracts.CreateOrUpdateNetworkContainerRequest{ncID: payload})

	resp := nmagentcontracts.CreateNetworkContainerResponse{
		ResponseCode:       strconv.Itoa(http.StatusOK),
		NetworkContainerID: ncID,
		Version:            payload.Version,
	}

	respondJSON(w, resp)
}

func (s *Server) joinVirtualNetwork(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.WriteHeader(http.StatusOK)
}

const (
	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json"
)

func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Add(contentTypeHeader, jsonContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logrus.Errorf("could not encode type %T: %v", v, err)
	}
}
