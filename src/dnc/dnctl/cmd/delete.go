package cmd

import (
	"dnc/dnctl/common"

	"github.com/spf13/cobra"
)

func DeleteCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "delete",
		Short: "Adds an item to DNC",
	}
	cmd.AddCommand(deleteNetworkContainerCmd())
	cmd.AddCommand(deleteReservationCmd())
	cmd.Aliases = []string{"del"}

	return cmd
}

func deleteNetworkContainerCmd() *cobra.Command {
	var (
		vnetId string
		ncId   string
	)

	var cmd = &cobra.Command{
		Use:   "nc",
		Short: "Deletes a network container",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.DeleteNC(vnetId, ncId)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().StringVar(&vnetId, "vnet-id", "", "Network ID")
	cmd.Flags().StringVar(&ncId, "nc-id", "", "Network Container ID")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("nc-id")
	return cmd
}

func deleteReservationCmd() *cobra.Command {
	var (
		vnetId        string
		reservationId string
	)

	var cmd = &cobra.Command{
		Use:   "reservation",
		Short: "Delete a reservation",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.DeleteReservation(vnetId, reservationId)
			if err != nil {
				common.PrettyPrint(err)
			}
		},
	}
	cmd.Aliases = []string{"res"}
	cmd.Flags().StringVar(&vnetId, "vnet-id", "", "Network ID")
	cmd.Flags().StringVar(&reservationId, "reservation-id", "", "Reservation ID")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("reservation-id")
	return cmd
}
