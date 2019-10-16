package utils

// default vnet info
const (
	DefaultDNSServers = "8.8.8.8,9.9.9.9"
	Subnet1           = "subnet1"
	Subnet2           = "subnet2"
	Subnet3           = "subnet3"
	Subnet1AddrSpace  = "10.0.1.0/24"
	Subnet2AddrSpace  = "10.0.2.0/24"
	Subnet3AddrSpace  = "10.0.3.0/24"
	Subnet1DefaultGW  = "10.0.1.1"
	Subnet2DefaultGW  = "10.0.2.1"
	Subnet3DefaultGW  = "10.0.3.1"
	DefaultCnetSpace  = "10.0.0.0/8,172.16.0.0/16,192.168.1.0/24"
	DefaultVnetSpace  = "10.0.0.0/16"
)

// vnets
const (
	OneMinuteFailureVnet = "3c35875a-bfaf-4192-a511-303a7e0a5039"
	ThrottledVnet        = "8ecb43ed-96f6-4670-bb67-3f7d4dd39880"
	PendingVnet          = "c774ea03-0009-46aa-8774-b9fcb2aba33a"
)

// NCs
const (
	OneMinuteFailureNC = "6a8d7f5d-7fcc-4a25-a9e1-e828000bbafb"
	ThrottledNC        = "acd1b609-763f-4dd6-9ca4-c4cb2c4a16d1"
	UnauthorizedNC     = "617a3122-1ef6-481c-b6dc-9851fef130b2"
)
