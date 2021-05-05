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
	datastoreClient datastoreClient
	mutex           sync.Mutex
	cache           map[string]map[string]requestcontroller.SubnetInfo // delegatedSubnetID -> {subnetName -> SubnetInfo, subnetName -> SubnetInfo}
}

/*
Table:            SubnetTokenMap
PartitionKey:     <DelegatedSubnetID>
RowKey:           <SubnetName>
PrimaryTokenId:   <Token>
SecondaryTokenId: <Token> (optional)
VnetResourceGuid: <VnetID>
*/

func NewSubnetInfoProvider(dataStoreSettings configuration.DataStoreSettings, loggingConfig requestcontroller.LoggingConfig) (*SubnetInfoProvider, error) {
	//datastoreshim has a dependency on loggershim, and loggershim has a global that needs to be initialized with the NewLoggerShim function followed by Init
	if loggershim.LS == nil {
		// loggershim.LS hasn't been initialized elsewhere, so do it here
		_ = loggershim.NewLoggerShim(&configuration.Configuration{})
		loggershim.LS.Init(true, nil)
	}

	ds, err := datastoreshim.NewDataStoreShim(dataStoreSettings, loggingConfig.ThresholdForLoggingDataStoreOpLatencyMs, map[string]string{subnetTokenMapTableName: vnetGUIDPropertyName})
	if err != nil {
		return nil, err
	}

	s := SubnetInfoProvider{
		datastoreClient: ds,
		cache:           make(map[string]map[string]requestcontroller.SubnetInfo),
	}

	return &s, nil
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
	entity, status, err := s.datastoreClient.QueryInPartition(subnetTokenMapTableName, delegatedSubnetID, subnetName)
	if err != nil {
		return subnetInfo, err
	}

	if status != http.StatusOK {
		return subnetInfo, fmt.Errorf("Unexpected status %d returned by cosmos", status)
	}

	if entity.Properties[vnetGUIDPropertyName] != nil {
		subnetInfo.VnetID = entity.Properties[vnetGUIDPropertyName].(string)
	}

	if entity.Properties[primaryTokenPropertyName] != nil {
		subnetInfo.PrimaryToken = entity.Properties[primaryTokenPropertyName].(string)
	}

	if entity.Properties[secondaryTokenPropertyName] != nil {
		subnetInfo.SecondaryToken = entity.Properties[secondaryTokenPropertyName].(string)
	}

	// Add to the cache
	if _, found := s.cache[delegatedSubnetID]; !found {
		s.cache[delegatedSubnetID] = make(map[string]requestcontroller.SubnetInfo)
	}
	s.cache[delegatedSubnetID][subnetName] = subnetInfo

	return subnetInfo, nil
}
