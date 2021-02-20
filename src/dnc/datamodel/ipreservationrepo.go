package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
)

type ReservationRepo struct {
	Datastore datastoreshim.DataStoreShim
}

// Todo: Move api from networkManager to reservationRepo
func (*ReservationRepo) TryQueryIPReservation(networkID, reservationID string) (ReservationRow, *api.JsonErr) {
	return TryQueryIPReservation(networkID, reservationID)
}

func (*ReservationRepo) TryQueryIPReservationByKey(key string) (ReservationRow, *api.JsonErr) {
	return TryQueryIPReservationByKey(key)
}

func (*ReservationRepo) PutIPReservationConfiguration(res *ReservationConfiguration) *api.JsonErr {
	return PutIPReservationConfiguration(res)
}

func (r *ReservationRepo) UpdateReservations(reservations []ReservationConfiguration) *api.JsonErr {
	return UpdateIPReservations(reservations)
}

func (r *ReservationRepo) DeleteReservation(reservationKey string) *api.JsonErr {
	deleteErr := r.Datastore.Delete(ipReservationsTableName, reservationKey)
	if deleteErr != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete configuration for reservationKey: %s. Error: %s", reservationKey, deleteErr)
	}

	return nil
}
