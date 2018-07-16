// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package network

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/ebtables"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/store"
)

const (
	// Network store key.
	storeKey  = "Network"
	VlanIDKey = "VlanID"
)

type NetworkClient interface {
	CreateBridge() error
	DeleteBridge() error
	AddL2Rules(extIf *externalInterface) error
	DeleteL2Rules(extIf *externalInterface)
	SetBridgeMasterToHostInterface() error
	SetHairpinOnHostInterface(bool) error
}

type EndpointClient interface {
	AddEndpoints(epInfo *EndpointInfo) error
	AddEndpointRules(epInfo *EndpointInfo) error
	DeleteEndpointRules(ep *endpoint)
	MoveEndpointsToContainerNS(epInfo *EndpointInfo, nsID uintptr) error
	SetupContainerInterfaces(epInfo *EndpointInfo) error
	ConfigureContainerInterfacesAndRoutes(epInfo *EndpointInfo) error
	DeleteEndpoints(ep *endpoint) error
}

type NetworkMonitor struct {
	AddRulesToBeValidated    map[string]int
	DeleteRulesToBeValidated map[string]int
}

// NetworkManager manages the set of container networking resources.
type networkManager struct {
	Version            string
	TimeStamp          time.Time
	ExternalInterfaces map[string]*externalInterface
	store              store.KeyValueStore
	sync.Mutex
}

// NetworkManager API.
type NetworkManager interface {
	Initialize(config *common.PluginConfig) error
	Uninitialize()

	AddExternalInterface(ifName string, subnet string) error

	CreateNetwork(nwInfo *NetworkInfo) error
	DeleteNetwork(networkId string) error
	GetNetworkInfo(networkId string) (*NetworkInfo, error)

	CreateEndpoint(networkId string, epInfo *EndpointInfo) error
	DeleteEndpoint(networkId string, endpointId string) error
	GetEndpointInfo(networkId string, endpointId string) (*EndpointInfo, error)
	AttachEndpoint(networkId string, endpointId string, sandboxKey string) (*endpoint, error)
	DetachEndpoint(networkId string, endpointId string) error
	SetupNetworkUsingState(networkMonitor *NetworkMonitor) error
}

// Creates a new network manager.
func NewNetworkManager() (NetworkManager, error) {
	nm := &networkManager{
		ExternalInterfaces: make(map[string]*externalInterface),
	}

	return nm, nil
}

// Initialize configures network manager.
func (nm *networkManager) Initialize(config *common.PluginConfig) error {
	nm.Version = config.Version
	nm.store = config.Store

	// Restore persisted state.
	err := nm.restore()
	return err
}

// Uninitialize cleans up network manager.
func (nm *networkManager) Uninitialize() {
}

// Restore reads network manager state from persistent store.
func (nm *networkManager) restore() error {
	// Skip if a store is not provided.
	if nm.store == nil {
		log.Printf("[net] network store is nil")
		return nil
	}

	rebooted := false
	// After a reboot, all address resources are implicitly released.
	// Ignore the persisted state if it is older than the last reboot time.

	// Read any persisted state.
	nm.Lock()
	defer nm.Unlock()

	err := nm.store.Read(storeKey, nm)
	if err != nil {
		if err == store.ErrKeyNotFound {
			log.Printf("[net] network store key not found")
			// Considered successful.
			return nil
		} else {
			log.Printf("[net] Failed to restore state, err:%v\n", err)
			return err
		}
	}

	modTime, err := nm.store.GetModificationTime()
	if err == nil {
		rebootTime, err := platform.GetLastRebootTime()
		log.Printf("[net] reboot time %v store mod time %v", rebootTime, modTime)
		if err == nil && rebootTime.After(modTime) {
			rebooted = true
		}
	}

	// Populate pointers.
	for _, extIf := range nm.ExternalInterfaces {
		for _, nw := range extIf.Networks {
			nw.extIf = extIf
		}
	}

	// if rebooted recreate the network that existed before reboot.
	if rebooted {
		log.Printf("[net] Rehydrating network state from persistent store")
		for _, extIf := range nm.ExternalInterfaces {
			for _, nw := range extIf.Networks {
				nwInfo, err := nm.GetNetworkInfo(nw.Id)
				if err != nil {
					log.Printf("[net] Failed to fetch network info for network %v extif %v err %v. This should not happen", nw, extIf, err)
					return err
				}

				extIf.BridgeName = ""

				_, err = nm.newNetworkImpl(nwInfo, extIf)
				if err != nil {
					log.Printf("[net] Restoring network failed for nwInfo %v extif %v. This should not happen %v", nwInfo, extIf, err)
					return err
				}
			}
		}
	}

	log.Printf("[net] Restored state, %+v\n", nm)
	return nil
}

