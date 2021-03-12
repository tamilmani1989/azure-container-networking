package aznm

import (
	"dnc/microsoft/azure/aznm/contracts"
	"dnc/utils"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	guidLen = 16
)

// TODO: Activity ID needs to be in format XX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX-XXXXXXXXXXXXXXXX-XX
// TODO: Get code/api to generate this. This is an requirement from telemetry API aznm is using
func CreateAznmRequestWithAzName(azIdOrName string, aznmPort int, azureEnvName string) (AznmRequest, error) {

	suffix := getEndpointSuffix(azureEnvName)
	if suffix == "" {
		return AznmRequest{}, fmt.Errorf("aznm url suffix not found for environment: %s", azureEnvName)
	}

	url := strings.Join([]string{azIdOrName, suffix}, ".")
	endpoint := strings.Join([]string{url, fmt.Sprintf("%d", aznmPort)}, ":")

	return AznmRequest{
		Endpoint:   endpoint,
		ActivityId: "11-11111111111111111111111111111111-1111111111111111-11",
	}, nil
}

// TODO: Activity ID needs to be in format XX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX-XXXXXXXXXXXXXXXX-XX
// TODO: Get code/api to generate this. This is an requirement from telemetry API aznm is using
func CreateAznmRequestWithAzIPAddress(azIPAddress string, aznmPort int) (AznmRequest, error) {

	endpoint := strings.Join([]string{azIPAddress, fmt.Sprintf("%d", aznmPort)}, ":")
	return AznmRequest{
		Endpoint:   endpoint,
		ActivityId: "11-11111111111111111111111111111111-1111111111111111-11",
	}, nil
}

func getEndpointSuffix(azureEnvName string) string {

	suffix := ""
	switch strings.ToUpper(azureEnvName) {
	case "AZUREPUBLICCLOUD":
		suffix = "aznm.core.windows.net"
	case "AZUREGERMANCLOUD":
		suffix = "aznm.core.cloudapi.de"
	case "AZUREUSGOVERNMENTCLOUD":
		suffix = "aznm.core.usgovcloudapi.net"
	case "AZURECHINACLOUD":
		suffix = "aznm.core.chinacloudapi.cn"

		// TODO: Find out Azure environment name for USNAT
		//case "USNAT" :
		//suffix = "aznm.core.eaglex.ic.gov"

	}

	return suffix
}

// Aznm is written in .Net. .NET guid's internal byte structure stores the first 3 section of guid
// in little endian format and rest is big endian. This function first converts GO uuid to a byte format that .NET uses
// and then convert to uint64. When aznm converts these uint back to guid byte array in .NET and then to a guid eventually,
// it will and then be in same format that .NET expects.
// TODO: Aznm team is working on providing a GO client to do these conversions. We can deprecate this method once
// TODO: we get the client.
func convertGuidStrToProtobufUuid(guid string) (*contracts.Uuid, error) {
	var high, low uint64

	guidDehyphened := utils.DehyphenString(guid)
	data, err := hex.DecodeString(guidDehyphened)
	if err != nil {
		return nil, err
	}

	dataLen := len(data)
	if dataLen != guidLen {
		return nil, fmt.Errorf("Length of uuid/guid is not correct. Expected:%d, Actual:%d", guidLen, dataLen)
	}

	part1 := utils.ReverseByteArray(data[0:4])
	part2 := utils.ReverseByteArray(data[4:6])
	part3 := utils.ReverseByteArray(data[6:8])
	lowBytes := data[8:len(data)]

	var highBytes []byte
	for _, b := range part1 {
		highBytes = append(highBytes, b)
	}

	for _, b := range part2 {
		highBytes = append(highBytes, b)
	}

	for _, b := range part3 {
		highBytes = append(highBytes, b)
	}

	high = binary.LittleEndian.Uint64(highBytes)
	low = binary.LittleEndian.Uint64(lowBytes)

	uuid := &contracts.Uuid{
		HighBits: high,
		LowBits:  low,
	}

	return uuid, nil
}