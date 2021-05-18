package networkupdatehandler

import (
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggermock"
	"dnc/logger/loggershim"
	"dnc/networkcontainer"
	"dnc/nmagentcontracts"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestUpdateNetworkConfiguration(t *testing.T) {
	// todo: really need to improve wiring up dependencies

	cfg := configuration.Configuration{
		DataStoreSettings: configuration.DataStoreSettings{
			TestPersistenceStoreInfo: configuration.TestPersistenceStoreInfo{Test: "yes"},
			MaxBatchSize:             1000,
		},
		FeatureFlagSettings: configuration.FeatureFlagSettings{UsePrepopulatedIPAllocations: true},
	}

	loggershim.LS = loggermock.NewNoOpLogger()
	initErr := datamodel.Init(cfg.CNSSettings, cfg.DataStoreSettings, cfg.DNCSettings, cfg.HostAgentSettings, 0, 0, 0, 0, nil, nil, "")
	require.NoError(t, initErr)
	alloc := networkcontainer.IPAllocator{Allocations: datamodel.AllocRepo, Logger: loggermock.NewNoOpLogger(), MaxBatchSize: 1000}
	Init(&cfg, nil, nil, nil, &alloc, loggermock.NewDummyLogger())

	/*
	  tests the following scenario:
	  when using join subnet v2, the subnet record will exist before the network is joined,
	  so when executing network refresh, it will be processed as an update.
	  given the config above, we expect subnet to be processed to completion and ip allocation records to exist.
	*/

	vnetID := "vnet1"
	subnetName := "subnet1"
	subnetPrefix := "10.1.0.0/16"
	defaultGW := "10.1.0.1"

	existingSubnetCfg := datamodel.SubnetConfiguration{
		NetworkID:    vnetID,
		SubnetName:   subnetName,
		UpdateStatus: datamodel.SubnetUpdatePending,
		Joined: true,
	}

	err := datamodel.PutSubnetConfiguration(&existingSubnetCfg)
	require.Nil(t, err)

	network := datamodel.NetworkConfiguration{
		NetworkID: vnetID,
		Version:   1,
		Subnets:   map[string]string{subnetName: datamodel.GetSubnetKey(vnetID, subnetName)},
		Status:    datamodel.Pending,
	}

	nmagentResponse := nmagentcontracts.VirtualNetworkResponse{
		Subnets: []nmagentcontracts.Subnet{
			{
				SubnetName:     subnetName,
				AddressPrefix:  subnetPrefix,
				DefaultGateway: defaultGW,
			},
		},
	}

	shouldNotRetry, err := UpdateNetworkConfiguration(&network, &nmagentResponse)
	assert.Nil(t, err)
	assert.True(t, shouldNotRetry)

	newSubnetCfg, err := datamodel.TryQuerySubnetConfiguration(vnetID, subnetName)
	require.Nil(t, err)

	assert.Equal(t, subnetPrefix, newSubnetCfg.IPSubnet)
	assert.Equal(t, defaultGW, newSubnetCfg.Gateway)
	assert.True(t, newSubnetCfg.IsDelegatedSubnet)
	assert.Equal(t, datamodel.SubnetUpdateCompleted, newSubnetCfg.UpdateStatus)

	allocs, err := datamodel.AllocRepo.TryQueryAllIPAllocations(vnetID, subnetName)
	require.Nil(t, err)
	assert.Len(t, allocs, 65536)
}

func TestUnregisteredSubnetPrepopulation(t *testing.T) {
	cfg := configuration.Configuration{
		DataStoreSettings: configuration.DataStoreSettings{
			TestPersistenceStoreInfo: configuration.TestPersistenceStoreInfo{Test: "yes"},
			MaxBatchSize:             1000,
		},
		FeatureFlagSettings: configuration.FeatureFlagSettings{UsePrepopulatedIPAllocations: true},
	}

	loggershim.LS = loggermock.NewNoOpLogger()
	initErr := datamodel.Init(cfg.CNSSettings, cfg.DataStoreSettings, cfg.DNCSettings, cfg.HostAgentSettings, 0, 0, 0, 0, nil, nil, "")
	require.NoError(t, initErr)
	alloc := networkcontainer.IPAllocator{Allocations: datamodel.AllocRepo, Logger: loggermock.NewNoOpLogger(), MaxBatchSize: 1000}
	Init(&cfg, nil, nil, nil, &alloc, loggermock.NewDummyLogger())

	// Simulate a case where NMA response contains subnets which are not delegated so those are not registerd with
	// DNC with register subnet API.
	vnetID := "vnet1"
	unregisteredSubnet := "unregisteredSubnet"
	unregisteredsubnetPrefix := "10.2.0.0/16"
	unregisteredSubnetDefaultGW := "10.2.0.1"
	network := datamodel.NetworkConfiguration{
		NetworkID: vnetID,
		Version:   1,
		Subnets:   map[string]string{},
		Status:    datamodel.Pending,
	}

	nmagentResponse := nmagentcontracts.VirtualNetworkResponse{
		Subnets: []nmagentcontracts.Subnet{
			{
				SubnetName:     unregisteredSubnet,
				AddressPrefix:  unregisteredsubnetPrefix,
				DefaultGateway: unregisteredSubnetDefaultGW,
			},
		},
	}

	shouldNotRetry, err := UpdateNetworkConfiguration(&network, &nmagentResponse)
	assert.Nil(t, err)
	assert.True(t, shouldNotRetry)

	// There should be no prepopulated IPs from unregistered subnet
	allocs, err := datamodel.AllocRepo.TryQueryAllIPAllocations(vnetID, unregisteredSubnet)
	require.Nil(t, err)
	assert.Len(t, allocs, 0)
}

// todo: test more scenarios