// Save writes network manager state to persistent store.
func (nm *networkManager) save() error {
	// Skip if a store is not provided.
	if nm.store == nil {
		return nil
	}

	// Update time stamp.
	nm.TimeStamp = time.Now()

	err := nm.store.Write(storeKey, nm)
	if err == nil {
		log.Printf("[net] Save succeeded.\n")
	} else {
		log.Printf("[net] Save failed, err:%v\n", err)
	}
	return err
}

//
// NetworkManager API
//
// Provides atomic stateful wrappers around core networking functionality.
//

// AddExternalInterface adds a host interface to the list of available external interfaces.
func (nm *networkManager) AddExternalInterface(ifName string, subnet string) error {
	nm.Lock()
	defer nm.Unlock()

	err := nm.newExternalInterface(ifName, subnet)
	if err != nil {
		return err
	}

	err = nm.save()
	if err != nil {
		return err
	}

	return nil
}

// CreateNetwork creates a new container network.
func (nm *networkManager) CreateNetwork(nwInfo *NetworkInfo) error {
	nm.Lock()
	defer nm.Unlock()

	_, err := nm.newNetwork(nwInfo)
	if err != nil {
		return err
	}

	err = nm.save()
	if err != nil {
		return err
	}

	return nil
}

// DeleteNetwork deletes an existing container network.
func (nm *networkManager) DeleteNetwork(networkId string) error {
	nm.Lock()
	defer nm.Unlock()

	err := nm.deleteNetwork(networkId)
	if err != nil {
		return err
	}

	err = nm.save()
	if err != nil {
		return err
	}

	return nil
}

// GetNetworkInfo returns information about the given network.
func (nm *networkManager) GetNetworkInfo(networkId string) (*NetworkInfo, error) {
	nm.Lock()
	defer nm.Unlock()

	nw, err := nm.getNetwork(networkId)
	if err != nil {
		return nil, err
	}

	nwInfo := &NetworkInfo{
		Id:      networkId,
		Subnets: nw.Subnets,
		Mode:    nw.Mode,
		Options: make(map[string]interface{}),
	}

	getNetworkInfoImpl(nwInfo, nw)

	if nw.extIf != nil {
		nwInfo.BridgeName = nw.extIf.BridgeName
	}

	return nwInfo, nil
}

// CreateEndpoint creates a new container endpoint.
func (nm *networkManager) CreateEndpoint(networkId string, epInfo *EndpointInfo) error {
	nm.Lock()
	defer nm.Unlock()

	nw, err := nm.getNetwork(networkId)
	if err != nil {
		return err
	}

	if nw.VlanId != 0 {
		if epInfo.Data[VlanIDKey] == nil {
			log.Printf("overriding endpoint vlanid with network vlanid")
			epInfo.Data[VlanIDKey] = nw.VlanId
		}
	}

	_, err = nw.newEndpoint(epInfo)
	if err != nil {
		return err
	}

	err = nm.save()
	if err != nil {
		return err
	}

	return nil
}

// DeleteEndpoint deletes an existing container endpoint.
func (nm *networkManager) DeleteEndpoint(networkId string, endpointId string) error {
	nm.Lock()
	defer nm.Unlock()

	nw, err := nm.getNetwork(networkId)
	if err != nil {
		return err
	}

	err = nw.deleteEndpoint(endpointId)
	if err != nil {
		return err
	}

	err = nm.save()
	if err != nil {
		return err
	}

	return nil
}

