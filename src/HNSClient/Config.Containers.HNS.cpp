// This file is generated code.
// DO NOT CHANGE THIS FILE. Change the MOF instead and rerun the MarshalGen.exe.
//
// Copyright (c) Microsoft Corporation. All rights reserved.

#include "Config.Containers.HNS.h"

namespace Config
{
	namespace Containers
	{
		namespace HNS
		{

			const Marshal::FieldData HNSNetwork_FieldData[]
			{
				{ L"ID", offsetof(::Config::Containers::HNS::HNSNetwork, ID), Marshal::FieldOption::OmitEmpty },
				{ L"Name", offsetof(::Config::Containers::HNS::HNSNetwork, Name), Marshal::FieldOption::OmitEmpty },
				{ L"HNSNetworkType", offsetof(::Config::Containers::HNS::HNSNetwork, HNSNetworkType), Marshal::FieldOption::OmitEmpty },
				{ L"Type", offsetof(::Config::Containers::HNS::HNSNetwork, Type), Marshal::FieldOption::OmitEmpty },
				{ L"NetworkAdapterName", offsetof(::Config::Containers::HNS::HNSNetwork, NetworkAdapterName), Marshal::FieldOption::OmitEmpty },
				{ L"SourceMac", offsetof(::Config::Containers::HNS::HNSNetwork, SourceMac), Marshal::FieldOption::OmitEmpty },
				{ L"Policies", offsetof(::Config::Containers::HNS::HNSNetwork, Policies), Marshal::FieldOption::OmitEmpty },
				{ L"MacPools", offsetof(::Config::Containers::HNS::HNSNetwork, MacPools), Marshal::FieldOption::OmitEmpty },
				{ L"Subnets", offsetof(::Config::Containers::HNS::HNSNetwork, Subnets), Marshal::FieldOption::OmitEmpty },
				{ L"DNSSuffix", offsetof(::Config::Containers::HNS::HNSNetwork, DNSSuffix), Marshal::FieldOption::OmitEmpty },
				{ L"DNSServerList", offsetof(::Config::Containers::HNS::HNSNetwork, DNSServerList), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSNetwork::FieldData{ HNSNetwork_FieldData };

			const Marshal::FieldData HNSEndpoint_FieldData[]
			{
				{ L"ID", offsetof(::Config::Containers::HNS::HNSEndpoint, ID), Marshal::FieldOption::OmitEmpty },
				{ L"Name", offsetof(::Config::Containers::HNS::HNSEndpoint, Name), Marshal::FieldOption::OmitEmpty },
				{ L"VirtualNetwork", offsetof(::Config::Containers::HNS::HNSEndpoint, VirtualNetwork), Marshal::FieldOption::OmitEmpty },
				{ L"VirtualNetworkName", offsetof(::Config::Containers::HNS::HNSEndpoint, VirtualNetworkName), Marshal::FieldOption::OmitEmpty },
				{ L"Policies", offsetof(::Config::Containers::HNS::HNSEndpoint, Policies), Marshal::FieldOption::OmitEmpty },
				{ L"MacAddress", offsetof(::Config::Containers::HNS::HNSEndpoint, MacAddress), Marshal::FieldOption::OmitEmpty },
				{ L"IPAddress", offsetof(::Config::Containers::HNS::HNSEndpoint, IPAddress), Marshal::FieldOption::OmitEmpty },
				{ L"DNSServerList", offsetof(::Config::Containers::HNS::HNSEndpoint, DNSServerList), Marshal::FieldOption::OmitEmpty },
				{ L"DNSSuffix", offsetof(::Config::Containers::HNS::HNSEndpoint, DNSSuffix), Marshal::FieldOption::OmitEmpty },
				{ L"GatewayAddress", offsetof(::Config::Containers::HNS::HNSEndpoint, GatewayAddress), Marshal::FieldOption::OmitEmpty },
				{ L"PrefixLength", offsetof(::Config::Containers::HNS::HNSEndpoint, PrefixLength), Marshal::FieldOption::OmitEmpty },
				{ L"EnableLowMetric", offsetof(::Config::Containers::HNS::HNSEndpoint, EnableLowMetric), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSEndpoint::FieldData{ HNSEndpoint_FieldData };

			const Marshal::FieldData HNSNetworkResponse_FieldData[]
			{
				{ L"Success", offsetof(::Config::Containers::HNS::HNSNetworkResponse, Success), Marshal::FieldOption::Required },
				{ L"Error", offsetof(::Config::Containers::HNS::HNSNetworkResponse, Error), Marshal::FieldOption::OmitEmpty },
				{ L"Output", offsetof(::Config::Containers::HNS::HNSNetworkResponse, Output), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSNetworkResponse::FieldData{ HNSNetworkResponse_FieldData };

			const Marshal::FieldData HNSSingleNetworkResponse_FieldData[]
			{
				{ L"Success", offsetof(::Config::Containers::HNS::HNSSingleNetworkResponse, Success), Marshal::FieldOption::Required },
				{ L"Error", offsetof(::Config::Containers::HNS::HNSSingleNetworkResponse, Error), Marshal::FieldOption::OmitEmpty },
				{ L"Output", offsetof(::Config::Containers::HNS::HNSSingleNetworkResponse, Output), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSSingleNetworkResponse::FieldData{ HNSSingleNetworkResponse_FieldData };

			const Marshal::FieldData HNSEndpointResponse_FieldData[]
			{
				{ L"Success", offsetof(::Config::Containers::HNS::HNSEndpointResponse, Success), Marshal::FieldOption::Required },
				{ L"Error", offsetof(::Config::Containers::HNS::HNSEndpointResponse, Error), Marshal::FieldOption::OmitEmpty },
				{ L"Output", offsetof(::Config::Containers::HNS::HNSEndpointResponse, Output), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSEndpointResponse::FieldData{ HNSEndpointResponse_FieldData };

			const Marshal::FieldData HNSEndpointsResponse_FieldData[]
			{
				{ L"Success", offsetof(::Config::Containers::HNS::HNSEndpointsResponse, Success), Marshal::FieldOption::Required },
				{ L"Error", offsetof(::Config::Containers::HNS::HNSEndpointsResponse, Error), Marshal::FieldOption::OmitEmpty },
				{ L"Output", offsetof(::Config::Containers::HNS::HNSEndpointsResponse, Output), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSEndpointsResponse::FieldData{ HNSEndpointsResponse_FieldData };

			const Marshal::FieldData HNSEndpointStatsResponse_FieldData[]
			{
				{ L"Success", offsetof(::Config::Containers::HNS::HNSEndpointStatsResponse, Success), Marshal::FieldOption::Required },
				{ L"Error", offsetof(::Config::Containers::HNS::HNSEndpointStatsResponse, Error), Marshal::FieldOption::OmitEmpty },
				{ L"Output", offsetof(::Config::Containers::HNS::HNSEndpointStatsResponse, Output), Marshal::FieldOption::OmitEmpty },
			};

			const Marshal::Details::ArrayView<Marshal::FieldData> HNSEndpointStatsResponse::FieldData{ HNSEndpointStatsResponse_FieldData };

		}
	}
}
