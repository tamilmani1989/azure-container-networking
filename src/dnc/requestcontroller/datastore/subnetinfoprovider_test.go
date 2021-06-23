package datastore

import (
	"context"
	"dnc/configuration"
	"dnc/requestcontroller"
	"dnc/requestcontroller/appinsights"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/sirupsen/logrus"
)

func TestSubnetInfoProvider(t *testing.T) {
	metricsClient := appinsights.NewMetricsClient(appinsights.MetricsClientOptions{
		Logger: logrus.StandardLogger(),
		Config: configuration.TelemetrySettings{DisableTelemetry: true},
	})
	subnetInfoProvider, err := NewSubnetInfoProvider(SubnetInfoProviderOptions{
		DataStoreSettings: configuration.DataStoreSettings{
			TestPersistenceStoreInfo: configuration.TestPersistenceStoreInfo{
				Test: "true",
			},
		},
		LoggingConfig: requestcontroller.DefaultConfig().LoggingConfig,
		MetricsClient: metricsClient,
	})

	if err != nil {
		t.Fatalf("could not build subnetInfoProvider: %v", err)
	}

	delegatedSubnetID := "576d174c-57b8-4688-9d53-1b74caa642ca"
	subnetName := "subnet1"
	primaryToken := "cbabdad0-f88a-4791-b6d2-f18078953d23"
	secondaryToken := "92b714b6-0a70-4d65-830f-cbeab1826e01"

	// replace the datastoreClient with our mock
	subnetInfoProvider.datastoreClient = &mockDatastoreClient{
		rows: map[string]*storage.Entity{
			subnetName: {
				PartitionKey: delegatedSubnetID,
				RowKey:       subnetName,
				Properties: map[string]interface{}{
					primaryTokenPropertyName:   primaryToken,
					secondaryTokenPropertyName: secondaryToken,
					vnetGUIDPropertyName:       delegatedSubnetID,
				},
			},
		},
	}

	subnetInfo, err := subnetInfoProvider.GetSubnetInfo(context.Background(), delegatedSubnetID, subnetName)
	if err != nil {
		t.Fatalf("could not get subnet info: %v", err)
	}

	expectedSubnetInfo := requestcontroller.SubnetInfo{
		DelegatedSubnetID: delegatedSubnetID,
		SubnetName:        subnetName,
		PrimaryToken:      primaryToken,
		SecondaryToken:    secondaryToken,
		VnetID:            delegatedSubnetID,
	}

	if !reflect.DeepEqual(subnetInfo, expectedSubnetInfo) {
		t.Fatalf("subnet info did not match, expected: %+v, got: %+v", expectedSubnetInfo, subnetInfo)
	}

}
