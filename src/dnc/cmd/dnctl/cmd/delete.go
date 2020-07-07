package cmd

import (
	"dnc/client"

	"github.com/spf13/cobra"
)

// DeleteCmd can delete an item to DNC
func DeleteCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "delete",
		Short: "Deletes an item to DNC",
	}
	cmd.AddCommand(deleteNetworkContainerCmd())
	cmd.AddCommand(deleteReservationCmd())
	cmd.Aliases = []string{"del"}

	return cmd
}

func deleteNetworkContainerCmd() *cobra.Command {
	var (
		vnetID string
		ncID   string
	)

	var cmd = &cobra.Command{
		Use:   "nc",
		Short: "Deletes a network container",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.DeleteNC(vnetID, ncID)
			if err != nil {
				client.PrettyPrint(err)
			}
		},
	}
	cmd.Flags().StringVar(&vnetID, "vnet-id", "", "Network ID")
	cmd.Flags().StringVar(&ncID, "nc-id", "", "Network Container ID")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("nc-id")
	return cmd
}

func deleteReservationCmd() *cobra.Command {
	var (
		vnetID        string
		reservationID string
	)

	var cmd = &cobra.Command{
		Use:   "reservation",
		Short: "Delete a reservation",
		Run: func(cmd *cobra.Command, args []string) {

			err := DncClient.DeleteReservation(vnetID, reservationID)
			if err != nil {
				client.PrettyPrint(err)
			}
		},
	}
	cmd.Aliases = []string{"res"}
	cmd.Flags().StringVar(&vnetID, "vnet-id", "", "Network ID")
	cmd.Flags().StringVar(&reservationID, "reservation-id", "", "Reservation ID")
	cmd.MarkFlagRequired("vnet-id")
	cmd.MarkFlagRequired("reservation-id")
	return cmd
}
