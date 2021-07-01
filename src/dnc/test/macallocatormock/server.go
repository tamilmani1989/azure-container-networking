package macallocatormock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	defaultAddr        = "localhost:5012"
	contentTypeHeader  = "Content-Type"
	jsonContentType    = "application/json"
	startingMAC        = "001111AAAA"
	allocateMacRoute   = "/macallocator/allocate"
	deallocateMacRoute = "/macallocator/deallocate"
	healthRoute        = "/macallocator/health"
)

type macAllocatorResponse struct {
	AllocationId string `json:"AllocationId"`
	MacAddress   string `json:"MacAddress"`
	IsSuccess    bool   `json:"IsSuccess"`
	ErrorMessage string `json:"ErrorMessage"`
	ShouldRetry  bool   `json:"ShouldRetry"`
}

type healthResponse struct {
	Status string `json:"Status"`
}

type macAllocatorRequest struct {
	AllocationId string
	Type         string
	Tag          string
}

type Server struct {
	macMap  map[string]string
	macList []string
	m       sync.Mutex
	httpS   *http.Server
}

func NewServer(address string) *Server {
	if address == "" {
		address = defaultAddr
	}

	s := Server{
		//dict to store AllocationID:MAC mapping
		macMap: make(map[string]string),
		//slice to store the MACs
		macList: make([]string, 0),
	}

	r := mux.NewRouter()
	r.SkipClean(true)
	s.registerRoutes(r)
	s.httpS = &http.Server{Addr: address, Handler: r}

	return &s
}

func (s *Server) Start() error {
	mac := startingMAC
	for i := 10; i < 60; i++ {
		s.macList = append(s.macList, mac+strconv.Itoa(i))
	}
	logrus.Infof("starting macallocator mock server on %s", s.httpS.Addr)
	return s.httpS.ListenAndServe()
}

func (s *Server) Close() error {
	return s.httpS.Close()
}

func (s *Server) registerRoutes(r *mux.Router) {
	r.HandleFunc(allocateMacRoute, s.allocate).Methods(http.MethodPost)
	r.HandleFunc(deallocateMacRoute, s.deallocate).Methods(http.MethodPost)
	r.HandleFunc(healthRoute, s.health).Methods(http.MethodGet)
}

func (s *Server) validateMACRequest(w http.ResponseWriter, r *http.Request) (macAllocatorRequest, bool) {
	//decode macallocatorRequestBody to extract allocation ID
	decoder := json.NewDecoder(r.Body)
	var macRequest macAllocatorRequest
	err := decoder.Decode(&macRequest)
	if err != nil {
		logrus.Errorf("Bad HTTP Request could not decode request body %v", err)
		w.Header().Add(contentTypeHeader, jsonContentType)
		w.WriteHeader(http.StatusBadRequest)
		resp := macAllocatorResponse{
			AllocationId: macRequest.AllocationId,
			MacAddress:   "",
			IsSuccess:    false,
			ErrorMessage: "Bad HTTP Request could not decode request body",
			ShouldRetry:  false,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logrus.Errorf("could not encode type %T: %v", resp, err)
		}
		return macRequest, false
	}

	allocateID := macRequest.AllocationId
	//empty allocationID - Bad request
	if allocateID == "" {
		logrus.Errorf("Bad HTTP Request, Allocation ID is empty")
		w.Header().Add(contentTypeHeader, jsonContentType)
		w.WriteHeader(http.StatusBadRequest)
		resp := macAllocatorResponse{
			AllocationId: macRequest.AllocationId,
			MacAddress:   "",
			IsSuccess:    false,
			ErrorMessage: "Bad HTTP Request, Allocation ID is empty",
			ShouldRetry:  false,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logrus.Errorf("could not encode type %T: %v", resp, err)
		}
		return macRequest, false
	}

	return macRequest, true
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{Status: "healthy"}
	w.Header().Add(contentTypeHeader, jsonContentType)
	w.WriteHeader(http.StatusOK)
	logrus.Infof("Health check api")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logrus.Errorf("could not encode type %T: %v", resp, err)
	}
}

func (s *Server) allocate(w http.ResponseWriter, r *http.Request) {
	macRequest, validRequest := s.validateMACRequest(w, r)
	//if request is invalid response was already returned in validateMACRequest
	if !validRequest {
		return
	}

	resp := s.GetMacAllocatorResponse(macRequest.AllocationId)
	if !resp.IsSuccess {
		logrus.Errorf("Could not allocate MAC: %s", resp.ErrorMessage)
		respondJSON(w, resp)
		return
	}

	logrus.Infof("MACAddress %s successfully allocated for %s", resp.MacAddress, macRequest.AllocationId)
	respondJSON(w, resp)
}

func (s *Server) deallocate(w http.ResponseWriter, r *http.Request) {
	macRequest, validRequest := s.validateMACRequest(w, r)
	//if request is invalid response was already returned in validateMACRequest
	if !validRequest {
		return
	}

	resp, macaddr := s.GetDeallocatorResponse(macRequest.AllocationId)
	if !resp.IsSuccess {
		logrus.Errorf("Could not allocate MAC: %s", resp.ErrorMessage)
		respondJSON(w, resp)
		return
	}

	logrus.Infof("MACAddress %s successfully deallocated for %s", macaddr, macRequest.AllocationId)
	respondJSON(w, resp)
}

func (s *Server) GetMacAllocatorResponse(allocateID string) macAllocatorResponse {
	s.m.Lock()
	defer s.m.Unlock()

	var resp = macAllocatorResponse{
		AllocationId: allocateID,
		MacAddress:   "",
		IsSuccess:    false,
		ErrorMessage: "",
		ShouldRetry:  false,
	}

	//if already allocated return response
	if _, found := s.macMap[allocateID]; found {
		resp.MacAddress = s.macMap[allocateID]
		resp.IsSuccess = true
		logrus.Infof("MAC %s is already allocated for %s", s.macMap[allocateID], allocateID)
		return resp
	}

	//not found in macmap so allocate mac here
	if len(s.macList) < 1 {
		resp.ErrorMessage = "Not enough MACs in list to allocate"
		resp.ShouldRetry = true
	} else {
		var macAddress string
		//take last address in maclist and remove it from slice
		macAddress = s.macList[len(s.macList)-1]
		s.macList = s.macList[:len(s.macList)-1]
		//add mac entry in the macaddress map
		s.macMap[allocateID] = macAddress
		//set response
		resp.MacAddress = macAddress
		resp.IsSuccess = true
	}
	return resp
}

func (s *Server) GetDeallocatorResponse(allocateID string) (macAllocatorResponse, string) {
	s.m.Lock()
	defer s.m.Unlock()

	mac, exists := s.macMap[allocateID]
	var resp = macAllocatorResponse{
		AllocationId: allocateID,
		MacAddress:   "",
		IsSuccess:    false,
		ErrorMessage: "",
		ShouldRetry:  false,
	}

	if exists {
		//delete entry from map
		delete(s.macMap, allocateID)
		//add back MAC address to the macList
		s.macList = append(s.macList, mac)
		resp.IsSuccess = true
	} else {
		logrus.Infof("MAC %s was not allocated to %s", mac, allocateID)
		resp.IsSuccess = true
	}
	return resp, mac
}

func respondJSON(w http.ResponseWriter, v macAllocatorResponse) {
	w.Header().Add(contentTypeHeader, jsonContentType)
	if v.IsSuccess {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logrus.Errorf("could not encode type %T: %v", v, err)
	}
}
