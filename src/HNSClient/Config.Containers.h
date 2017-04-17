// This file is generated code.
// DO NOT CHANGE THIS FILE. Change the MOF instead and rerun the MarshalGen.exe.
//
// Copyright (c) Microsoft Corporation. All rights reserved.

#pragma once
#include <guiddef.h>
#include "MacAddressUtils.h"
#include "marshal\inc\MarshalBase.h"
#include "marshal\inc\MarshalJson.h"

namespace Config
{
	namespace Containers
	{

		enum class SystemType;
		enum class ContainerType;
		enum class NatPortProtocol;
		enum class NetworkMode;
		enum class PolicyType;
		struct Layer;
		struct MappedDirectory;
		struct DeviceBase;
		struct UtilityVmNetworkSettings;
		struct UtilityVmSettings;
		struct SettingsBase;
		struct ContainerSettingsBase;
		struct ContainerSettings;
		struct NatPortBinding;
		struct NatSettings;
		struct NetworkConnection;
		struct NetworkSettings;
		struct NetworkDevice;
		struct Subnet;
		struct MacPool;
		struct Policy;
		struct NatPolicyData;
		struct QosPolicyData;
		struct VlanPolicyData;
		struct VsidPolicyData;
		struct NetworkEndpoint;

		enum class SystemType
		{
			Container = 0,
			VirtualMachine = 1,
		};

		const Marshal::EnumData& GetEnumData(SystemType);

		enum class ContainerType
		{
			VmHosted = 0,
		};

		const Marshal::EnumData& GetEnumData(ContainerType);

		enum class NatPortProtocol
		{
			TCP = 0,
			UDP = 1,
		};

		const Marshal::EnumData& GetEnumData(NatPortProtocol);

		enum class NetworkMode
		{
			NAT = 0,
			Transparent = 1,
			L2Bridge = 2,
			L2Tunnel = 3,
		};

		const Marshal::EnumData& GetEnumData(NetworkMode);

		enum class PolicyType
		{
			NAT = 0,
			QOS = 1,
			VLAN = 2,
			VSID = 3,
		};

		const Marshal::EnumData& GetEnumData(PolicyType);

		struct Layer
		{
			GUID Id = GUID{};
			std::wstring Path = std::wstring{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				GUID,
				std::wstring
			>;
		};

		std::true_type CanMarshalObject(const Layer&);

		struct MappedDirectory
		{
			std::wstring HostPath = std::wstring{};
			std::wstring ContainerPath = std::wstring{};
			bool ReadOnly = bool{};
			vmstd::optional<UINT64> IOPSMaximum = vmstd::optional<UINT64>{};
			vmstd::optional<UINT64> BandwidthMaximum = vmstd::optional<UINT64>{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				std::wstring,
				bool,
				vmstd::optional<UINT64>,
				vmstd::optional<UINT64>
			>;
		};

		std::true_type CanMarshalObject(const MappedDirectory&);

		struct DeviceBase
		{
			std::wstring DeviceType = std::wstring{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring
			>;
		};

		std::true_type CanMarshalObject(const DeviceBase&);

		struct UtilityVmNetworkSettings
		{
			std::wstring SwitchName = std::wstring{};
			std::wstring PortName = std::wstring{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				std::wstring
			>;
		};

		std::true_type CanMarshalObject(const UtilityVmNetworkSettings&);

		struct UtilityVmSettings
		{
			std::wstring ImagePath = std::wstring{};
			GUID RuntimeId = GUID{};
			bool SkipTemplate = bool{};
			bool EnableConsole = bool{};
			UtilityVmNetworkSettings NetworkSettings = ::Config::Containers::UtilityVmNetworkSettings{};
			std::wstring Com1PipeName = std::wstring{};
			std::wstring Com2PipeName = std::wstring{};
			bool EnableUefiDebugger = bool{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				GUID,
				bool,
				bool,
				::Config::Containers::UtilityVmNetworkSettings,
				std::wstring,
				std::wstring,
				bool
			>;
		};

