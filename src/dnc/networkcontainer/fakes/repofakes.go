package fakes

import (
	"dnc/api"
	"dnc/datamodel"
	"fmt"
	"time"
)

type NCRepo struct {
	Data     map[string]datamodel.NCConfiguration
	QueryErr *api.JsonErr
	PutErr   *api.JsonErr
}

func (n *NCRepo) TryQueryNetworkContainer(networkID, ncID string) (*datamodel.NCConfiguration, *api.JsonErr) {
	if n.QueryErr != nil {
		return nil, n.QueryErr
	}

	ncKey := datamodel.GetNCKey(networkID, ncID)
	nc, ok := n.Data[ncKey]
	if !ok {
		return nil, nil
	}

	return &nc, nil
}

func (n *NCRepo) PersistNCConfiguration(ncConfiguration *datamodel.NCConfiguration, addToPendingQueue bool) *api.JsonErr {
	panic("implement me")
}

type ReservationRepo struct {
	Data     map[string]datamodel.ReservationRow
	QueryErr *api.JsonErr
	PutErr   *api.JsonErr
}

func NewReservationRepo() *ReservationRepo {
	return &ReservationRepo{Data: make(map[string]datamodel.ReservationRow)}
}

func (f *ReservationRepo) TryQueryIPReservation(networkID, reservationID string) (datamodel.ReservationRow, *api.JsonErr) {
	if f.QueryErr != nil {
		return datamodel.ReservationRow{}, f.QueryErr
	}
	return f.TryQueryIPReservationByKey(datamodel.GetReservationKey(networkID, reservationID))
}

func (f *ReservationRepo) TryQueryIPReservationByKey(key string) (datamodel.ReservationRow, *api.JsonErr) {
	if f.QueryErr != nil {
		return datamodel.ReservationRow{}, f.QueryErr
	}

	res, ok := f.Data[key]
	if !ok {
		return datamodel.ReservationRow{}, nil
	}

	return res, nil
}

func (f *ReservationRepo) PutIPReservationConfiguration(res *datamodel.ReservationConfiguration) *api.JsonErr {
	if f.PutErr != nil {
		return f.PutErr
	}
	key := datamodel.GetReservationKey(res.NetworkID, res.ReservationID)
	reservationRow := datamodel.ReservationRow{
		RowKey:        key,
		Configuration: res,
		SubnetKey:     datamodel.GetSubnetKey(res.NetworkID, res.AllocationRequest.SubnetName),
	}
	f.Data[key] = reservationRow
	return nil
}

func (f *ReservationRepo) GetReservations(networkID string, reservationIDs []string) ([]datamodel.ReservationRow, *api.JsonErr) {
	var out []datamodel.ReservationRow

	for _, resID := range reservationIDs {
		key := datamodel.GetReservationKey(networkID, resID)
		res, ok := f.Data[key]
		if ok {
			out = append(out, res)
		}
	}

	return out, nil
}

func (f *ReservationRepo) UpdateReservations(reservations []datamodel.ReservationConfiguration) *api.JsonErr {
	for i, reservation := range reservations {
		key := datamodel.GetReservationKey(reservation.NetworkID, reservation.ReservationID)
		reservationRow := datamodel.ReservationRow{
			RowKey:        key,
			Configuration: &reservations[i],
			SubnetKey:     datamodel.GetSubnetKey(reservation.NetworkID, reservation.AllocationRequest.SubnetName),
		}
		f.Data[key] = reservationRow
	}
	return nil
}

func (f *ReservationRepo) DeleteReservation(reservationKey string) *api.JsonErr {
	if _, found := f.Data[reservationKey]; found {
		delete(f.Data, reservationKey)
	}

	return nil
}

type SubnetRepo struct {
	Data     map[string]datamodel.SubnetConfiguration
	QueryErr *api.JsonErr
}

func (f *SubnetRepo) PutSubnetConfiguration(subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return nil
}

