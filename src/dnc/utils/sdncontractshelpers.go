package utils

import sdn "dnc/microsoft/azure/sdncontracts"

// ConvertGUIDStrToSdnContractGUID - converts the string guid to sdncontract guid
func ConvertGUIDStrToSdnContractGUID(id string) (*sdn.Guid, error) {
	x := &sdn.Guid{}
	x.Bytes = []byte(id)
	return x, nil
}

// ConvertMacAddressToSdnContractMacAddress - converts the string macaddress to sdncontract macaddress
func ConvertMacAddressToSdnContractMacAddress(macAddress string) (*sdn.MacAddress, error) {
	x := &sdn.MacAddress{}
	x.Bytes = []byte(macAddress)
	return x, nil
}

// ConvertIPaddressToSdnContractIPAddress - converts the string ipaddress to sdncontract ipaddress
func ConvertIPaddressToSdnContractIPAddress(ipadress string) (*sdn.IPAddress, error) {
	x := &sdn.IPAddress{}
	x.Bytes = []byte(ipadress)
	return x, nil
}

/*func ConvertGuidStrToSdnContractGuid(id string) ([]*sdn.Guid, error) {
	x := &sdn.Guid{}
	x.Bytes = []byte(id)
	i := make([]*sdn.Guid, 1)
	i[0] = x
	return i, nil
}*/
