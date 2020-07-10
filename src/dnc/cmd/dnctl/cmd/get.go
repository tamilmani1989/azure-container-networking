package cmd

import (
	"dnc/client"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

// GetCmd can get an item from DNC
func GetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "get",
		Short: "Gets an item from DNC",
	}
	cmd.AddCommand(getNodeCmd())
	cmd.AddCommand(getVnetCmd())
	cmd.AddCommand(getReservationCmd())
	cmd.AddCommand(getSubnetCmd())
	cmd.AddCommand(getNetworkContainerCmd())

	return cmd
}

func getNodeCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "node",
		Short: "Gets a node",
		Run: func(cmd *cobra.Command, args []string) {
			id, _ := cmd.Flags().GetString("id")

			node, err := DncClient.GetNode(id)
			if err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", node)
		},
	}
	cmd.Flags().String("id", "", "Node id")
	cmd.MarkFlagRequired("id")
	return cmd
}

func getVnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "vnet",
		Short: "Gets an item from DNC",
	}
	cmd.AddCommand(getVnetStatusCmd())
	cmd.AddCommand(getVnetConfigCmd())
	cmd.AddCommand(getVnetIdsCmd())

	return cmd
}

func getVnetStatusCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "status",
		Short: "Gets a vnet",
		Run: func(cmd *cobra.Command, args []string) {
			id, _ := cmd.Flags().GetString("id")

			vnet, err := DncClient.GetNetworkStatus(id)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.Flags().String("id", "", "set name")
	cmd.MarkFlagRequired("id")
	return cmd
}

func getVnetConfigCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "config",
		Short: "Gets a vnet",
		Run: func(cmd *cobra.Command, args []string) {
			id, _ := cmd.Flags().GetString("id")

			vnet, err := DncClient.GetNetworkConfiguration(id)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.Flags().String("id", "", "set name")
	cmd.MarkFlagRequired("id")
	return cmd
}

func getVnetIdsCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "ids",
		Short: "Gets all vnets",
		Run: func(cmd *cobra.Command, args []string) {
			networksResponse, err := DncClient.GetNetworks()
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			for _, networkInfo := range networksResponse.NetworkInfos {
				fmt.Printf("%v\n", networkInfo.GUID)
			}
		},
	}
	return cmd
}

func getSubnetCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "subnet",
		Short: "Gets an subnet from DNC",
	}
	cmd.AddCommand(getSubnetStatusCmd())
	cmd.AddCommand(getSubnetConfigCmd())

	return cmd
}

func getSubnetStatusCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "status",
		Short: "Gets a vnet",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			subnetid, _ := cmd.Flags().GetString("subnet-id")

			vnet, err := DncClient.GetSubnetStatus(vnetid, subnetid)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("subnet-id", "", "subnet-id")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("subnet-id")
	return cmd
}

func getSubnetConfigCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "config",
		Short: "Gets a subnet",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			ncid, _ := cmd.Flags().GetString("nc-id")

			vnet, err := DncClient.GetSubnetConfiguration(vnetid, ncid)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("subnet-id", "", "subnet-id")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("subnet-id")
	return cmd
}

func getReservationCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "reservation",
		Short: "Gets an reservation from DNC",
	}
	cmd.Aliases = []string{"res"}
	cmd.AddCommand(getReservationStatusCmd())
	cmd.AddCommand(getReservationConfigCmd())

	return cmd
}

func getReservationStatusCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "status",
		Short: "Gets reservation status",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			reservationid, _ := cmd.Flags().GetString("reservation-id")

			reservation, err := DncClient.GetReservationStatus(vnetid, reservationid)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", reservation)
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("reservation-id", "", "reservation-id")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("reservation-id")
	return cmd
}

func getReservationConfigCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "config",
		Short: "Gets reservation configuration",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			reservationid, _ := cmd.Flags().GetString("reservation-id")

			reservation, err := DncClient.GetReservation(vnetid, reservationid)
			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", reservation)
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("reservation-id", "", "reservation-id")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("reservation-id")
	return cmd
}

func getNetworkContainerCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "nc",
		Short: "Gets an network container from DNC",

		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			ncid, _ := cmd.Flags().GetString("nc-id")

			vnet, err := DncClient.GetNC(vnetid, ncid)

			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.AddCommand(getNetworkContainerConfigCmd())
	cmd.AddCommand(getNetworkContainerStatusCmd())
	return cmd
}

func getNetworkContainerConfigCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "config",
		Short: "Gets a NC Config",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			ncid, _ := cmd.Flags().GetString("nc-id")

			vnet, err := DncClient.GetNCConfiguration(vnetid, ncid)

			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("nc-id", "", "nc-id")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("nc-id")
	return cmd
}

func getNetworkContainerStatusCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "status",
		Short: "Gets a NC status",
		Run: func(cmd *cobra.Command, args []string) {
			vnetid, _ := cmd.Flags().GetString("vnet-id")
			ncid, _ := cmd.Flags().GetString("nc-id")

			vnet, err := DncClient.GetNCStatus(vnetid, ncid)

			if err != nil && err.HTTPStatus != http.StatusOK {
				client.PrettyPrint(err)
				return
			}
			fmt.Printf("%+v\n", vnet)
		},
	}
	cmd.Flags().String("vnet-id", "", "vnet-id")
	cmd.Flags().String("nc-id", "", "nc-id")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("nc-id")
	return cmd
}