func (f *SubnetRepo) TryQuerySubnetConfiguration(networkID, subnetName string) (*datamodel.SubnetConfiguration, *api.JsonErr) {
	if f.QueryErr != nil {
		return nil, f.QueryErr
	}

	key := datamodel.GetSubnetKey(networkID, subnetName)
	sub, ok := f.Data[key]
	if !ok {
		return nil, nil
	}

	return &sub, nil
}

func (f *SubnetRepo) UpdateSubnetReservationSet(reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return nil
}

// note: fake allocation repo is not thread safe
type AllocationRepo struct {
	Data []datamodel.IPAllocationEntity
}

func (f *AllocationRepo) TryQueryIPAllocation(networkID, subnetName, rowKey string) (datamodel.IPAllocationEntity, *api.JsonErr) {
	for _, alloc := range f.Data {
		if alloc.RowKey == rowKey {
			return alloc, nil
		}
	}
	return datamodel.IPAllocationEntity{}, nil
}

func (f *AllocationRepo) UpdateIPAllocationEntities(networkID, subnetName string, entities []datamodel.IPAllocationEntity) *api.JsonErr {
	for _, entity := range entities {
		found := false
		for i := 0; i < len(f.Data); i++ {
			if f.Data[i].RowKey == entity.RowKey {
				found = true
				if f.Data[i].ETag == entity.ETag {
					row := f.Data[i]
					row.ToBeDeletedOwner = entity.ToBeDeletedOwner
					row.IPAllocation = entity.IPAllocation
					row.AllocationID = entity.AllocationID
					row.ETag = entity.ETag

					f.Data[i] = row
					break
				} else {
					return api.InternalServerError(api.StatusInternalServerError,
						fmt.Sprintf("DeleteIPAllocationEntityWithEtag failed, etag not matched, actual %v, expected %v",
							entity, f.Data[i]))
				}
			}
		}

		if !found {
			return api.InternalServerError(api.StatusInternalServerError,
				fmt.Sprintf("DeleteIPAllocationEntityWithEtag failed, entity not found %v",
					entity))
		}
	}

	return nil
}

func (f *AllocationRepo) QueryIPAllocationsByAllocationID(_, _ string, allocIDs []string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	ids := make(map[string]struct{})
	for _, allocID := range allocIDs {
		ids[allocID] = struct{}{}
	}

	var results []datamodel.IPAllocationEntity
	for _, d := range f.Data {
		if _, found := ids[d.AllocationID]; found {
			results = append(results, d)
		}
	}
	return results, nil
}

func (f *AllocationRepo) QueryIPAllocationsByReservationKey(networkID, subnetName string, reservationKey string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	var results []datamodel.IPAllocationEntity
	for _, d := range f.Data {
		if d.IPAllocation == reservationKey {
			results = append(results, d)
		}
	}
	return results, nil
}

func (f *AllocationRepo) QueryIPAllocationsByToBeDeletedOwner(_, _ string, tobeDeletedOwnerId string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	var results []datamodel.IPAllocationEntity
	for _, d := range f.Data {
		if d.ToBeDeletedOwner == tobeDeletedOwnerId {
			results = append(results, d)
		}
	}
	return results, nil
}

func (f *AllocationRepo) InsertIPAllocationEntities(networkID, subnetName string, entities []datamodel.IPAllocationEntity) *api.JsonErr {
	for _, entity := range entities {
		entity.ETag = time.Now().UTC().String()
		f.Data = append(f.Data, entity)
	}
	return nil
}

func (f *AllocationRepo) AddIPAllocations(_, _ string, rows map[string]map[string]interface{}) *api.JsonErr {
	for rowKey, rowData := range rows {
		allocID, _ := rowData[datamodel.AllocationIDProperty].(string)
		ipAlloc, _ := rowData[datamodel.IPAllocationProperty].(string)

		row := datamodel.IPAllocationEntity{
			RowKey:       rowKey,
			AllocationID: allocID,
			IPAllocation: ipAlloc,
			ETag:         time.Now().UTC().String(),
		}

		f.Data = append(f.Data, row)
	}
	return nil
}