		std::true_type CanMarshalObject(const UtilityVmSettings&);

		struct SettingsBase
		{
			SystemType SystemType = ::Config::Containers::SystemType{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::SystemType
			>;
		};

		std::true_type CanMarshalObject(const SettingsBase&);

		struct ContainerSettingsBase : public SettingsBase
		{
			std::wstring Name = std::wstring{};
			bool HvPartition = bool{};
			vmstd::optional<ContainerType> ContainerType = vmstd::optional<::Config::Containers::ContainerType>{};
			bool Servicing = bool{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::SystemType,
				std::wstring,
				bool,
				vmstd::optional<::Config::Containers::ContainerType>,
				bool
			>;
		};

		std::true_type CanMarshalObject(const ContainerSettingsBase&);

		struct ContainerSettings : public ContainerSettingsBase
		{
			std::wstring Owner = std::wstring{};
			bool IsDummy = bool{};
			bool TerminateOnLastHandleClosed = bool{};
			UtilityVmSettings HvRuntime = ::Config::Containers::UtilityVmSettings{};
			std::wstring HostName = std::wstring{};
			std::wstring Credentials = std::wstring{};
			vmstd::optional<INT64> MemoryMaximumInMB = vmstd::optional<INT64>{};
			vmstd::optional<UINT32> ProcessorCount = vmstd::optional<UINT32>{};
			vmstd::optional<INT64> ProcessorMaximum = vmstd::optional<INT64>{};
			vmstd::optional<INT64> ProcessorWeight = vmstd::optional<INT64>{};
			bool IgnoreFlushesDuringBoot = bool{};
			std::wstring SandboxPath = std::wstring{};
			std::wstring VolumePath = std::wstring{};
			std::wstring LayerFolderPath = std::wstring{};
			std::vector<Layer> Layers = std::vector<::Config::Containers::Layer>{};
			std::vector<MappedDirectory> MappedDirectories = std::vector<::Config::Containers::MappedDirectory>{};
			vmstd::optional<UINT64> StorageIOPSMaximum = vmstd::optional<UINT64>{};
			vmstd::optional<UINT64> StorageBandwidthMaximum = vmstd::optional<UINT64>{};
			vmstd::optional<UINT64> StorageSandboxSize = vmstd::optional<UINT64>{};
			std::vector<web::json::value> Devices = std::vector<web::json::value>{};
			std::vector<NetworkEndpoint> NetworkEndpoints = std::vector<::Config::Containers::NetworkEndpoint>{};
			std::vector<GUID> EndpointList = std::vector<GUID>{};
			bool EnableServiceDiscovery = bool{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::SystemType,
				std::wstring,
				bool,
				vmstd::optional<::Config::Containers::ContainerType>,
				bool,
				std::wstring,
				bool,
				bool,
				::Config::Containers::UtilityVmSettings,
				std::wstring,
				std::wstring,
				vmstd::optional<INT64>,
				vmstd::optional<UINT32>,
				vmstd::optional<INT64>,
				vmstd::optional<INT64>,
				bool,
				std::wstring,
				std::wstring,
				std::wstring,
				std::vector<::Config::Containers::Layer>,
				std::vector<::Config::Containers::MappedDirectory>,
				vmstd::optional<UINT64>,
				vmstd::optional<UINT64>,
				vmstd::optional<UINT64>,
				std::vector<web::json::value>,
				std::vector<::Config::Containers::NetworkEndpoint>,
				std::vector<GUID>,
				bool
			>;
		};

		std::true_type CanMarshalObject(const ContainerSettings&);

		struct NatPortBinding
		{
			NatPortProtocol Protocol = ::Config::Containers::NatPortProtocol{};
			UINT16 InternalPort = UINT16{};
			UINT16 ExternalPort = UINT16{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::NatPortProtocol,
				UINT16,
				UINT16
			>;
		};

		std::true_type CanMarshalObject(const NatPortBinding&);

