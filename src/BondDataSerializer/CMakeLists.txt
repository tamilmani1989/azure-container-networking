cmake_minimum_required(VERSION 3.10)
project(tmp VERSION 1.0.1)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_BUILD_TYPE Release)
set(NCDIR ${CMAKE_CURRENT_SOURCE_DIR})

set(gbc "gbc") 
set(cpp "c++")
set(ncdata "${NCDIR}/BondDataTypes/NcData.bond")
set(ipAddress "${NCDIR}/BondDataTypes/IPAddress.bond")
set(macAddress "${NCDIR}/BondDataTypes/MacAddress.bond")
set(pubsubMapping "${NCDIR}/BondDataTypes/PubSubMapping.bond")
set(subnet "${NCDIR}/BondDataTypes/Subnet.bond")
set(vnetconfig "${NCDIR}/BondDataTypes/VNetConfiguration.bond")
set(vnetAddress "${NCDIR}/BondDataTypes/VNetAddress.bond")
set(gwInfo "${NCDIR}/BondDataTypes/GatewayInfo.bond")
execute_process(COMMAND ${gbc} ${cpp} ${ncdata} -o ${NCDIR})
execute_process(COMMAND ${gbc} ${cpp} ${ipAddress} -o ${NCDIR})
execute_process(COMMAND ${gbc} ${cpp} ${macAddress} -o ${NCDIR})
execute_process(COMMAND ${gbc} ${cpp} ${subnet} -o ${NCDIR})
execute_process(COMMAND ${gbc} ${cpp} ${vnetconfig} -o ${NCDIR})
execute_process(COMMAND ${gbc} ${cpp} ${vnetAddress} -o ${NCDIR})
execute_process(COMMAND ${gbc} ${cpp} ${gwInfo} -o ${NCDIR})
message("Generated bond files at ${NCDIR}")


add_library(BondDataSerializer SHARED
    ${NCDIR}/BondDataSerializer.cpp
	NcData_apply.cpp
	NcData_types.cpp
	IPAddress_apply.cpp
	IPAddress_types.cpp
	MacAddress_apply.cpp
	MacAddress_types.cpp
	Subnet_apply.cpp
	Subnet_types.cpp
	VNetConfiguration_apply.cpp
	VNetConfiguration_types.cpp
	VNetAddress_apply.cpp
	VNetAddress_types.cpp
	GatewayInfo_apply.cpp
	GatewayInfo_types.cpp
)

target_link_libraries(BondDataSerializer)