// GetEndpointInfo returns information about the given endpoint.
func (nm *networkManager) GetEndpointInfo(networkId string, endpointId string) (*EndpointInfo, error) {
	nm.Lock()
	defer nm.Unlock()

	nw, err := nm.getNetwork(networkId)
	if err != nil {
		return nil, err
	}

	ep, err := nw.getEndpoint(endpointId)
	if err != nil {
		return nil, err
	}

	return ep.getInfo(), nil
}

// AttachEndpoint attaches an endpoint to a sandbox.
func (nm *networkManager) AttachEndpoint(networkId string, endpointId string, sandboxKey string) (*endpoint, error) {
	nm.Lock()
	defer nm.Unlock()

	nw, err := nm.getNetwork(networkId)
	if err != nil {
		return nil, err
	}

	ep, err := nw.getEndpoint(endpointId)
	if err != nil {
		return nil, err
	}

	err = ep.attach(sandboxKey)
	if err != nil {
		return nil, err
	}

	err = nm.save()
	if err != nil {
		return nil, err
	}

	return ep, nil
}

// DetachEndpoint detaches an endpoint from its sandbox.
func (nm *networkManager) DetachEndpoint(networkId string, endpointId string) error {
	nm.Lock()
	defer nm.Unlock()

	nw, err := nm.getNetwork(networkId)
	if err != nil {
		return err
	}

	ep, err := nw.getEndpoint(endpointId)
	if err != nil {
		return err
	}

	err = ep.detach()
	if err != nil {
		return err
	}

	err = nm.save()
	if err != nil {
		return err
	}

	return nil
}

func (nm *networkManager) SetupNetworkUsingState(networkMonitor *NetworkMonitor) error {
	var currentEbtableRulesMap map[string]string
	var currentStateRulesMap map[string]string

	preRules, err := common.ExecuteShellCommand("ebtables -t nat -L PREROUTING --Lmac2")
	if err != nil {
		log.Printf("Error while getting prerules list")
	}

	preRulesList := strings.Split(preRules, "\n")
	log.Printf("PreRouting rule count : %v", len(preRulesList)-4)

	currentEbtableRulesMap = make(map[string]string)
	for _, rule := range preRulesList {
		rule = strings.TrimSpace(rule)
		if rule != "" && !strings.Contains(rule, "Bridge table") && !strings.Contains(rule, "Bridge chain") {
			currentEbtableRulesMap[rule] = ebtables.PreRouting
		}
	}

	postRules, err := common.ExecuteShellCommand("ebtables -t nat -L POSTROUTING --Lmac2")
	if err != nil {
		log.Printf("Error while getting postrules list")
	}

	postRulesList := strings.Split(postRules, "\n")
	log.Printf("PostRouting rule count : %v", len(postRulesList)-4)

	for _, rule := range postRulesList {
		rule = strings.TrimSpace(rule)
		if rule != "" && !strings.Contains(rule, "Bridge table") && !strings.Contains(rule, "Bridge chain") {
			currentEbtableRulesMap[rule] = ebtables.PostRouting
		}
	}

	currentStateRulesMap = nm.AddStateRulesToMap()

	nm.createRequiredL2Rules(currentEbtableRulesMap, currentStateRulesMap, networkMonitor)
	nm.removeInvalidL2Rules(currentEbtableRulesMap, currentStateRulesMap, networkMonitor)
	return nil
}

