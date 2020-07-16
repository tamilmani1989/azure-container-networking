package fakes

import (
	"dnc/api"
	"dnc/datamodel"
)

type ReservationRepo struct {
	Payload datamodel.ReservationConfiguration
}

func (f *ReservationRepo) TryQueryIPReservation(networkID, reservationID string) (*datamodel.ReservationConfiguration, *api.JsonErr) {
	return &f.Payload, nil
}

func (f *ReservationRepo) TryQueryIPReservationByKey(key string) (*datamodel.ReservationConfiguration, *api.JsonErr) {
	return &f.Payload, nil
}

func (f *ReservationRepo) PutIPReservationConfiguration(res *datamodel.ReservationConfiguration) *api.JsonErr {
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

type AllocationRepo struct {
	Data []struct {
		Key   string
		Value interface{}
	}
	MaxPageSize int
}

func (f *AllocationRepo) AddIPAllocations(_, _ string, _ map[string]map[string]interface{}) *api.JsonErr {
	return nil
}

func (f *AllocationRepo) TryQueryAllIPAllocations(_, _ string, pageToken interface{}) (map[string]interface{}, bool, interface{}, *api.JsonErr) {
	index := 0
	if pageToken != nil {
		index = pageToken.(int)
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

func (f *AllocationRepo) AddIPAllocation(_, _, _, _ string) *api.JsonErr { return nil }