func (f *AllocationRepo) TryQueryAllIPAllocations(networkID, subnetName string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	return f.Data, nil
}

func (f *AllocationRepo) DeleteIPAllocations(_, _ string, keys []string) *api.JsonErr {
	tbd := make(map[string]struct{})
	for _, key := range keys {
		if key == "" {
			return api.InternalServerError(api.StatusInternalServerError, "DeleteIPAllocation failed, key is empty")
		}
		tbd[key] = struct{}{}
	}

	for i := 0; i < len(f.Data); i++ {
		if _, found := tbd[f.Data[i].RowKey]; found {
			f.Data = append(f.Data[:i], f.Data[i+1:]...)
			i--
		}
	}

	return nil
}

func (f *AllocationRepo) DeleteIPAllocation(networkID, subnetName, allocatedIPKey string) *api.JsonErr {
	for i := 0; i < len(f.Data); i++ {
		if f.Data[i].RowKey == allocatedIPKey {
			f.Data = append(f.Data[:i], f.Data[i+1:]...)
			i--
		}
	}

	return nil
}

func (f *AllocationRepo) DeleteIPAllocationEntityWithEtag(networkID, subnetName string, entity datamodel.IPAllocationEntity) *api.JsonErr {
	found := false
	for i := 0; i < len(f.Data); i++ {
		if f.Data[i].RowKey == entity.RowKey {
			found = true
			if f.Data[i].ETag == entity.ETag {
				f.Data = append(f.Data[:i], f.Data[i+1:]...)
				i--
			} else {
				return api.InternalServerError(api.StatusInternalServerError,
					fmt.Sprintf("DeleteIPAllocationEntityWithEtag failed, etag not matched, actual %v, expected %v",
						entity, f.Data[i]))
			}
		}
	}

	if !found {
		return api.InternalServerError(api.StatusInternalServerError,
			fmt.Sprintf("DeleteIPAllocationEntityWithEtag failed, entity not found %v",
				entity))
	}

	return nil
}

func (f *AllocationRepo) QueryIPAllocationsByReservationKeys(networkID, subnetName string, reservationIDs []string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	reservationKeys := make(map[string]struct{})
	for _, reservationID := range reservationIDs {
		resKey := datamodel.GetReservationKey(networkID, reservationID)
		reservationKeys[resKey] = struct{}{}
	}

	var results []datamodel.IPAllocationEntity
	for _, d := range f.Data {
		if _, found := reservationKeys[d.IPAllocation]; found {
			results = append(results, d)
		}
	}
	return results, nil
}

func (f *AllocationRepo) QueryUnreservedAllocations(networkID, subnetName string, count int) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	var results []datamodel.IPAllocationEntity

	for _, d := range f.Data {
		if d.IPAllocation == "" {
			results = append(results, d)
		}

		if len(results) == count {
			break
		}
	}

	return results, nil
}

type NodeRepo struct {
	FakeNodeConfig datamodel.NodeConfiguration
}

func (n *NodeRepo) TryQueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	return &n.FakeNodeConfig, nil
}

func (n *NodeRepo) QueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	return &n.FakeNodeConfig, nil
}

func (n *NodeRepo) UpdateNodeConstrainedReservationSet(networkID, reservationID string, remove bool, nodeConfiguration *datamodel.NodeConfiguration) *api.JsonErr {
	// TODO: implement to testing.
	return nil
}

func (n *NodeRepo) UpdateNodeConstrainedReservationSetByNodeID(nodeID, networkID, reservationID string, remove bool) *api.JsonErr {
	// TODO: implement to testing.
	return nil
}

func (n *NodeRepo) UpdateNodeContainers(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr {
	// TODO: implement for testing.
	return nil
}
