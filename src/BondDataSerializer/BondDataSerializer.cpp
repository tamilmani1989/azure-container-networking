/**
 * @file BondDataSerializer.cpp
 * @brief Implementation for the data serialization using Bond.
 *
 * @copyright Copyright (c) Microsoft Corporation
 *
**/

#include <iostream>
#include "BondDataSerializer.h"
#include "BondSerializer.h"
#include "NcData_reflection.h"
#include "MacAddress_reflection.h"
#include "IPAddress_reflection.h"
#include "Subnet_reflection.h"
#include "VNetConfiguration_reflection.h"
#include "VNetAddress_reflection.h"
#include "GatewayInfo_reflection.h"
#include <bond/core/bond.h>
#include <bond/stream/output_buffer.h>
#include <bond/stream/stdio_output_stream.h>
#include <bond/protocol/simple_json_reader.h>
#include <bond/protocol/simple_json_writer.h>
#include <boost/algorithm/string.hpp>
#include <stdio.h>
#include <stdarg.h>

typedef unsigned char BYTE;
typedef unsigned char *PBYTE;

const size_t IPV4_SIZE_BYTES = 4;
const size_t IPV6_SIZE_BYTES = 16;

const int SUCCESS = 0;
const int FAILURE = -1;

struct IpAddress
{
    Sdn::Bond::Data::IPAddressType Type;
    PBYTE Data;
};

struct Subnet
{
    IpAddress IpAddressData;
    int32_t Prefix;
};

void LOG(const char *format, ...)
{
    va_list args;
    va_start(args, format);
    vprintf(format, args);
    va_end(args);
}

std::wstring StringToWString(const std::string &s)
{
    std::wstring ws(s.begin(), s.end());
    return ws;
}

char *convertWStringToCharPtr(const std::wstring wstr)
{
    const wchar_t *pWChar = wstr.c_str();
    size_t size = (wcslen(pWChar) + 1) * sizeof(wchar_t);
    char *pChar = new (std::nothrow) char[size];
    if (pChar != NULL)
    {
        std::wcstombs(pChar, pWChar, size);
    }
    return pChar;
}

int DeserializeIpV4ToBytes(
    const Sdn::Bond::Data::IPAddress &bondIpAddress,
    PBYTE &pIpAddressAsBytes)
{
    if (bondIpAddress.Type != Sdn::Bond::Data::IPAddressType::IPv4)
    {
        LOG("ERROR: DeserializeIpV4ToBytes: Invalid IP address type: %d", bondIpAddress.Type);
        return FAILURE;
    }

    pIpAddressAsBytes[0] = (BYTE)bondIpAddress.Data1;
    pIpAddressAsBytes[1] = (BYTE)bondIpAddress.Data2;
    pIpAddressAsBytes[2] = (BYTE)bondIpAddress.Data3;
    pIpAddressAsBytes[3] = (BYTE)bondIpAddress.Data4;

    return SUCCESS;
}

int DeserializeIpV6ToBytes(
    const Sdn::Bond::Data::IPAddress &bondIpAddress,
    PBYTE &pIpAddressAsBytes)
{
    if (bondIpAddress.Type != Sdn::Bond::Data::IPAddressType::IPv6)
    {
        LOG("ERROR: DeserializeIpV6ToBytes: Invalid IP address type: %d", bondIpAddress.Type);
        return FAILURE;
    }

    pIpAddressAsBytes[0] = (bondIpAddress.Data1 >> 8) & 0xFF;
    pIpAddressAsBytes[1] = (bondIpAddress.Data1) & 0xFF;

    pIpAddressAsBytes[2] = (bondIpAddress.Data2 >> 8) & 0xFF;
    pIpAddressAsBytes[3] = (bondIpAddress.Data2) & 0xFF;

    pIpAddressAsBytes[4] = (bondIpAddress.Data3 >> 8) & 0xFF;
    pIpAddressAsBytes[5] = (bondIpAddress.Data3) & 0xFF;

    pIpAddressAsBytes[6] = (bondIpAddress.Data4 >> 8) & 0xFF;
    pIpAddressAsBytes[7] = (bondIpAddress.Data4) & 0xFF;

    pIpAddressAsBytes[8] = (bondIpAddress.Data5 >> 8) & 0xFF;
    pIpAddressAsBytes[9] = (bondIpAddress.Data5) & 0xFF;

    pIpAddressAsBytes[10] = (bondIpAddress.Data6 >> 8) & 0xFF;
    pIpAddressAsBytes[11] = (bondIpAddress.Data6) & 0xFF;

    pIpAddressAsBytes[12] = (bondIpAddress.Data7 >> 8) & 0xFF;
    pIpAddressAsBytes[13] = (bondIpAddress.Data7) & 0xFF;

    pIpAddressAsBytes[14] = (bondIpAddress.Data8 >> 8) & 0xFF;
    pIpAddressAsBytes[15] = (bondIpAddress.Data8) & 0xFF;

    return SUCCESS;
}

