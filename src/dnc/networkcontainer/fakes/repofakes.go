package fakes

import (
	"dnc/api"
	"dnc/datamodel"
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
	Data     map[string]datamodel.ReservationConfiguration
	QueryErr *api.JsonErr
	PutErr   *api.JsonErr
}

func NewReservationRepo() *ReservationRepo {
	return &ReservationRepo{Data: make(map[string]datamodel.ReservationConfiguration)}
}

func (f *ReservationRepo) TryQueryIPReservation(networkID, reservationID string) (*datamodel.ReservationConfiguration, *api.JsonErr) {
	if f.QueryErr != nil {
		return nil, f.QueryErr
	}
	return f.TryQueryIPReservationByKey(datamodel.GetReservationKey(networkID, reservationID))
}

func (f *ReservationRepo) TryQueryIPReservationByKey(key string) (*datamodel.ReservationConfiguration, *api.JsonErr) {
	if f.QueryErr != nil {
		return nil, f.QueryErr
	}

	res, ok := f.Data[key]
	if !ok {
		return nil, nil
	}

	return &res, nil
}

func (f *ReservationRepo) PutIPReservationConfiguration(res *datamodel.ReservationConfiguration) *api.JsonErr {
	if f.PutErr != nil {
		return f.PutErr
	}
	key := datamodel.GetReservationKey(res.NetworkID, res.ReservationID)
	f.Data[key] = *res
	return nil
}

func (f *ReservationRepo) GetReservations(networkID string, reservationIDs []string) ([]datamodel.ReservationConfiguration, *api.JsonErr) {
	var out []datamodel.ReservationConfiguration

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
	for _, reservation := range reservations {
		key := datamodel.GetReservationKey(reservation.NetworkID, reservation.ReservationID)
		f.Data[key] = reservation
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

func (f *SubnetRepo) UpdateSubnetReservationSet(networkID, reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return nil
}

type RepoKV struct {
	Key   string
	Value interface{}
}

// todo: refactor to just use a map for data. fake pagination can be done via sorting and using key as page token
type AllocationRepo struct {
	Data        []RepoKV
	MaxPageSize int
}

func (f *AllocationRepo) QueryIPAllocationsByAllocationID(_, _ string, allocIDs []string) (map[string]interface{}, *api.JsonErr) {
	ids := make(map[string]struct{})
	for _, allocID := range allocIDs {
		ids[allocID] = struct{}{}
	}

	results := make(map[string]interface{})
	for _, d := range f.Data {
		row, ok := d.Value.(map[string]interface{})
		if !ok {
			continue
		}
		val := row["AllocationID"]
		allocID, ok := val.(string)
		if !ok {
			continue
		}
		if _, found := ids[allocID]; found {
			results[d.Key] = d.Value
		}
	}
	return results, nil
}

func (f *AllocationRepo) AddIPAllocations(_, _ string, rows map[string]map[string]interface{}) *api.JsonErr {
	for rowKey, rowData := range rows {
		f.Data = append(f.Data, RepoKV{Key: rowKey, Value: rowData})
	}
	return nil
}

func (f *AllocationRepo) TryQueryAllIPAllocations(_, _ string, pageToken interface{}) (map[string]interface{}, bool, interface{}, *api.JsonErr) {
	index := 0
	if pageToken != nil {
		idx, ok := pageToken.(int)
		if !ok {
			return nil, false, nil, api.InternalServerError(0, "wrong token type")
		}
		index = idx
	}
	if index > len(f.Data)-1 {
		return nil, false, nil, nil
	}

	out := make(map[string]interface{})
	for ; index < len(f.Data) && len(out) < f.MaxPageSize; index++ {
		out[f.Data[index].Key] = f.Data[index].Value
	}
	var nextPageToken interface{}
	if index < len(f.Data) {
		nextPageToken = index
	}

	return out, nextPageToken != nil, nextPageToken, nil
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
		if _, found := tbd[f.Data[i].Key]; found {
			f.Data = append(f.Data[:i], f.Data[i+1:]...)
			i--
		}
	}

	return nil
}

type NodeRepo struct {
	FakeNodeConfig datamodel.NodeConfiguration
}

func (n *NodeRepo) QueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	return &n.FakeNodeConfig, nil
}

func (n *NodeRepo) UpdateNodeConstrainedReservationSetByNodeID(nodeID, networkID, reservationID string, remove bool) *api.JsonErr {
	// TODO: implement to testing.
	return nil
}

func (n *NodeRepo) UpdateNodeContainers(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr {
	// TODO: implement for testing.
	return nil
}