		struct NatSettings
		{
			std::wstring Name = std::wstring{};
			std::vector<NatPortBinding> PortBindings = std::vector<::Config::Containers::NatPortBinding>{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				std::vector<::Config::Containers::NatPortBinding>
			>;
		};

		std::true_type CanMarshalObject(const NatSettings&);

		struct NetworkConnection
		{
			std::wstring NetworkName = std::wstring{};
			std::wstring Ip4Address = std::wstring{};
			bool EnableNat = bool{};
			NatSettings Nat = ::Config::Containers::NatSettings{};
			vmstd::optional<UINT64> MaximumOutgoingBandwidthInBytes = vmstd::optional<UINT64>{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				std::wstring,
				bool,
				::Config::Containers::NatSettings,
				vmstd::optional<UINT64>
			>;
		};

		std::true_type CanMarshalObject(const NetworkConnection&);

		struct NetworkSettings
		{
			vmstd::optional<MacAddressUtils::MacAddress> MacAddress = vmstd::optional<MacAddressUtils::MacAddress>{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				vmstd::optional<MacAddressUtils::MacAddress>
			>;
		};

		std::true_type CanMarshalObject(const NetworkSettings&);

		struct NetworkDevice : public DeviceBase
		{
			NetworkConnection Connection = ::Config::Containers::NetworkConnection{};
			NetworkSettings Settings = ::Config::Containers::NetworkSettings{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				::Config::Containers::NetworkConnection,
				::Config::Containers::NetworkSettings
			>;
		};

		std::true_type CanMarshalObject(const NetworkDevice&);

		struct Subnet
		{
			std::wstring GatewayAddress = std::wstring{};
			std::wstring AddressPrefix = std::wstring{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				std::wstring
			>;
		};

		std::true_type CanMarshalObject(const Subnet&);

		struct MacPool
		{
			std::wstring StartMacAddress = std::wstring{};
			std::wstring EndMacAddress = std::wstring{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				std::wstring,
				std::wstring
			>;
		};

		std::true_type CanMarshalObject(const MacPool&);

		struct Policy
		{
			PolicyType Type = ::Config::Containers::PolicyType{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::PolicyType
			>;
		};

		std::true_type CanMarshalObject(const Policy&);

		struct NatPolicyData : public Policy
		{
			NatPortProtocol Protocol = ::Config::Containers::NatPortProtocol{};
			UINT16 InternalPort = UINT16{};
			UINT16 ExternalPort = UINT16{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::PolicyType,
				::Config::Containers::NatPortProtocol,
				UINT16,
				UINT16
			>;
		};

		std::true_type CanMarshalObject(const NatPolicyData&);

		struct QosPolicyData : public Policy
		{
			UINT64 MaximumOutgoingBandwidthInBytes = UINT64{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::PolicyType,
				UINT64
			>;
		};

		std::true_type CanMarshalObject(const QosPolicyData&);

		struct VlanPolicyData : public Policy
		{
			UINT32 VLAN = UINT32{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::PolicyType,
				UINT32
			>;
		};

		std::true_type CanMarshalObject(const VlanPolicyData&);

		struct VsidPolicyData : public Policy
		{
			UINT32 VSID = UINT32{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				::Config::Containers::PolicyType,
				UINT32
			>;
		};

		std::true_type CanMarshalObject(const VsidPolicyData&);

		struct NetworkEndpoint
		{
			GUID Id = GUID{};
			std::wstring EndpointName = std::wstring{};
			std::wstring StaticMacAddress = std::wstring{};
			std::wstring StaticIPAddress = std::wstring{};
			GUID NetworkId = GUID{};
			std::vector<std::wstring> Policies = std::vector<std::wstring>{};

			static const Marshal::Details::ArrayView<Marshal::FieldData> FieldData;
			using FieldTypes = Marshal::TypeList<
				GUID,
				std::wstring,
				std::wstring,
				std::wstring,
				GUID,
				std::vector<std::wstring>
			>;
		};

		std::true_type CanMarshalObject(const NetworkEndpoint&);

	}
}
