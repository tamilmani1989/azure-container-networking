// This file is generated code.
// DO NOT CHANGE THIS FILE. Change the MOF instead and rerun the MarshalGen.exe.
//
// Copyright (c) Microsoft Corporation. All rights reserved.

#include "Config.Containers.h"

namespace Config
{
	namespace Containers
	{

		const PCWSTR SystemType_EnumData_Names[] =
		{
			L"Container",
			L"VirtualMachine",
		};

		const PCWSTR SystemType_EnumData_SortedNames[] =
		{
			L"Container",
			L"VirtualMachine",
		};

		const UINT32 SystemType_EnumData_SortedNamesToValues[] =
		{
			0,
			1,
		};

		const Marshal::EnumData SystemType_EnumData =
		{
			SystemType_EnumData_Names,
			SystemType_EnumData_SortedNames,
			SystemType_EnumData_SortedNamesToValues,
			2
		};

		const Marshal::EnumData& GetEnumData(SystemType)
		{
			return SystemType_EnumData;
		}

		const PCWSTR ContainerType_EnumData_Names[] =
		{
			L"VmHosted",
		};

		const PCWSTR ContainerType_EnumData_SortedNames[] =
		{
			L"VmHosted",
		};

		const UINT32 ContainerType_EnumData_SortedNamesToValues[] =
		{
			0,
		};

		const Marshal::EnumData ContainerType_EnumData =
		{
			ContainerType_EnumData_Names,
			ContainerType_EnumData_SortedNames,
			ContainerType_EnumData_SortedNamesToValues,
			1
		};

		const Marshal::EnumData& GetEnumData(ContainerType)
		{
			return ContainerType_EnumData;
		}

		const PCWSTR NatPortProtocol_EnumData_Names[] =
		{
			L"TCP",
			L"UDP",
		};

		const PCWSTR NatPortProtocol_EnumData_SortedNames[] =
		{
			L"TCP",
			L"UDP",
		};

		const UINT32 NatPortProtocol_EnumData_SortedNamesToValues[] =
		{
			0,
			1,
		};

		const Marshal::EnumData NatPortProtocol_EnumData =
		{
			NatPortProtocol_EnumData_Names,
			NatPortProtocol_EnumData_SortedNames,
			NatPortProtocol_EnumData_SortedNamesToValues,
			2
		};

		const Marshal::EnumData& GetEnumData(NatPortProtocol)
		{
			return NatPortProtocol_EnumData;
		}

		const PCWSTR NetworkMode_EnumData_Names[] =
		{
			L"NAT",
			L"Transparent",
			L"L2Bridge",
			L"L2Tunnel",
		};

		const PCWSTR NetworkMode_EnumData_SortedNames[] =
		{
			L"L2Bridge",
			L"L2Tunnel",
			L"NAT",
			L"Transparent",
		};

		const UINT32 NetworkMode_EnumData_SortedNamesToValues[] =
		{
			2,
			3,
			0,
			1,
		};

		const Marshal::EnumData NetworkMode_EnumData =
		{
			NetworkMode_EnumData_Names,
			NetworkMode_EnumData_SortedNames,
			NetworkMode_EnumData_SortedNamesToValues,
			4
		};

		const Marshal::EnumData& GetEnumData(NetworkMode)
		{
			return NetworkMode_EnumData;
		}

		const PCWSTR PolicyType_EnumData_Names[] =
		{
			L"NAT",
			L"QOS",
			L"VLAN",
			L"VSID",
		};

		const PCWSTR PolicyType_EnumData_SortedNames[] =
		{
			L"NAT",
			L"QOS",
			L"VLAN",
			L"VSID",
		};

		const UINT32 PolicyType_EnumData_SortedNamesToValues[] =
		{
			0,
			1,
			2,
			3,
		};

		const Marshal::EnumData PolicyType_EnumData =
		{
			PolicyType_EnumData_Names,
			PolicyType_EnumData_SortedNames,
			PolicyType_EnumData_SortedNamesToValues,
			4
		};

