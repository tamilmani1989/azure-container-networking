package utils

import (
	sdn "dnc/microsoft/azure/sdncontracts"
	"encoding/hex"
	"net"
)

type byteSlice []byte

// ConvertGUIDStrToSdnContractGUID - converts the string guid to sdncontract guid
func ConvertGuidStrToCSharpGUID(id string) (*sdn.Guid, error) {
	deGuid :=  DehyphenString(id)
	data, err := hex.DecodeString(deGuid)
	if err != nil {
		return nil, err
	}

	highBytes1 := ReverseByteArray(data[0:4])
	highBytes2 := ReverseByteArray(data[4:6])
	highBytes3 := ReverseByteArray(data[6:8])
	lowBytes := data[8:len(data)]

	// Order matter in this array and should be highBytes1, highBytes2, highBytes3, lowBytes,
	guidSlices := []byteSlice {
		highBytes1, highBytes2, highBytes3, lowBytes,
	}

	var guidBytes byteSlice
	for _, b := range guidSlices {
		guidBytes = append(guidBytes, b...)
	}

	x := &sdn.Guid{}
	x.Bytes = guidBytes
	return x, nil
}

// ConvertMacAddressToSdnContractMacAddress - converts the string macaddress to sdncontract macaddress
func ConvertMacAddressToSdnContractMacAddress(macAddress string) (*sdn.MacAddress, error) {
	deMac :=  DehyphenString(macAddress)
	data, err := hex.DecodeString(deMac)
	if err != nil {
		return nil, err
	}

	x := &sdn.MacAddress{}
	x.Bytes = data
	return x, nil
}

// ConvertIPaddressToSdnContractIPAddress - converts the string ipaddress to sdncontract ipaddress
func ConvertIPaddressToSdnContractIPAddress(ipadress string) (*sdn.IPAddress, error) {

	ip := net.ParseIP(ipadress).To4()

	x := &sdn.IPAddress{}
	x.Bytes = ip
	return x, nil
}