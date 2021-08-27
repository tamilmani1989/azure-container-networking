package ipam

import (
	"context"
	"errors"
	"net"
	"os"
	"runtime"

	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultIPv4SubnetMaskSizeLimit = 24
	ipv4BitLength                  = 32
)

type nodeIpamV4 struct {
	name           string
	nodeHostname   string
	kubeConfigPath string
	kubeClient     kubernetes.Interface
	kubeNode       *v1.Node
	isLoaded       bool
	sink           addressConfigSink
}

// creates a new IPv6 Ipam source
func newNodeIpamV4Source(options map[string]interface{}, isLoaded bool) (*nodeIpamV4, error) {
	var kubeConfigPath string
	name := options[common.OptEnvironment].(string)

	if runtime.GOOS == windows {
		kubeConfigPath = defaultWindowsKubeConfigFilePath
	} else {
		kubeConfigPath = defaultLinuxKubeConfigFilePath
	}

	nodeName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &nodeIpamV4{
		name:           name,
		nodeHostname:   nodeName,
		kubeConfigPath: kubeConfigPath,
		isLoaded:       isLoaded,
	}, nil
}

// Starts the MAS source.
func (source *nodeIpamV4) start(sink addressConfigSink) error {
	source.sink = sink
	return nil
}

// Stops the MAS source.
func (source *nodeIpamV4) stop() {
	source.sink = nil
}

// creates a KubernetesClientset using the Kubeconfig stored on each agent node
func (source *nodeIpamV4) loadKubernetesConfig() (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", source.kubeConfigPath)
	if err != nil {
		log.Printf("[ipam] Failed to load Kubernetes config from disk: %+v", err)
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("[ipam] Failed to create Kubernetes config: %+v", err)
		return nil, err
	}

	return client, err
}

func (source *nodeIpamV4) refresh() error {
	if source == nil {
		return errors.New("ipv6ipam is nil")
	}

	if source.isLoaded {
		return nil
	}

	if source.kubeClient == nil {
		kubeClient, err := source.loadKubernetesConfig()
		if err != nil {
			log.Printf("[ipam] Failed to load Kubernetes config: %+v", err)
			return err
		}

		source.kubeClient = kubeClient
	}

	kubeNode, err := source.kubeClient.CoreV1().Nodes().Get(context.TODO(), source.nodeHostname, metav1.GetOptions{})
	if err != nil {
		log.Printf("[ipam] Failed to retrieve node using hostname: %+v with err %+v", source.nodeHostname, err)
		return err
	}

	source.kubeNode = kubeNode
	log.Printf("[ipam] Discovered IPv4 CIDR's %v.", source.kubeNode.Spec.PodCIDRs)

	// Query the list of Kubernetes Pod IPs
	interfaceIPs, err := retrieveK8sV4PodCidr(source.kubeNode)
	if err != nil {
		log.Printf("[ipam] Failed retrieve Kubernetes IP's from subnet: %v.", err)
		return err
	}

	// Configure the local default address space.
	local, err := source.sink.newAddressSpace(LocalDefaultAddressSpaceId, LocalScope)
	if err != nil {
		log.Printf("[ipam] Failed to configure local default address space: %v.", err)
		return err
	}

	for _, i := range interfaceIPs.Interfaces {
		for _, s := range i.IPSubnets {
			_, subnet, err := net.ParseCIDR(s.Prefix)
			ifaceName := ""
			priority := 0
			ap, err := local.newAddressPool(ifaceName, priority, subnet)

			for _, a := range s.IPAddresses {
				address := net.ParseIP(a.Address)
				_, err = ap.newAddressRecord(&address)
				if err != nil {
					log.Printf("[ipam] Failed to create address:%v err:%v.", address, err)
					continue
				}
			}
		}
	}

	// Set the local address space as active.
	if err = source.sink.setAddressSpace(local); err != nil {
		return err
	}

	log.Printf("[ipam] Address space successfully populated from Kubernetes API Server")
	return err
}

// retrieves the allocated pod IP's, and populates the NetworkInterfaces struture
func retrieveK8sV4PodCidr(node *v1.Node) (*NetworkInterfaces, error) {
	var nodeCidr *net.IPNet
	var startIP net.IP

	// get IPv6 subnet allocated to node
	for _, cidr := range node.Spec.PodCIDRs {
		ip, subnetCidr, _ := net.ParseCIDR(cidr)
		if ip.To4() != nil {
			startIP = ip
			nodeCidr = subnetCidr
			break
		}
	}

	if nodeCidr == nil {
		return nil, errors.New("[ipam] Failed to retrieve subnet, node doesn't have IPv4 subnet allocated from Kubernetes")
	}

	ones, _ := nodeCidr.Mask.Size()
	if ones < 24 {
		log.Printf("Subnet size is bigger /%d than expected. using /24", ones)
		nodeCidr.Mask = net.CIDRMask(defaultIPv4SubnetMaskSizeLimit, ipv4BitLength)
	}

	addresses := getIPsFromAddresses(startIP, nodeCidr)
	networkSubnet := IPSubnet{
		Prefix: nodeCidr.String(),
	}

	// skip the first address, explicitly save all IP's from the given subnet
	for i := 2; i < len(addresses); i++ {
		ipaddress := IPAddress{
			IsPrimary: false,
			Address:   addresses[i].String(),
		}
		networkSubnet.IPAddresses = append(networkSubnet.IPAddresses, ipaddress)
	}

	return &NetworkInterfaces{
		Interfaces: []Interface{
			{
				IsPrimary: true,
				IPSubnets: []IPSubnet{
					networkSubnet,
				},
			},
		},
	}, nil
}
