/**
 * @file BondDataSerializer.h
 * @brief Defines the interface for the data serialization using Bond.
 *
 * @copyright Copyright (c) Microsoft Corporation
 *
**/

#pragma once

using namespace std;

#include <cstdint>
#ifdef __cplusplus
extern "C" {
#endif

struct NC_DATA {
    uint64_t NetworkContainerVersion;
    char* SubnetPrefix;
    char* CustomerAddress;
    char* RoleInstanceId;
    char* RoleInstanceName;
    char* VnetId;
    char* VlanId;
};

struct TagConfiguration {
	char* Name;
	char* Type;
};

struct SubnetConfiguration {
	char* SubnetName;
	char* AddressPrefix;
	TagConfiguration* Tags;
    uint32_t TagCount;
	char* DefaultGateway;
	char* ExpectedToken;
    char* OwningService;
};

struct VnetConfiguration {
	char* VnetVersion;
	char* VNetSpace;
	char* CNetSpace;
	char* DNSServers;
	SubnetConfiguration* Subnets;
    uint32_t SubnetCount;
};

struct ResultStatus {
    int32_t ResultCode;
    const char* ErrorDetails;
};

ResultStatus DeserializeVnetConfiguration(void *serializedData, size_t serializedDataSize, VnetConfiguration *vnetConfig);
ResultStatus SerializeNetworkContainerPublishData(NC_DATA* ncData, void** serializedNcData, size_t* sizeOfSerializedNcData);

#ifdef __cplusplus
}
#endif