int DeserializeBondIpAddress(
    const Sdn::Bond::Data::IPAddress &bondIpAddress,
    IpAddress &ipAddress)
{
    if ((bondIpAddress.Type != Sdn::Bond::Data::IPAddressType::IPv4) &&
        (bondIpAddress.Type != Sdn::Bond::Data::IPAddressType::IPv6))
    {
        LOG("ERROR: DeserializeBondIpAddress: Invalid IP address type: %d", bondIpAddress.Type);
        return FAILURE;
    }

    if (bondIpAddress.Type == Sdn::Bond::Data::IPAddressType::IPv4)
    {
        ipAddress.Type = Sdn::Bond::Data::IPAddressType::IPv4;
        ipAddress.Data = new (std::nothrow) BYTE[IPV4_SIZE_BYTES];
        if (ipAddress.Data == NULL)
        {
            LOG("ERROR: DeserializeBondIpAddress: Failed to allocate memory for IPv4 address");
            return FAILURE;
        }

        if (DeserializeIpV4ToBytes(bondIpAddress, ipAddress.Data) != SUCCESS)
        {
            LOG("ERROR: DeserializeBondIpAddress: Failed DeserializeIpV4ToBytes");
            return FAILURE;
        }
    }
    else if (bondIpAddress.Type == Sdn::Bond::Data::IPAddressType::IPv6)
    {
        ipAddress.Type = Sdn::Bond::Data::IPAddressType::IPv6;
        ipAddress.Data = new (std::nothrow) BYTE[IPV6_SIZE_BYTES];
        if (ipAddress.Data == NULL)
        {
            LOG("ERROR: DeserializeBondIpAddress: Failed to allocate memory for IPv6 address");
            return FAILURE;
        }

        if (DeserializeIpV6ToBytes(bondIpAddress, ipAddress.Data) != SUCCESS)
        {
            LOG("ERROR: DeserializeBondIpAddress: Failed DeserializeIpV6ToBytes");
            return FAILURE;
        }
    }

    return SUCCESS;
}

int DeserializeBondSubnet(
    const Sdn::Bond::Data::Subnet &bondSubnet,
    Subnet &subnet)
{
    if ((bondSubnet.Address.Type != Sdn::Bond::Data::IPAddressType::IPv4) &&
        (bondSubnet.Address.Type != Sdn::Bond::Data::IPAddressType::IPv6))
    {
        LOG("ERROR: DeserializeBondSubnet: Invalid Subnet address type: %d", bondSubnet.Address.Type);
        return FAILURE;
    }

    if (DeserializeBondIpAddress(bondSubnet.Address, subnet.IpAddressData) != SUCCESS)
    {
        LOG("ERROR: DeserializeBondSubnet: Failed DeserializeBondIpAddress");
        return FAILURE;
    }

    subnet.Prefix = bondSubnet.Prefix;
    return SUCCESS;
}

