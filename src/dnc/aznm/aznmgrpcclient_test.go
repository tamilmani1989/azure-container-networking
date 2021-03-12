package aznm

import (
	"context"
	"dnc/configuration"
	"dnc/logger/loggermock"
	"dnc/microsoft/azure/aznm/contracts"
	"dnc/utils"
	"fmt"
	"github.com/google/uuid"
	"os"
	"testing"
)

var (
	config      *configuration.Configuration
)

const (
	maxLogFileSize  = 16 * 1024 * 1024 // Set 16MB as the max log file size
	maxLogFileCount = 8                // Set 8 as max log file count.

	// Note: Change this before running these tests
	certFile = "/mnt/c/Code/Networking-RNM/Networking/Aquarius/src/dnc/acnkv-CertDncPubsub-20201001.pem"
	azId     = "uscentraleuap-az01"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestUpdateAndDeleteContainer(t *testing.T) {
	client := NewAznmGrpcClient(certFile, loggermock.NewDummyLogger())

	aznmReq, err := CreateAznmRequestWithAzName(azId, config.AznmSettings.Port, "AZUREPUBLICCLOUD")
	if err != nil {
		t.Fatalf("Error creating aznm request: %s", err)
	}

	t.Logf("Calling aznm endpoint: %s, activityId: %s \n", aznmReq.Endpoint, aznmReq.ActivityId)

	logicalGuid := uuid.New().String()
	physicalGuid := uuid.New().String()
	nodeGuid := uuid.New().String()

	t.Logf("logicalGuid; %s\n", logicalGuid)
	t.Logf("physicalGuid; %s\n", physicalGuid)
	t.Logf("nodeGuid; %s\n", nodeGuid)

	err = client.UpdateContainer(context.Background(), aznmReq, logicalGuid, physicalGuid, nodeGuid)
	if err != nil {
		t.Fatalf("Error calling UpdateContainer: %s", err)
	}
	t.Logf("UpdateContainer succeeded\n")

	err = client.DeleteContainer(context.Background(), aznmReq, logicalGuid, physicalGuid)
	if err != nil {
		t.Fatalf("Error calling TestDeleteContainer: %s", err)
	}
	t.Logf("TestDeleteContainer succeeded\n")
}

func TestSdnContracts(t *testing.T) {
	// We use reservation ID to uniquely identify the NC on front end network
	expectedGuidLen := 16
	expectedMacLen := 6
	expectedIPLen := 4

	guidstr := "e781c8e2-8375-4436-8edc-9ca917c5b2e3"
	csharpGuidStr := "e2c881e7758336448edc9ca917c5b2e3"
	macStr := "00125A00B000"
	ipStr := "10.1.2.3"
	guid, _ := utils.ConvertGuidStrToCSharpGUID(guidstr)
	mac, _ := utils.ConvertMacAddressToSdnContractMacAddress(macStr)
	ip, _ := utils.ConvertIPaddressToSdnContractIPAddress(ipStr)

	t.Logf("sdn contracts guid(actual):%s, guid: %x, len: %d \n", guidstr, guid.Bytes, len(guid.Bytes))
	t.Logf("sdn contracts mac(actual):%s, mac: %x, len: %d \n", macStr, mac.Bytes, len(mac.Bytes))
	t.Logf("sdn contracts ip(actual):%s, ip: %x, len: %d \n", ipStr, ip.Bytes, len(ip.Bytes))

	guidlength := len(guid.Bytes)
	if guidlength != expectedGuidLen {
		t.Fatalf("Invalid length %d for guid. Expected is %d", guidlength, expectedGuidLen)
	}

	maclength := len(mac.Bytes)
	if maclength != expectedMacLen {
		t.Fatalf("Invalid length %d for mac. Expected is %d", maclength, expectedMacLen)
	}

	iplength := len(ip.Bytes)
	if iplength != expectedIPLen {
		t.Fatalf("Invalid length %d for ip. Expected is %d", iplength, expectedIPLen)
	}

	convertedGuidStr := fmt.Sprintf("%x", guid.Bytes)
	if convertedGuidStr != csharpGuidStr {
		t.Fatalf("Invalid Guid. Output %s . Expected is %s", convertedGuidStr, csharpGuidStr)
	}
}

func TestGoUuidToSdnUuid(t *testing.T) {
	guidstr := "dfd0ba66-1a47-4bac-8722-9c657082a189"
	var expectedHighUint uint64 = 5452762144838826598
	var expectedlowUint uint64 = 9917351273699222151

	var aznmUuid *contracts.Uuid
	var err error
	if aznmUuid, err = convertGuidStrToProtobufUuid(guidstr); err != nil {
		t.Fatalf("TestGoUuidToSdnUuid failed with error: %s", err)
	}

	if aznmUuid.HighBits != expectedHighUint {
		t.Fatalf("TestGoUuidToSdnUuid highbits don't match. Expected: %d, Actual:%d",
			expectedHighUint, aznmUuid.HighBits)
	}

	if aznmUuid.LowBits != expectedlowUint {
		t.Fatalf("TestGoUuidToSdnUuid lowbits don't match. Expected: %d, Actual:%d",
			expectedlowUint, aznmUuid.LowBits)
	}
}

func setup() {

	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config_test.json"
	}

	fmt.Printf("configpath: %s\n", configpath)
	config = configuration.Load(configpath)
}