		const Marshal::EnumData& GetEnumData(PolicyType)
		{
			return PolicyType_EnumData;
		}

		const Marshal::FieldData Layer_FieldData[]
		{
			{ L"Id", offsetof(::Config::Containers::Layer, Id), Marshal::FieldOption::Required },
			{ L"Path", offsetof(::Config::Containers::Layer, Path), Marshal::FieldOption::Required },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> Layer::FieldData{ Layer_FieldData };

		const Marshal::FieldData MappedDirectory_FieldData[]
		{
			{ L"HostPath", offsetof(::Config::Containers::MappedDirectory, HostPath), Marshal::FieldOption::Required },
			{ L"ContainerPath", offsetof(::Config::Containers::MappedDirectory, ContainerPath), Marshal::FieldOption::Required },
			{ L"ReadOnly", offsetof(::Config::Containers::MappedDirectory, ReadOnly), Marshal::FieldOption::OmitEmpty },
			{ L"IOPSMaximum", offsetof(::Config::Containers::MappedDirectory, IOPSMaximum), Marshal::FieldOption::OmitEmpty },
			{ L"BandwidthMaximum", offsetof(::Config::Containers::MappedDirectory, BandwidthMaximum), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> MappedDirectory::FieldData{ MappedDirectory_FieldData };

		const Marshal::FieldData DeviceBase_FieldData[]
		{
			{ L"DeviceType", offsetof(::Config::Containers::DeviceBase, DeviceType), Marshal::FieldOption::None },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> DeviceBase::FieldData{ DeviceBase_FieldData };

		const Marshal::FieldData UtilityVmNetworkSettings_FieldData[]
		{
			{ L"SwitchName", offsetof(::Config::Containers::UtilityVmNetworkSettings, SwitchName), Marshal::FieldOption::OmitEmpty },
			{ L"PortName", offsetof(::Config::Containers::UtilityVmNetworkSettings, PortName), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> UtilityVmNetworkSettings::FieldData{ UtilityVmNetworkSettings_FieldData };

		const Marshal::FieldData UtilityVmSettings_FieldData[]
		{
			{ L"ImagePath", offsetof(::Config::Containers::UtilityVmSettings, ImagePath), Marshal::FieldOption::OmitEmpty },
			{ L"RuntimeId", offsetof(::Config::Containers::UtilityVmSettings, RuntimeId), Marshal::FieldOption::OmitEmpty },
			{ L"SkipTemplate", offsetof(::Config::Containers::UtilityVmSettings, SkipTemplate), Marshal::FieldOption::OmitEmpty },
			{ L"EnableConsole", offsetof(::Config::Containers::UtilityVmSettings, EnableConsole), Marshal::FieldOption::OmitEmpty },
			{ L"NetworkSettings", offsetof(::Config::Containers::UtilityVmSettings, NetworkSettings), Marshal::FieldOption::OmitEmpty },
			{ L"Com1PipeName", offsetof(::Config::Containers::UtilityVmSettings, Com1PipeName), Marshal::FieldOption::OmitEmpty },
			{ L"Com2PipeName", offsetof(::Config::Containers::UtilityVmSettings, Com2PipeName), Marshal::FieldOption::OmitEmpty },
			{ L"EnableUefiDebugger", offsetof(::Config::Containers::UtilityVmSettings, EnableUefiDebugger), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> UtilityVmSettings::FieldData{ UtilityVmSettings_FieldData };

		const Marshal::FieldData SettingsBase_FieldData[]
		{
			{ L"SystemType", offsetof(::Config::Containers::SettingsBase, SystemType), Marshal::FieldOption::Required },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> SettingsBase::FieldData{ SettingsBase_FieldData };

		const Marshal::FieldData ContainerSettingsBase_FieldData[]
		{
			{ L"SystemType", offsetof(::Config::Containers::ContainerSettingsBase, SystemType), Marshal::FieldOption::Required },
			{ L"Name", offsetof(::Config::Containers::ContainerSettingsBase, Name), Marshal::FieldOption::None },
			{ L"HvPartition", offsetof(::Config::Containers::ContainerSettingsBase, HvPartition), Marshal::FieldOption::OmitEmpty },
			{ L"ContainerType", offsetof(::Config::Containers::ContainerSettingsBase, ContainerType), Marshal::FieldOption::OmitEmpty },
			{ L"Servicing", offsetof(::Config::Containers::ContainerSettingsBase, Servicing), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> ContainerSettingsBase::FieldData{ ContainerSettingsBase_FieldData };

		const Marshal::FieldData ContainerSettings_FieldData[]
		{
			{ L"SystemType", offsetof(::Config::Containers::ContainerSettings, SystemType), Marshal::FieldOption::Required },
			{ L"Name", offsetof(::Config::Containers::ContainerSettings, Name), Marshal::FieldOption::None },
			{ L"HvPartition", offsetof(::Config::Containers::ContainerSettings, HvPartition), Marshal::FieldOption::OmitEmpty },
			{ L"ContainerType", offsetof(::Config::Containers::ContainerSettings, ContainerType), Marshal::FieldOption::OmitEmpty },
			{ L"Servicing", offsetof(::Config::Containers::ContainerSettings, Servicing), Marshal::FieldOption::OmitEmpty },
			{ L"Owner", offsetof(::Config::Containers::ContainerSettings, Owner), Marshal::FieldOption::OmitEmpty },
			{ L"IsDummy", offsetof(::Config::Containers::ContainerSettings, IsDummy), Marshal::FieldOption::OmitEmpty },
			{ L"TerminateOnLastHandleClosed", offsetof(::Config::Containers::ContainerSettings, TerminateOnLastHandleClosed), Marshal::FieldOption::OmitEmpty },
			{ L"HvRuntime", offsetof(::Config::Containers::ContainerSettings, HvRuntime), Marshal::FieldOption::OmitEmpty },
			{ L"HostName", offsetof(::Config::Containers::ContainerSettings, HostName), Marshal::FieldOption::OmitEmpty },
			{ L"Credentials", offsetof(::Config::Containers::ContainerSettings, Credentials), Marshal::FieldOption::OmitEmpty },
			{ L"MemoryMaximumInMB", offsetof(::Config::Containers::ContainerSettings, MemoryMaximumInMB), Marshal::FieldOption::OmitEmpty },
			{ L"ProcessorCount", offsetof(::Config::Containers::ContainerSettings, ProcessorCount), Marshal::FieldOption::OmitEmpty },
			{ L"ProcessorMaximum", offsetof(::Config::Containers::ContainerSettings, ProcessorMaximum), Marshal::FieldOption::OmitEmpty },
			{ L"ProcessorWeight", offsetof(::Config::Containers::ContainerSettings, ProcessorWeight), Marshal::FieldOption::OmitEmpty },
			{ L"IgnoreFlushesDuringBoot", offsetof(::Config::Containers::ContainerSettings, IgnoreFlushesDuringBoot), Marshal::FieldOption::OmitEmpty },
			{ L"SandboxPath", offsetof(::Config::Containers::ContainerSettings, SandboxPath), Marshal::FieldOption::OmitEmpty },
			{ L"VolumePath", offsetof(::Config::Containers::ContainerSettings, VolumePath), Marshal::FieldOption::OmitEmpty },
			{ L"LayerFolderPath", offsetof(::Config::Containers::ContainerSettings, LayerFolderPath), Marshal::FieldOption::OmitEmpty },
			{ L"Layers", offsetof(::Config::Containers::ContainerSettings, Layers), Marshal::FieldOption::OmitEmpty },
			{ L"MappedDirectories", offsetof(::Config::Containers::ContainerSettings, MappedDirectories), Marshal::FieldOption::OmitEmpty },
			{ L"StorageIOPSMaximum", offsetof(::Config::Containers::ContainerSettings, StorageIOPSMaximum), Marshal::FieldOption::OmitEmpty },
			{ L"StorageBandwidthMaximum", offsetof(::Config::Containers::ContainerSettings, StorageBandwidthMaximum), Marshal::FieldOption::OmitEmpty },
			{ L"StorageSandboxSize", offsetof(::Config::Containers::ContainerSettings, StorageSandboxSize), Marshal::FieldOption::OmitEmpty },
			{ L"Devices", offsetof(::Config::Containers::ContainerSettings, Devices), Marshal::FieldOption::OmitEmpty },
			{ L"NetworkEndpoints", offsetof(::Config::Containers::ContainerSettings, NetworkEndpoints), Marshal::FieldOption::OmitEmpty },
			{ L"EndpointList", offsetof(::Config::Containers::ContainerSettings, EndpointList), Marshal::FieldOption::OmitEmpty },
			{ L"AllowUnqualifiedDnsQuery", offsetof(::Config::Containers::ContainerSettings, EnableServiceDiscovery), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> ContainerSettings::FieldData{ ContainerSettings_FieldData };

		const Marshal::FieldData NatPortBinding_FieldData[]
		{
			{ L"Protocol", offsetof(::Config::Containers::NatPortBinding, Protocol), Marshal::FieldOption::None },
			{ L"InternalPort", offsetof(::Config::Containers::NatPortBinding, InternalPort), Marshal::FieldOption::None },
			{ L"ExternalPort", offsetof(::Config::Containers::NatPortBinding, ExternalPort), Marshal::FieldOption::None },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NatPortBinding::FieldData{ NatPortBinding_FieldData };

		const Marshal::FieldData NatSettings_FieldData[]
		{
			{ L"Name", offsetof(::Config::Containers::NatSettings, Name), Marshal::FieldOption::OmitEmpty },
			{ L"PortBindings", offsetof(::Config::Containers::NatSettings, PortBindings), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NatSettings::FieldData{ NatSettings_FieldData };

		const Marshal::FieldData NetworkConnection_FieldData[]
		{
			{ L"NetworkName", offsetof(::Config::Containers::NetworkConnection, NetworkName), Marshal::FieldOption::None },
			{ L"Ip4Address", offsetof(::Config::Containers::NetworkConnection, Ip4Address), Marshal::FieldOption::OmitEmpty },
			{ L"EnableNat", offsetof(::Config::Containers::NetworkConnection, EnableNat), Marshal::FieldOption::None },
			{ L"Nat", offsetof(::Config::Containers::NetworkConnection, Nat), Marshal::FieldOption::OmitEmpty },
			{ L"MaximumOutgoingBandwidthInBytes", offsetof(::Config::Containers::NetworkConnection, MaximumOutgoingBandwidthInBytes), Marshal::FieldOption::None },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NetworkConnection::FieldData{ NetworkConnection_FieldData };

		const Marshal::FieldData NetworkSettings_FieldData[]
		{
			{ L"MacAddress", offsetof(::Config::Containers::NetworkSettings, MacAddress), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NetworkSettings::FieldData{ NetworkSettings_FieldData };

		const Marshal::FieldData NetworkDevice_FieldData[]
		{
			{ L"DeviceType", offsetof(::Config::Containers::NetworkDevice, DeviceType), Marshal::FieldOption::None },
			{ L"Connection", offsetof(::Config::Containers::NetworkDevice, Connection), Marshal::FieldOption::OmitEmpty },
			{ L"Settings", offsetof(::Config::Containers::NetworkDevice, Settings), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NetworkDevice::FieldData{ NetworkDevice_FieldData };

		const Marshal::FieldData Subnet_FieldData[]
		{
			{ L"GatewayAddress", offsetof(::Config::Containers::Subnet, GatewayAddress), Marshal::FieldOption::OmitEmpty },
			{ L"AddressPrefix", offsetof(::Config::Containers::Subnet, AddressPrefix), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> Subnet::FieldData{ Subnet_FieldData };

		const Marshal::FieldData MacPool_FieldData[]
		{
			{ L"StartMacAddress", offsetof(::Config::Containers::MacPool, StartMacAddress), Marshal::FieldOption::OmitEmpty },
			{ L"EndMacAddress", offsetof(::Config::Containers::MacPool, EndMacAddress), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> MacPool::FieldData{ MacPool_FieldData };

		const Marshal::FieldData Policy_FieldData[]
		{
			{ L"Type", offsetof(::Config::Containers::Policy, Type), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> Policy::FieldData{ Policy_FieldData };

		const Marshal::FieldData NatPolicyData_FieldData[]
		{
			{ L"Type", offsetof(::Config::Containers::NatPolicyData, Type), Marshal::FieldOption::OmitEmpty },
			{ L"Protocol", offsetof(::Config::Containers::NatPolicyData, Protocol), Marshal::FieldOption::OmitEmpty },
			{ L"InternalPort", offsetof(::Config::Containers::NatPolicyData, InternalPort), Marshal::FieldOption::OmitEmpty },
			{ L"ExternalPort", offsetof(::Config::Containers::NatPolicyData, ExternalPort), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NatPolicyData::FieldData{ NatPolicyData_FieldData };

		const Marshal::FieldData QosPolicyData_FieldData[]
		{
			{ L"Type", offsetof(::Config::Containers::QosPolicyData, Type), Marshal::FieldOption::OmitEmpty },
			{ L"MaximumOutgoingBandwidthInBytes", offsetof(::Config::Containers::QosPolicyData, MaximumOutgoingBandwidthInBytes), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> QosPolicyData::FieldData{ QosPolicyData_FieldData };

		const Marshal::FieldData VlanPolicyData_FieldData[]
		{
			{ L"Type", offsetof(::Config::Containers::VlanPolicyData, Type), Marshal::FieldOption::OmitEmpty },
			{ L"VLAN", offsetof(::Config::Containers::VlanPolicyData, VLAN), Marshal::FieldOption::None },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> VlanPolicyData::FieldData{ VlanPolicyData_FieldData };

		const Marshal::FieldData VsidPolicyData_FieldData[]
		{
			{ L"Type", offsetof(::Config::Containers::VsidPolicyData, Type), Marshal::FieldOption::OmitEmpty },
			{ L"VSID", offsetof(::Config::Containers::VsidPolicyData, VSID), Marshal::FieldOption::None },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> VsidPolicyData::FieldData{ VsidPolicyData_FieldData };

		const Marshal::FieldData NetworkEndpoint_FieldData[]
		{
			{ L"Id", offsetof(::Config::Containers::NetworkEndpoint, Id), Marshal::FieldOption::Required },
			{ L"EndpointName", offsetof(::Config::Containers::NetworkEndpoint, EndpointName), Marshal::FieldOption::OmitEmpty },
			{ L"StaticMacAddress", offsetof(::Config::Containers::NetworkEndpoint, StaticMacAddress), Marshal::FieldOption::OmitEmpty },
			{ L"StaticIPAddress", offsetof(::Config::Containers::NetworkEndpoint, StaticIPAddress), Marshal::FieldOption::OmitEmpty },
			{ L"NetworkId", offsetof(::Config::Containers::NetworkEndpoint, NetworkId), Marshal::FieldOption::Required },
			{ L"Policies", offsetof(::Config::Containers::NetworkEndpoint, Policies), Marshal::FieldOption::OmitEmpty },
		};

		const Marshal::Details::ArrayView<Marshal::FieldData> NetworkEndpoint::FieldData{ NetworkEndpoint_FieldData };

	}
}