int SubnetToString(
    Subnet &subnet,
    std::wstring &subnetStr)
{
    wchar_t str[512];
    int charCount = 0;

    if ((charCount = swprintf(
             str,
             (sizeof(str) / sizeof(str[0])),
             L"%d.%d.%d.%d",
             subnet.IpAddressData.Data[0],
             subnet.IpAddressData.Data[1],
             subnet.IpAddressData.Data[2],
             subnet.IpAddressData.Data[3])) == -1)
    {
        LOG("ERROR: SubnetToString: Failed to convert Subnet IP");
        return FAILURE;
    }

    subnetStr.append(str, charCount);

    if ((charCount = swprintf(
             str,
             (sizeof(str) / sizeof(str[0])),
             L"/%d",
             subnet.Prefix)) == -1)
    {
        LOG("ERROR: SubnetToString: Failed to convert Subnet CIDR");
        return FAILURE;
    }

    subnetStr.append(str, charCount);
    return SUCCESS;
}

int IpAddressToString(
    IpAddress &ipAddress,
    std::wstring &ipAddressStr)
{
    wchar_t str[512];
    int charCount = 0;

    if ((charCount = swprintf(
             str,
             (sizeof(str) / sizeof(str[0])),
             L"%d.%d.%d.%d",
             ipAddress.Data[0],
             ipAddress.Data[1],
             ipAddress.Data[2],
             ipAddress.Data[3])) == -1)
    {
        LOG("ERROR: IpAddressToString: Failed to convert IP address");
        return FAILURE;
    }

    ipAddressStr.append(str, charCount);
    return SUCCESS;
}

int ConvertSubnetVectorToWString(
    std::vector<Sdn::Bond::Data::Subnet> &subnetVector,
    std::wstring &resultWstring)
{
    int count = 0;
    for (Sdn::Bond::Data::Subnet &bondSubnet : subnetVector)
    {
        Subnet subnet;
        if (DeserializeBondSubnet(bondSubnet, subnet) != SUCCESS)
        {
            LOG("ERROR: ConvertSubnetVectorToWString: Failed DeserializeBondSubnet");
            return FAILURE;
        }

        if (count++ > 0)
        {
            resultWstring += L",";
        }

        std::wstring subnetStr;
        SubnetToString(subnet, subnetStr);
        resultWstring += subnetStr;
    }

    return SUCCESS;
}

int ConvertIPAddressVectorToWString(
    std::vector<Sdn::Bond::Data::IPAddress> &ipAddressVector,
    std::wstring &resultWstring)
{
    int count = 0;
    for (Sdn::Bond::Data::IPAddress &bondIpAddress : ipAddressVector)
    {
        IpAddress ipAddress;
        if (SUCCESS != DeserializeBondIpAddress(bondIpAddress, ipAddress))
        {
            LOG("ERROR: ConvertIPAddressVectorToWString: Failed DeserializeBondIpAddress");
            return FAILURE;
        }

        if (count++ > 0)
        {
            resultWstring += L",";
        }

        std::wstring ipAddressStr;
        IpAddressToString(ipAddress, ipAddressStr);
        resultWstring += ipAddressStr;
    }

    return SUCCESS;
}

inline ResultStatus generateResultStatus(
    std::string errorDetails)
{
    ResultStatus resultStatus;
    if (errorDetails.empty())
    {
        resultStatus.ResultCode = 0;
        resultStatus.ErrorDetails = NULL;
        return resultStatus;
    }

    resultStatus.ResultCode = -1;
    resultStatus.ErrorDetails = errorDetails.c_str();
    return resultStatus;
}

