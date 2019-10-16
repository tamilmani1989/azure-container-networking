package main

import (
	"dnc/configuration"
	"dnc/nmagentcontracts"
	"dnc/test/utils"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type NMAgent struct {
	Server       *httptest.Server
	ResponseFunc func(http.ResponseWriter, *http.Request)
	Lock         sync.Mutex
}

var (
	nmagent    = new(NMAgent)
	config     *configuration.Configuration
	regVnet, _ = regexp.Compile("/joinedVirtualNetworks/(.*)/api-version")
	regNC, _   = regexp.Compile("/networkContainers/(.*)/authenticationToken")
)

func init() {
	// use DNC config set with test storage settings
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	if config.HostAgentSettings.WireServerIP == "" {
		panic("Mock NMAgent EP not set")
	}

	configuration.SetDncConfigDefaults(config)
	nmagent.ResponseFunc = responder
	nmagent.setupServer(config.HostAgentSettings.WireServerIP)
}

func main() {
	nmagent.Server.Start()
	select {}
}

func (s *NMAgent) setupServer(endpoint string) {
	s.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.ResponseFunc != nil {
				s.ResponseFunc(w, r)
			}
		}))
	l, _ := net.Listen("tcp", endpoint)
	s.Server.Listener = l
}

func responder(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		handlePost(w, r)
	case "GET":
		handleGet(w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "networkContainers") {
		responseCode := "200"
		ncID := regNC.FindStringSubmatch(r.RequestURI)
		if len(ncID) < 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch ncID[1] {
		case utils.OneMinuteFailureNC:
			responseCode = "500"
		case utils.ThrottledNC:
			responseCode = "403"
		case utils.UnauthorizedNC:
			responseCode = "401"
		}

		// delete NC call
		if strings.Contains(r.RequestURI, "DELETE") {
			json.NewEncoder(w).Encode(nmagentcontracts.DeleteNetworkContainerResponse{ResponseCode: responseCode})
		} else {
			// create NC call
			var ncReq nmagentcontracts.CreateOrUpdateNetworkContainerRequest
			json.NewDecoder(r.Body).Decode(&ncReq)
			json.NewEncoder(w).Encode(nmagentcontracts.CreateNetworkContainerResponse{ResponseCode: responseCode, Version: ncReq.Version})
		}
	} else if strings.Contains(r.RequestURI, "joinedVirtualNetworks") {
		responseCode := http.StatusOK
		vnet := regVnet.FindStringSubmatch(r.RequestURI)
		if len(vnet) < 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch vnet[1] {
		case utils.OneMinuteFailureVnet:
			responseCode = http.StatusInternalServerError
		case utils.ThrottledVnet:
			responseCode = http.StatusForbidden
		case utils.PendingVnet:
			responseCode = http.StatusProcessing
		}

		w.WriteHeader(responseCode)
		return
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "joinedVirtualNetworks") {
		var (
			subnets = []nmagentcontracts.Subnet{
				nmagentcontracts.Subnet{
					SubnetName:     utils.Subnet1,
					AddressPrefix:  utils.Subnet1AddrSpace,
					DefaultGateway: utils.Subnet1DefaultGW},
				nmagentcontracts.Subnet{
					SubnetName:     utils.Subnet2,
					AddressPrefix:  utils.Subnet2AddrSpace,
					DefaultGateway: utils.Subnet2DefaultGW},
				nmagentcontracts.Subnet{
					SubnetName:     utils.Subnet3,
					AddressPrefix:  utils.Subnet3AddrSpace,
					DefaultGateway: utils.Subnet3DefaultGW}}
		)

		subnets[0].Tags = getTags(1)

		vnetResponse := nmagentcontracts.VirtualNetworkResponse{
			ResponseCode: strconv.Itoa(http.StatusOK),
			CNetSpace:    utils.DefaultCnetSpace,
			DNSServers:   utils.DefaultDNSServers,
			VNetSpace:    utils.DefaultVnetSpace,
			VnetVersion:  "1",
			Subnets:      subnets}

		json.NewEncoder(w).Encode(vnetResponse)
	} else if strings.Contains(r.RequestURI, "systemTags") {
		tagName := strings.Split(r.URL.Path, "/")[2]
		tagResponse, ok := updatedTagDefinitions[tagName]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(tagResponse)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

var updatedTagDefinitions = map[string]nmagentcontracts.TagResponse{
	"XStore":   nmagentcontracts.TagResponse{ResponseCode: "200", Name: "XStore", Values: []string{"167.10.10.0/24", "168.10.10.0/24", "144.10.12.12/24"}},
	"AzureSql": nmagentcontracts.TagResponse{ResponseCode: "200", Name: "AzureSql", Values: []string{"132.10.10.0/24", "133.10.10.0/24", "134.10.12.12/24"}}}

func getTags(tagCount int) []nmagentcontracts.Tag {
	var (
		tagsArray = make([]nmagentcontracts.Tag, tagCount)
		index     = 0
	)
	for tagName := range updatedTagDefinitions {
		if index < tagCount {
			tagsArray[index] = nmagentcontracts.Tag{Name: tagName, Type: "System"}
			index++
		} else {
			break
		}
	}

	return tagsArray
}
