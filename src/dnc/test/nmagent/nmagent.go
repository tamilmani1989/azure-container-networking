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
	errdVnets    sync.Map
	errdNcs      sync.Map
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
	nmagent.errdVnets.Store(utils.GenericFailureVnet, 2)
	nmagent.errdVnets.Store(utils.ThrottledVnet, 2)
	nmagent.errdVnets.Store(utils.PendingVnet, 2)
	nmagent.errdNcs.Store(utils.GenericFailureNC, 2)
	nmagent.errdNcs.Store(utils.ThrottledNC, 2)
	nmagent.errdNcs.Store(utils.UnauthorizedNC, 2)
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
		ncID := regNC.FindStringSubmatch(r.RequestURI)
		if len(ncID) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// delete NC call
		if strings.Contains(r.RequestURI, "DELETE") {
			json.NewEncoder(w).Encode(nmagentcontracts.DeleteNetworkContainerResponse{ResponseCode: generateNcResponseCode(ncID[1])})
		} else {
			// create NC call
			var ncReq nmagentcontracts.CreateOrUpdateNetworkContainerRequest
			json.NewDecoder(r.Body).Decode(&ncReq)
			json.NewEncoder(w).Encode(nmagentcontracts.CreateNetworkContainerResponse{ResponseCode: generateNcResponseCode(ncID[1]), Version: ncReq.Version})
		}
	} else if strings.Contains(r.RequestURI, "joinedVirtualNetworks") {
		vnet := regVnet.FindStringSubmatch(r.RequestURI)
		if len(vnet) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(generateVnetResponseCode(vnet[1], true))
		return
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "joinedVirtualNetworks") {
		var (
			vnet = regVnet.FindStringSubmatch(r.RequestURI)
		)

		if len(vnet) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

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
			ResponseCode: strconv.Itoa(generateVnetResponseCode(vnet[1], false)),
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
	"XStore":   nmagentcontracts.TagResponse{ResponseCode: "200", Name: "XStore", Values: []string{"167.10.10.0/24", "168.10.10.0/24", "144.10.12.0/24"}},
	"AzureSql": nmagentcontracts.TagResponse{ResponseCode: "200", Name: "AzureSql", Values: []string{"132.10.10.0/24", "133.10.10.0/24", "134.10.12.0/24"}}}

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

func generateNcResponseCode(ncID string) (responseCode string) {
	responseCode = "200"
	switch ncID {
	case utils.GenericFailureNC:
		res, _ := nmagent.errdNcs.Load(utils.GenericFailureNC)
		count := res.(int)
		if count > 0 {
			responseCode = "500"
			nmagent.errdNcs.Store(utils.GenericFailureNC, count-1)
		}
	case utils.ThrottledNC:
		res, _ := nmagent.errdNcs.Load(utils.ThrottledNC)
		count := res.(int)
		if count > 0 {
			responseCode = "403"
			nmagent.errdNcs.Store(utils.ThrottledNC, count-1)
		}
	case utils.UnauthorizedNC:
		res, _ := nmagent.errdNcs.Load(utils.UnauthorizedNC)
		count := res.(int)
		if count > 0 {
			responseCode = "401"
			nmagent.errdNcs.Store(utils.UnauthorizedNC, count-1)
		}
	}

	return
}

func generateVnetResponseCode(vnetID string, isPost bool) (responseCode int) {
	responseCode = http.StatusOK
	if isPost {
		switch vnetID {
		case utils.GenericFailureVnet:
			res, _ := nmagent.errdVnets.Load(utils.GenericFailureVnet)
			count := res.(int)
			if count > 0 {
				responseCode = http.StatusInternalServerError
				nmagent.errdVnets.Store(utils.GenericFailureVnet, count-1)
			}
		case utils.ThrottledVnet:
			res, _ := nmagent.errdVnets.Load(utils.ThrottledVnet)
			count := res.(int)
			if count > 0 {
				responseCode = http.StatusForbidden
				nmagent.errdVnets.Store(utils.ThrottledVnet, count-1)
			}
		}
	} else {
		switch vnetID {
		case utils.PendingVnet:
			res, _ := nmagent.errdVnets.Load(utils.PendingVnet)
			count := res.(int)
			if count > 0 {
				responseCode = http.StatusProcessing
				nmagent.errdVnets.Store(utils.PendingVnet, count-1)
			}
		}
	}

	return
}