ResultStatus DeserializeVnetConfiguration(
    void *serializedData,
    size_t serializedDataSize,
    VnetConfiguration *vnetConfiguration)
{
    Sdn::Bond::Data::VNetConfiguration vnetConfig;
    BYTE *updatedData = NULL;
    boost::shared_ptr<BYTE> paptr;
    bond::blob inputBlob;
    bond::InputBuffer inputBuffer;
    std::string successResultStatus;

    updatedData = new (std::nothrow) BYTE[serializedDataSize];
    if (updatedData == NULL)
    {
        LOG("ERROR: DeserializeVnetConfiguration: Failed to allocate memory for serialized data");
        return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to allocate memory for serialized data");
    }

    memcpy(updatedData, serializedData, serializedDataSize);

    paptr.reset(updatedData);
    //transfer ownership
    updatedData = NULL;
    inputBlob = bond::blob(paptr, static_cast<int>(serializedDataSize));
    inputBuffer = bond::InputBuffer(inputBlob);

    bool succeeded = BondSerializer::TryDeSerializeBuffer<Sdn::Bond::Data::VNetConfiguration>(inputBuffer, vnetConfig, BOND_FORMAT_JSON);
    if (!succeeded)
    {
        // try deserialize with compact binary format
        LOG("ERROR: DeserializeVnetConfiguration: Failed to deserialize VNetConfiguration with BOND_FORMAT_JSON");
        succeeded = BondSerializer::TryDeSerializeBuffer<Sdn::Bond::Data::VNetConfiguration>(inputBuffer, vnetConfig, BOND_FORMAT_COMPACT_BINARY);
        if (!succeeded)
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed with TryDeSerializeBuffer for BOND_FORMAT_COMPACT_BINARY");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to deserialize VNetConfiguration with BOND_FORMAT_COMPACT_BINARY");
        }
    }

    std::wstringstream vnetVersionStr;
    vnetVersionStr << vnetConfig.Version;

    char *vnetVersion = convertWStringToCharPtr(vnetVersionStr.str());
    vnetConfiguration->VnetVersion = vnetVersion;
    std::wstring cnets, vnets, dnsservers, subnetName, SubnetAuthToken, OwningService;

    if (vnetConfig.CNetSpace.hasvalue())
    {
        if (ConvertSubnetVectorToWString(vnetConfig.CNetSpace.value(), cnets) != SUCCESS)
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed to convert CNetSpace vector to string");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to convert CNetSpace vector to string");
        }
        vnetConfiguration->CNetSpace = convertWStringToCharPtr(cnets);
    }

    if (vnetConfig.VNetSpace.hasvalue())
    {
        if (ConvertSubnetVectorToWString(vnetConfig.VNetSpace.value(), vnets) != SUCCESS)
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed to convert VNetSpace vector to string");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to convert VNetSpace vector to string");
        }
        vnetConfiguration->VNetSpace = convertWStringToCharPtr(vnets);
    }

    if (vnetConfig.DnsServersInNetCfg.hasvalue())
    {
        if (ConvertIPAddressVectorToWString(vnetConfig.DnsServersInNetCfg.value(), dnsservers) != SUCCESS)
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed to convert VNetSpace vector to string");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to convert VNetSpace vector to string");
        }
        vnetConfiguration->DNSServers = convertWStringToCharPtr(dnsservers);
    }

    if (vnetConfig.VNetSubnets.hasvalue())
    {
        int subnetCount = 0;
        for (auto &bondSubnet : vnetConfig.VNetSubnets.value())
        {
            subnetCount++;
        }
        vnetConfiguration->SubnetCount = subnetCount;
        vnetConfiguration->Subnets = new (std::nothrow) SubnetConfiguration[vnetConfiguration->SubnetCount];
        if (vnetConfiguration->Subnets == NULL)
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed to allocate memory for subnet array");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to allocate memory for subnet array");
        }
    }

    int subnetCount = 0;
    for (auto &bondSubnet : vnetConfig.VNetSubnets.value())
    {
        SubnetConfiguration &subnetConfig = *(vnetConfiguration->Subnets + subnetCount);
        subnetConfig.SubnetName = convertWStringToCharPtr(bondSubnet.SubnetName);
        subnetConfig.OwningService = convertWStringToCharPtr(bondSubnet.ServiceName);
        subnetConfig.ExpectedToken = convertWStringToCharPtr(bondSubnet.Token);

        Subnet subnet;
        if (DeserializeBondSubnet(bondSubnet.SubnetAddress, subnet) == SUCCESS)
        {
            std::wstring addressPrefix;
            SubnetToString(subnet, addressPrefix);
            subnetConfig.AddressPrefix = convertWStringToCharPtr(addressPrefix);
        }
        else
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed to deserialize Bond Subnet");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to deserialize Bond Subnet");
        }

        IpAddress defaultGw;
        if (DeserializeBondIpAddress(bondSubnet.DefaultGateway, defaultGw) == SUCCESS)
        {
            std::wstring defaultGwStr;
            IpAddressToString(defaultGw, defaultGwStr);
            subnetConfig.DefaultGateway = convertWStringToCharPtr(defaultGwStr);
        }
        else
        {
            LOG("ERROR: DeserializeVnetConfiguration: Failed to deserialize Bond IP Address");
            return generateResultStatus("ERROR: DeserializeVnetConfiguration: Failed to deserialize Bond IP Address");
        }

        subnetConfig.TagCount = 0;
        if (bondSubnet.ServiceAddressTags.hasvalue())
        {
            int tagCount = 0;
            for (auto &bondTag : bondSubnet.ServiceAddressTags.value())
            {
                tagCount++;
            }
            subnetConfig.TagCount = tagCount;
            subnetConfig.Tags = new (std::nothrow) TagConfiguration[subnetConfig.TagCount];
            if (subnetConfig.Tags == NULL)
            {
                LOG("ERROR: DeserializeVnetConfiguration: Failed to allocate memory for tags array");
                return generateResultStatus("Failed to allocate memory for tags array");
            }

            tagCount = 0;
            for (auto &bondTag : bondSubnet.ServiceAddressTags.value())
            {
                TagConfiguration &tag = *(subnetConfig.Tags + tagCount);
                tag.Name = convertWStringToCharPtr(bondTag);
                tag.Type = convertWStringToCharPtr(L"System");
                tagCount++;
            }
        }
        subnetCount++;
    }

    free(updatedData);
    return generateResultStatus(successResultStatus);
}