func (nm *networkManager) AddStateRulesToMap() map[string]string {
	rulesMap := make(map[string]string)

	for _, extIf := range nm.ExternalInterfaces {
		arpDnatKey := fmt.Sprintf("-p ARP -i %s --arp-op Reply -j dnat --to-dst ff:ff:ff:ff:ff:ff --dnat-target ACCEPT", extIf.Name)
		rulesMap[arpDnatKey] = ebtables.PreRouting

		snatKey := fmt.Sprintf("-s Unicast -o %s -j snat --to-src %s --snat-arp --snat-target ACCEPT", extIf.Name, extIf.MacAddress.String())
		rulesMap[snatKey] = ebtables.PostRouting

		for _, nw := range extIf.Networks {
			for _, ep := range nw.Endpoints {
				for _, ipAddr := range ep.IPAddresses {
					arpReplyKey := fmt.Sprintf("-p ARP --arp-op Request --arp-ip-dst %s -j arpreply --arpreply-mac %s", ipAddr.IP.String(), ep.MacAddress.String())
					rulesMap[arpReplyKey] = ebtables.PreRouting

					dnatMacKey := fmt.Sprintf("-p IPv4 -i %s --ip-dst %s -j dnat --to-dst %s --dnat-target ACCEPT", extIf.Name, ipAddr.IP.String(), ep.MacAddress.String())
					rulesMap[dnatMacKey] = ebtables.PreRouting
				}
			}
		}
	}

	return rulesMap
}

func DeleteRulesNotExistInMap(networkMonitor *NetworkMonitor, chainRules map[string]string, stateRules map[string]string) {
	for rule, chain := range chainRules {
		if _, ok := stateRules[rule]; !ok {
			if itr, ok := networkMonitor.DeleteRulesToBeValidated[rule]; ok && itr > 0 {
				log.Printf("Deleting Ebtable rule as it didn't exist in state for %d iterations chain %v rule %v", itr, chain, rule)
				if err := ebtables.DeleteEbtableRule(chain, rule); err != nil {
					log.Printf("Error while deleting ebtable rule %v", err)
				}

				delete(networkMonitor.DeleteRulesToBeValidated, rule)
			} else {
				log.Printf("[DELETE] Found unmatched rule chain %v rule %v itr %d. Giving one more iteration", chain, rule, itr)
				networkMonitor.DeleteRulesToBeValidated[rule] = itr + 1
			}
		}
	}
}

func (nm *networkManager) removeInvalidL2Rules(
	currentEbtableRulesMap map[string]string,
	currentStateRulesMap map[string]string,
	networkMonitor *NetworkMonitor) error {

	if nm.ExternalInterfaces == nil {
		return fmt.Errorf("[Azure-CNIMonitor] Nothing to delete")
	}

	for rule := range networkMonitor.DeleteRulesToBeValidated {
		if _, ok := currentEbtableRulesMap[rule]; !ok {
			delete(networkMonitor.DeleteRulesToBeValidated, rule)
		}
	}

	DeleteRulesNotExistInMap(networkMonitor, currentEbtableRulesMap, currentStateRulesMap)
	return nil
}

func AddRulesNotExistInMap(networkMonitor *NetworkMonitor, stateRules map[string]string, chainRules map[string]string) {
	for rule, chain := range stateRules {
		if _, ok := chainRules[rule]; !ok {
			if itr, ok := networkMonitor.AddRulesToBeValidated[rule]; ok && itr > 0 {
				log.Printf("Adding Ebtable rule as it didn't exist in state for %d iterations chain %v rule %v", itr, chain, rule)
				if err := ebtables.AddEbtableRule(chain, rule); err != nil {
					log.Printf("Error while deleting ebtable rule %v", err)
				}

				delete(networkMonitor.AddRulesToBeValidated, rule)
			} else {
				log.Printf("[ADD] Found unmatched rule chain %v rule %v itr %d. Giving one more iteration", chain, rule, itr)
				networkMonitor.AddRulesToBeValidated[rule] = itr + 1
			}
		}
	}
}

func (nm *networkManager) createRequiredL2Rules(
	currentEbtableRulesMap map[string]string,
	currentStateRulesMap map[string]string,
	networkMonitor *NetworkMonitor) error {

	if nm.ExternalInterfaces == nil {
		return fmt.Errorf("[Azure-CNIMonitor] Nothing to add")
	}

	for rule := range networkMonitor.AddRulesToBeValidated {
		if _, ok := currentStateRulesMap[rule]; !ok {
			delete(networkMonitor.AddRulesToBeValidated, rule)
		}
	}

	AddRulesNotExistInMap(networkMonitor, currentStateRulesMap, currentEbtableRulesMap)
	return nil
}
