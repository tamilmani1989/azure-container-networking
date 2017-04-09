// This file is generated code.
// DO NOT CHANGE THIS FILE. Change the MOF instead and rerun the MarshalGen.exe.
//
// Copyright (c) Microsoft Corporation. All rights reserved.

#pragma once
#include <guiddef.h>
#include "marshal\inc\MarshalJson.h"
#include "marshal\inc\MarshalBase.h"
#include "Config.Containers.h"

namespace Config
{
	namespace Containers
	{
		namespace HNS
		{

			struct HNSNetwork;
			struct HNSEndpoint;
			struct HNSNetworkResponse;
			struct HNSSingleNetworkResponse;
			struct HNSEndpointResponse;
			struct HNSEndpointStatsResponse;

			struct HNSNetwork
			{
				std::wstring ID = std::wstring{};
				std::wstring Name = std::wstring{};
				std::wstring HNSNetworkType = std::wstring{};
				::Config::Containers::NetworkMode Type = ::Config::Containers::NetworkMode{};
				std::wstring NetworkAdapterName = std::wstring{};
				std::wstring SourceMac = std::wstring{};
				std::vector<::Config::Containers::Policy> Policies = std::vector<::Config::Containers::Policy>{};
				std::vector<::Config::Containers::MacPool> MacPools = std::vector<::Config::Containers::MacPool>{};
				std::vector<::Config::Containers::Subnet> Subnets = std::vector<::Config::Containers::Subnet>{};
				std::wstring DNSSuffix = std::wstring{};
				std::wstring DNSServerList = std::wstring{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					std::wstring,
					std::wstring,
					std::wstring,
					::Config::Containers::NetworkMode,
					std::wstring,
					std::wstring,
					std::vector<::Config::Containers::Policy>,
					std::vector<::Config::Containers::MacPool>,
					std::vector<::Config::Containers::Subnet>,
					std::wstring,
					std::wstring
				>;
			};

			std::true_type CanMarshalObject(const HNSNetwork&);

			struct HNSEndpoint
			{
				GUID ID = GUID{};
				std::wstring Name = std::wstring{};
				GUID VirtualNetwork = GUID{};
				std::wstring VirtualNetworkName = std::wstring{};
				std::vector<web::json::value> Policies = std::vector<web::json::value>{};
				std::wstring MacAddress = std::wstring{};
				std::wstring IPAddress = std::wstring{};
				std::wstring DNSServerList = std::wstring{};
				std::wstring DNSSuffix = std::wstring{};
				std::wstring GatewayAddress = std::wstring{};
				UINT8 PrefixLength = UINT8{};
				bool EnableLowMetric = bool{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					GUID,
					std::wstring,
					GUID,
					std::wstring,
					std::vector<web::json::value>,
					std::wstring,
					std::wstring,
					std::wstring,
					std::wstring,
					std::wstring,
					UINT8,
					bool
				>;
			};

			std::true_type CanMarshalObject(const HNSEndpoint&);

			struct HNSNetworkResponse
			{
				bool Success = bool{};
				std::wstring Error = std::wstring{};
				std::vector<HNSNetwork> Output = std::vector<::Config::Containers::HNS::HNSNetwork>{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					bool,
					std::wstring,
					std::vector<::Config::Containers::HNS::HNSNetwork>
				>;
			};

			std::true_type CanMarshalObject(const HNSNetworkResponse&);

			struct HNSSingleNetworkResponse
			{
				bool Success = bool{};
				std::wstring Error = std::wstring{};
				HNSNetwork Output = ::Config::Containers::HNS::HNSNetwork{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					bool,
					std::wstring,
					::Config::Containers::HNS::HNSNetwork
				>;
			};

			std::true_type CanMarshalObject(const HNSSingleNetworkResponse&);

			struct HNSEndpointResponse
			{
				bool Success = bool{};
				std::wstring Error = std::wstring{};
				HNSEndpoint Output = ::Config::Containers::HNS::HNSEndpoint{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					bool,
					std::wstring,
					::Config::Containers::HNS::HNSEndpoint
				>;
			};

			std::true_type CanMarshalObject(const HNSEndpointResponse&);

			struct HNSEndpointsResponse
			{
				bool Success = bool{};
				std::wstring Error = std::wstring{};
				std::vector<HNSEndpoint> Output = std::vector<::Config::Containers::HNS::HNSEndpoint>{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					bool,
					std::wstring,
					std::vector<::Config::Containers::HNS::HNSEndpoint>
				>;
			};

			std::true_type CanMarshalObject(const HNSEndpointsResponse&);

			struct HNSEndpointStatsResponse
			{
				bool Success = bool{};
				std::wstring Error = std::wstring{};
				web::json::value Output = web::json::value{};

				static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
				using FieldTypes = Marshal::TypeList<
					bool,
					std::wstring,
					web::json::value
				>;
			};

			std::true_type CanMarshalObject(const HNSEndpointStatsResponse&);

		}
	}
}
