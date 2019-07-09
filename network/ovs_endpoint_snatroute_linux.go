package network

import (
	"fmt"

	"github.com/Azure/azure-container-networking/network/ovssnat"
)

func NewSnatClient(client *OVSEndpointClient, snatBridgeIP string, localIP string, epInfo *EndpointInfo) {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC {

		hostIfName := fmt.Sprintf("%s%s", snatVethInterfacePrefix, epInfo.Id[:7])
		contIfName := fmt.Sprintf("%s%s-2", snatVethInterfacePrefix, epInfo.Id[:7])

		client.snatClient = ovssnat.NewSnatClient(hostIfName, contIfName, localIP, snatBridgeIP, epInfo.DNS.Servers)
	}
}

func AddSnatEndpoint(client *OVSEndpointClient) error {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC || client.allowInboundFromNCToHost {
		if err := client.snatClient.CreateSnatEndpoint(client.bridgeName); err != nil {
			return err
		}
	}

	return nil
}

func AddSnatEndpointRules(client *OVSEndpointClient) error {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC || client.allowInboundFromNCToHost {
		if err := client.snatClient.AllowIPAddresses(); err != nil {
			return err
		}

		if err := client.snatClient.BlockIPAddresses(); err != nil {
			return err
		}

		if err := AddStaticRoute(ovssnat.ImdsIP, client.bridgeName); err != nil {
			return err
		}

		if client.allowInboundFromHostToNC {
			if err := client.snatClient.AllowInboundFromHostToNC(); err != nil {
				return err
			}
		}

		if client.allowInboundFromNCToHost {
			return client.snatClient.AllowInboundFromNCToHost()
		}
	}

	return nil
}

func MoveSnatEndpointToContainerNS(client *OVSEndpointClient, netnsPath string, nsID uintptr) error {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC || client.allowInboundFromNCToHost {
		return client.snatClient.MoveSnatEndpointToContainerNS(netnsPath, nsID)
	}

	return nil
}

func SetupSnatContainerInterface(client *OVSEndpointClient) error {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC || client.allowInboundFromNCToHost {
		return client.snatClient.SetupSnatContainerInterface()
	}

	return nil
}

func ConfigureSnatContainerInterface(client *OVSEndpointClient) error {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC || client.allowInboundFromNCToHost {
		return client.snatClient.ConfigureSnatContainerInterface()
	}

	return nil
}

func DeleteSnatEndpoint(client *OVSEndpointClient) error {
	if client.enableSnatOnHost || client.allowInboundFromHostToNC || client.allowInboundFromNCToHost {
		return client.snatClient.DeleteSnatEndpoint()
	}

	return nil
}

func DeleteSnatEndpointRules(client *OVSEndpointClient) error {
	if client.allowInboundFromHostToNC {
		client.snatClient.DeleteInboundFromHostToNC()
	}

	if client.allowInboundFromNCToHost {
		client.snatClient.DeleteInboundFromNCToHost()
	}

	return nil
}
