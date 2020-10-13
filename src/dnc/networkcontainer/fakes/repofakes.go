package fakes

import (
	"dnc/api"
	"dnc/datamodel"
)

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
	res := f.Data[key]
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

type SubnetRepo struct {
	Payload datamodel.SubnetConfiguration
}

func (f *SubnetRepo) TryQuerySubnetConfiguration(networkID, subnetName string) (*datamodel.SubnetConfiguration, *api.JsonErr) {
	return &f.Payload, nil
}

func (f *SubnetRepo) UpdateSubnetReservationSet(networkID, reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return nil
}

type RepoKV struct {
	Key   string
	Value interface{}
}

type AllocationRepo struct {
	Data        []RepoKV
	MaxPageSize int
}

func (f *AllocationRepo) AddIPAllocations(_, _ string, _ map[string]map[string]interface{}) *api.JsonErr {
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
