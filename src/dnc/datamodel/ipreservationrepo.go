package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"encoding/json"
	"net/http"
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

func (r *ReservationRepo) AnyReservationConfigurationExists() (bool, *api.JsonErr) {
	exists, err := r.Datastore.AnyRowExists(ipReservationsTableName)
	if err != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to Query networkContainerTableName due to error: %+v",
			err)
	}

	return exists, nil
}

// TryQueryAllReservations returns all the reservations in the dnc partition
func (r *ReservationRepo) TryQueryAllReservations() ([]ReservationRow, *api.JsonErr) {
	var context interface{}
	var rows []ReservationRow
	for {
		entities, hasMoreData, cx, status, err := r.Datastore.QueryAllRows(ipReservationsTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return []ReservationRow{}, nil
			}

			return nil, api.InternalServerErrorf(api.StatusInternalServerError,
				"Failed to Query all ipReservationsTableName rows due to error: %+v",
				err)
		}

		for _, entity := range entities {
			var reservationConfiguration ReservationConfiguration
			if err = json.Unmarshal([]byte(entity.Properties[reservationConfigProperty].(string)), &reservationConfiguration); err != nil {
				return []ReservationRow{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
					"could not deserialize reservation %s. Error: %v", entity.RowKey, err)
			}

			row := ReservationRow{
				RowKey:        entity.RowKey,
				Configuration: &reservationConfiguration,
			}

			if prop, ok := entity.Properties[subnetKeyProperty]; ok {
				row.SubnetKey = prop.(string)
			}

			rows = append(rows, row)
		}

		if !hasMoreData {
			break
		}
		context = cx
	}

	return rows, nil
}

func (r *ReservationRepo) DeleteAllRows() *api.JsonErr {
	err := r.Datastore.WipePartition(ipReservationsTableName)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to wipe ipReservationsTableName due to error: %+v",
			err)
	}

	return nil
}
