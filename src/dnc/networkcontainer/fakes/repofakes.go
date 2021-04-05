package fakes

import (
	"dnc/api"
	"dnc/datamodel"
	"net/http"
	"fmt"
	"time"
	"sync"
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
	Data         []datamodel.IPAllocationEntity
	MaxBatchSize int
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
	if len(entities) > f.MaxBatchSize {
		return api.InternalServerError(api.StatusInternalServerError,
			fmt.Sprintf("UpdateIPAllocationEntities cannnot not exceed maxBatch size networkID %s, subnetName %s",
				networkID, subnetName))
	}

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

func (f *AllocationRepo) QueryIPAllocationsByToBeDeletedOwner(partitionKey string, tobeDeletedOwnerId string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	var results []datamodel.IPAllocationEntity
	for _, d := range f.Data {
		if d.ToBeDeletedOwner == tobeDeletedOwnerId {
			results = append(results, d)
		}
	}
	return results, nil
}

func (f *AllocationRepo) InsertIPAllocationEntities(networkID, subnetName string, entities []datamodel.IPAllocationEntity) *api.JsonErr {
	if len(entities) > f.MaxBatchSize {
		return api.InternalServerError(api.StatusInternalServerError,
			fmt.Sprintf("InsertIPAllocationEntities cannnot not exceed maxBatch size networkID %s, subnetName %s",
				networkID, subnetName))
	}

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

//TODO: Remove fake repos by accepting the DataStoreShim interface as a dependency in the Repo ( See AllocationRepo )
type NetworkRepo struct {
	Data map[string]datamodel.NetworkConfiguration
}

func NewNetworkRepo() *NetworkRepo {
	return &NetworkRepo{Data: make(map[string]datamodel.NetworkConfiguration)}
}

func (f *NetworkRepo) TryQueryNetworkConfiguration(networkID string) (*datamodel.NetworkConfiguration, *api.JsonErr) {
	networkConfig, found := f.Data[datamodel.GetNetworkKey(networkID)]
	if !found {
		return nil, nil
	}

	return &networkConfig, nil
}

func (f *NetworkRepo) GetAllNetworkIDs() (map[string]bool, error) {
	networkIDs := make(map[string]bool)
	for networkKey, _ := range f.Data {
		networkIDs[networkKey] = true
	}
	return networkIDs, nil
}

func (f *NetworkRepo) TryQueryAllNetworks() ([]string, error) {
	networks := []string{}
	for _, network := range f.Data {
		networks = append(networks, network.NetworkID)
	}
	return networks, nil
}

func (f *NetworkRepo) PersistNetworkConfiguration(networkConfig *datamodel.NetworkConfiguration) *api.JsonErr {
	f.Data[datamodel.GetNetworkKey(networkConfig.NetworkID)] = *networkConfig
	return nil
}

type NcPendingPublishRequestContextRepo struct {
	// Data is a sync map to allow concurrent read, writes
	Data sync.Map // This holds map[string]datamodel.NcPendingPublishRequestContext
}

func NewNcPendingPublishRequestContextRepo() *NcPendingPublishRequestContextRepo {
	return &NcPendingPublishRequestContextRepo{}
}

func (f *NcPendingPublishRequestContextRepo) GetNcPendingPublishRequestContexts() (map[string]datamodel.NcPendingPublishRequestContext, *api.JsonErr) {
	m := make(map[string]datamodel.NcPendingPublishRequestContext)
	f.Data.Range(func(key interface{}, value interface{}) bool {
		m[key.(string)] = value.(datamodel.NcPendingPublishRequestContext)
		return true
	})

	return m, nil
}

func (f *NcPendingPublishRequestContextRepo) RemoveNcPendingPublishContext(networkID, networkContainerID, pubsubAzID string) *api.JsonErr {
	key := datamodel.GetNcPendingPublishContextKey(networkID, networkContainerID, pubsubAzID)
	f.Data.Delete(key)
	return nil
}

func (f *NcPendingPublishRequestContextRepo) CreateOrUpdateNcPendingPublishRequest(requestContext *datamodel.NcPendingPublishRequestContext) *api.JsonErr {
	key := datamodel.GetNcPendingPublishContextKey(requestContext.NetworkID, requestContext.NetworkContainerID, requestContext.PubsubAzId)
	f.Data.Store(key, *requestContext)
	return nil
}

func (f *NcPendingPublishRequestContextRepo) QueryNcPendingPublishContext(key string) (*datamodel.NcPendingPublishRequestContext, *api.JsonErr) {
	requestContextValue, found := f.Data.Load(key)
	if !found {
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       "data not found",
			HTTPStatus: http.StatusInternalServerError}
	}
	requestContext := requestContextValue.(datamodel.NcPendingPublishRequestContext)
	return &requestContext, nil
}


func (f *AllocationRepo) DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr {
	if len(keys) > f.MaxBatchSize {
		return api.InternalServerError(api.StatusInternalServerError,
			fmt.Sprintf("InsertIPAllocationEntities cannnot not exceed maxBatch size networkID %s, subnetName %s",
				networkID, subnetName))
	}

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

func (f *AllocationRepo) DeleteIPAllocationEntityWithEtag(partitionKey string, entity datamodel.IPAllocationEntity) *api.JsonErr {
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

func (n *NodeRepo) CreateOrUpdateNodeConfiguration(
	register bool,
	cores int,
	nodeID string,
	nodeInfo api.NodeInfoRequest,
	apiVersion string,
	nmAgentSupportedApis []string) *api.JsonErr {
	// TODO: implement for testing.
	return nil
}

func (n *NodeRepo) PersistNodeConfiguration(nodeconfiguration *datamodel.NodeConfiguration) *api.JsonErr {
	// TODO: implement for testing.
	return nil
}

func (n *NodeRepo) DeleteNode(nodeID string) *api.JsonErr {
	// TODO: implement for testing.
	return nil
}

func (n *NodeRepo) TryQueryNodeConfigurationByKey(nodeKey string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	// TODO: implement for testing.
	return nil, nil
}

func (n *NodeRepo) GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	// TODO: implement for testing.
	return nil, nil
}

func (n *NodeRepo) UpdateNodeConfigurationNetworkContainerSet(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr {
	// TODO: implement for testing.
	return nil
}
