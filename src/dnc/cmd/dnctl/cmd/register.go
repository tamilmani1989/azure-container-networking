package cmd

import (
	"dnc/client"
	"net/http"

	"github.com/spf13/cobra"
)

// RegisterCmd can register an object
func RegisterCmd() *cobra.Command {
	var registercmd = &cobra.Command{
		Use:   "register",
		Short: "Registers an object with DNC",
	}
	registercmd.AddCommand(registerNodeCmd())
	registercmd.AddCommand(registerVnetCmd())
	registercmd.AddCommand(registerSubnetCmd())
	registercmd.AddCommand(registerAzStorageCmd())
	registercmd.Aliases = []string{"reg"}
	return registercmd
}

func registerNodeCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "node",
		Short: "Registers a node",
		Run: func(cmd *cobra.Command, args []string) {
			nodeid, _ := cmd.Flags().GetString("id")
			ipaddr, _ := cmd.Flags().GetString("ip-address")
			orchtype, _ := cmd.Flags().GetString("orchestrator-type")
			infravnet, _ := cmd.Flags().GetString("infra-vnet")
			azid, _ := cmd.Flags().GetString("az-id")
			nodetype, _ := cmd.Flags().GetString("node-type")
			nodeSet, _ := cmd.Flags().GetString("node-set")

			err := DncClient.AddNode(nodeid, ipaddr, orchtype, infravnet, azid, nodetype, nodeSet)
			if err != nil {
				client.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("id", "", "Name of the node")
	cmd.Flags().String("ip-address", "", "IP Address of the node")
	cmd.Flags().String("orchestrator-type", "", "The orchestrator type of the node")
	cmd.Flags().String("infra-vnet", "", "Node infrastructure network (guid)")
	cmd.Flags().String("az-id", "", "Availability Zone ID")
	cmd.Flags().String("node-type", "", "Node type")
	cmd.Flags().String("node-set", "", "node set")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("ip-address")
	cmd.MarkFlagRequired("orchestrator-type")

	return cmd
}

func registerVnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "vnet",
		Short: "Registers a VNET",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("id")

			err := DncClient.AddNetwork(vnetid)
			if err != nil {
				client.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().String("id", "", "The VNET ID for delegation")
	cmd.MarkFlagRequired("id")

	return cmd
}

func registerSubnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "subnet",
		Short: "Registers a subnet",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			subnetid, _ := cmd.Flags().GetString("subnet-id")
			authtoken, _ := cmd.Flags().GetString("auth-token")
			apiVer, _ := cmd.Flags().GetString("apiversion")

			err := DncClient.AddSubnet(vnetid, subnetid, authtoken, apiVer)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("subnet-id", "", "subnet-id")
	cmd.Flags().String("auth-token", "", "auth-token")
	cmd.Flags().String("apiversion", "", "apiversion")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("subnet-id")
	cmd.MarkFlagRequired("auth-id")
	return cmd
}

func registerAzStorageCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "azstorage",
		Short: "Register and connect DNC to Azure storage",
		Run: func(cmd *cobra.Command, args []string) {
			accountName, _ := cmd.Flags().GetString("name")
			key, _ := cmd.Flags().GetString("key")
			partitionKey, _ := cmd.Flags().GetString("partition-key")
			blob, _ := cmd.Flags().GetString("blob")
			container, _ := cmd.Flags().GetString("container")

			err := DncClient.InitAzStorage(accountName, key, partitionKey, blob, container)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
		},
	}
	cmd.Flags().String("name", "", "Storage account name")
	cmd.Flags().String("key", "", "Storage account key")
	cmd.Flags().String("partition-key", "", "Storage partition")
	cmd.Flags().String("blob", "", "Storage blob")
	cmd.Flags().String("container", "", "Storage container")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("partition-key")
	cmd.MarkFlagRequired("blob")
	cmd.MarkFlagRequired("container")
	return cmd
}