ResultStatus SerializeNetworkContainerPublishData(
    NC_DATA *ncData,
    void **serializedNcData,
    size_t *sizeOfSerializedNcData)
{
    Sdn::Bond::Data::NcData SerializedNcData;
    std::map<std::wstring, std::wstring> NcDataMap;
    std::string successResultStatus;

    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"Subnet", StringToWString(std::string(ncData->SubnetPrefix))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"CustomerAddress", StringToWString(std::string(ncData->CustomerAddress))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"SecondaryCustomerAddresses", StringToWString(std::string(ncData->SecondaryCustomerAddresses))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"RoleInstanceId", StringToWString(std::string(ncData->RoleInstanceId))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"RoleInstanceName", StringToWString(std::string(ncData->RoleInstanceName))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"VnetId", StringToWString(std::string(ncData->VnetId))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"VlanId", StringToWString(std::string(ncData->VlanId))));
    NcDataMap.insert(std::pair<std::wstring, std::wstring>(
        L"NcSlbGreKey", StringToWString(std::string(ncData->NcSlbGreKey))));
    SerializedNcData.Version = ncData->NetworkContainerVersion;
    for (auto itr : NcDataMap)
    {
        Sdn::Bond::Data::NcProperty bondNcProperty;
        bondNcProperty.KeyName = itr.first;
        bondNcProperty.Value = itr.second;
        SerializedNcData.NcProperties.push_back(bondNcProperty);
    }

    // Pre-allocating buffer of 2*sizeof(Sdn::Bond::Data::NcData) for performance
    bond::OutputBuffer buffer(sizeof(Sdn::Bond::Data::NcData) << 1);
    bool succeeded = BondSerializer::TrySerializeBuffer<Sdn::Bond::Data::NcData>(buffer, SerializedNcData, BOND_FORMAT_JSON);
    if (!succeeded)
    {
        LOG("ERROR: SerializeNetworkContainerPublishData: Failed to deserialize with Bond for NcData");
        return generateResultStatus("ERROR: SerializeNetworkContainerPublishData: Failed to deserialize with Bond for NcData");
    }

    bond::blob bondData = buffer.GetBuffer();
    *sizeOfSerializedNcData = (size_t)bondData.length();
    BYTE *updatedData = new (std::nothrow) BYTE[*sizeOfSerializedNcData];
    if (updatedData == NULL)
    {
        LOG("ERROR: SerializeNetworkContainerPublishData: Cannot allocate memory to hold serialized NC data");
        return generateResultStatus("ERROR: SerializeNetworkContainerPublishData: Cannot allocate memory to hold serialized NC data");
    }

    memcpy(updatedData, bondData.data(), *sizeOfSerializedNcData);
    (*serializedNcData) = updatedData;
    updatedData = NULL;
    free(updatedData);

    return generateResultStatus(successResultStatus);
}
