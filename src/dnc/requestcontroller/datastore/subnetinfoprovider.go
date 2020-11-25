// Copyright 2020 Microsoft. All rights reserved.

package datastore

import (
	"context"
	"dnc/configuration"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/requestcontroller"
	"fmt"
	"net/http"
	"sync"
)

const (
	subnetTokenMapTableName    = "SubnetTokenMap"
	primaryTokenPropertyName   = "PrimaryTokenId"
	secondaryTokenPropertyName = "SecondaryTokenId"
	vnetGUIDPropertyName       = "VnetResourceGuid"
)

type SubnetInfoProvider struct {
	dataStoreShim *datastoreshim.DataStoreShim
	mutex         sync.Mutex
	cache         map[string]map[string]requestcontroller.SubnetInfo // delegatedSubnetID -> {subnetName -> SubnetInfo, subnetName -> SubnetInfo}
}

var _ requestcontroller.SubnetInfoProvider = &SubnetInfoProvider{}

/*
Table:            SubnetTokenMap
PartitionKey:     <DelegatedSubnetID>
RowKey:           <SubnetName>
PrimaryTokenId:   <Token>
SecondaryTokenId: <Token> (optional)
VnetResourceGuid: <VnetID>
*/

func NewSubnetInfoProvider(dataStoreSettings configuration.DataStoreSettings, loggingConfig requestcontroller.LoggingConfig) *SubnetInfoProvider {
	//datastoreshim has a dependency on loggershim, and loggershim has a global that needs to be initialized with the NewLoggerShim function followed by Init
	if loggershim.LS == nil {
		// loggershim.LS hasn't been initialized elsewhere, so do it here
		_ = loggershim.NewLoggerShim(&configuration.Configuration{})
		loggershim.LS.Init(true, nil)
	}

	s := SubnetInfoProvider{
		dataStoreShim: datastoreshim.NewDataStoreShim(dataStoreSettings, configuration.DNCSettings{
			ThresholdForLoggingDataStoreOpLatencyMs: loggingConfig.ThresholdForLoggingDataStoreOpLatencyMs,
		}, map[string]string{subnetTokenMapTableName: vnetGUIDPropertyName}),
		cache: make(map[string]map[string]requestcontroller.SubnetInfo),
	}

	return &s
}

func (s *SubnetInfoProvider) GetSubnetInfo(ctx context.Context, delegatedSubnetID, subnetName string) (requestcontroller.SubnetInfo, error) {
	// Take a lock since this function needs to be thread safe
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// First, check the cache
	if subnetInfoMap, foundMap := s.cache[delegatedSubnetID]; foundMap {
		if subnetInfo, foundSubnetInfo := subnetInfoMap[subnetName]; foundSubnetInfo {
			return subnetInfo, nil
		}
	}

	// Query the DB if not found in cache
	subnetInfo := requestcontroller.SubnetInfo{
		DelegatedSubnetID: delegatedSubnetID,
		SubnetName:        subnetName,
	}
	value, status, err := s.dataStoreShim.QueryInPartition(subnetTokenMapTableName, delegatedSubnetID, subnetName)
	if err != nil {
		return subnetInfo, err
	}

	if status != http.StatusOK {
		return subnetInfo, fmt.Errorf("Unexpected status %d returned by cosmos", status)
	}

	switch v := value.(type) {
	case map[string]interface{}:
		if v[vnetGUIDPropertyName] != nil {
			subnetInfo.VnetID = v[vnetGUIDPropertyName].(string)
		}

		if v[primaryTokenPropertyName] != nil {
			subnetInfo.PrimaryToken = v[primaryTokenPropertyName].(string)
		}

		if v[secondaryTokenPropertyName] != nil {
			subnetInfo.SecondaryToken = v[secondaryTokenPropertyName].(string)
		}

	default:
		return subnetInfo, fmt.Errorf("Unexpected type %v returned by Cosmos", v)
	}

	// Add to the cache
	if _, found := s.cache[delegatedSubnetID]; !found {
		s.cache[delegatedSubnetID] = make(map[string]requestcontroller.SubnetInfo)
	}
	s.cache[delegatedSubnetID][subnetName] = subnetInfo

	return subnetInfo, nil
}
