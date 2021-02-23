package datamodel

import (
	"dnc/api"
	"dnc/datastore/azurestorage"
	"dnc/datastoreshim"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	IPAllocationsTableName   = "IPAllocationsTable"
	IPAllocationProperty     = "IPAllocation"
	AllocationIDProperty     = "AllocationID"
	ToBeDeletedOwnerProperty = "ToBeDeletedOwner"
)

// todo: the 3 properties that any row will have are partition key, row key, and etag.
//  we should consider having a separate metadata struct that can be embedded within any struct representing a row.
// todo: Should store PartitionKey as well
// An IPAllocationEntity holds the data for a record in the IPAllocationsTable, and represents an IP address within a subnet.
type IPAllocationEntity struct {
	// RowKey is a unique key within a partition. Encoded as "10.0.0.1_32".
	RowKey string
	// ETag is a metadata property used for optimistic locking.
	ETag string
	// IPAllocation is a property which contains the row key for a corresponding reservation in IPReservationsTable.
	IPAllocation string
	// AllocationID is a property which associates the allocation record to either a primary or secondary IP address in a reservation.
	// If the allocation is for a primary IP, this UUID will be the same as the Reservation ID.
	// If the allocation is for a secondary IP, this UUID will be contained within the secondary allocation keys in the reservation.
	AllocationID string
	// ToBeDeletedOwner is a property which is set during deletion of an NC, and contains the NC ID that owns the allocation.
	ToBeDeletedOwner string
}

// EncodeIPAllocationKey encodes the primary key for IP allocations table
func EncodeIPAllocationKey(allocatedSubnet string) string {
	return strings.Replace(allocatedSubnet, "/", "_", 1)
}

// DecodeIPAllocationKey decodes the primary key for IP allocations table
func DecodeIPAllocationKey(allocatedSubnet string) string {
	return strings.Replace(allocatedSubnet, "_", "/", 1)
}

func StorageEntityToIPAllocationEntity(e *storage.Entity) IPAllocationEntity {
	return IPAllocationEntity{
		RowKey:           e.RowKey,
		ETag:             e.OdataEtag,
		IPAllocation:     ExtractStringProperty(e, IPAllocationProperty),
		AllocationID:     ExtractStringProperty(e, AllocationIDProperty),
		ToBeDeletedOwner: ExtractStringProperty(e, ToBeDeletedOwnerProperty),
	}
}

// todo: we should consider keeping the partition in the row struct for completeness
//  since it would better encapsulate the record and its metadata
func (r *IPAllocationEntity) ToStorageEntity(networkID, subnetName string) storage.Entity {
	return storage.Entity{
		Table:        &storage.Table{Name: IPAllocationsTableName},
		PartitionKey: GetSubnetKey(networkID, subnetName),
		RowKey:       r.RowKey,
		OdataEtag:    r.ETag,
		Properties: map[string]interface{}{
			IPAllocationProperty:     r.IPAllocation,
			AllocationIDProperty:     r.AllocationID,
			ToBeDeletedOwnerProperty: r.ToBeDeletedOwner,
		},
	}
}

// AllocationRepo performs CRUD operations on the IPAllocationsTable
type AllocationRepo struct {
	Datastore datastoreshim.DataStoreShim
}

func (a *AllocationRepo) TryQueryIPAllocation(networkID, subnetName, rowKey string) (IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)

	entity, status, err := a.Datastore.QueryInPartition(IPAllocationsTableName, partitionKey, rowKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf("Failed to retrieve IP allocation for key:[%s] in partition:[%s] due to error:[%s]", rowKey, partitionKey, err.Error())
		return IPAllocationEntity{}, api.InternalServerError(api.StatusInternalServerError, msg)
	}

	if entity == nil {
		return IPAllocationEntity{}, nil
	}

	return StorageEntityToIPAllocationEntity(entity), nil
}

func (a *AllocationRepo) TryQueryAllIPAllocations(networkID, subnetName string) ([]IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)

	var (
		rows      []IPAllocationEntity
		pageToken interface{}
	)

	for {
		entities, hasNextPage, nextPageToken, _, err := a.Datastore.QueryAllRowsInPartition(IPAllocationsTableName, partitionKey, pageToken)
		if err != nil {
			return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		for _, entity := range entities {
			rows = append(rows, StorageEntityToIPAllocationEntity(entity))
		}

		if !hasNextPage {
			break
		}

		pageToken = nextPageToken
	}

	return rows, nil
}

func (a *AllocationRepo) QueryIPAllocationsByReservationKey(networkID, subnetName string, reservationKey string) ([]IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	if reservationKey == "" {
		errMsg := fmt.Sprintf("QueryIPAllocationsByReservationKey: "+
			"reservationId cant be null for partitionKey %s", partitionKey)
		return nil, api.BadRequest(api.InvalidOrEmptyToBeDeletedOwner, errMsg)
	}

	reservationIds := []string{reservationKey}
	filter := azurestorage.BuildFilterOnProperty(partitionKey, IPAllocationProperty, reservationIds)

	entities, err := a.Datastore.QueryFilter(IPAllocationsTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	rows := make([]IPAllocationEntity, len(entities))
	for i, entity := range entities {
		rows[i] = StorageEntityToIPAllocationEntity(entity)
	}

	return rows, nil
}

func (a *AllocationRepo) QueryIPAllocationsByAllocationID(networkID, subnetName string, allocationIDs []string) ([]IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	filter := azurestorage.BuildFilterOnProperty(partitionKey, AllocationIDProperty, allocationIDs)

	entities, err := a.Datastore.QueryFilter(IPAllocationsTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	rows := make([]IPAllocationEntity, len(entities))
	for i, entity := range entities {
		rows[i] = StorageEntityToIPAllocationEntity(entity)
	}

	return rows, nil
}

// Function to fetch all rows By PartitionKey and ToBeDeletedOwnerId property
func (a *AllocationRepo) QueryIPAllocationsByToBeDeletedOwner(networkID, subnetName string, tobeDeletedOwnerId string) ([]IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	if tobeDeletedOwnerId == "" {
		errMsg := fmt.Sprintf("QueryIPAllocationsByToBeDeletedOwner: "+
			"tobeDeletedOwnerId cant be null for partitionKey %s", partitionKey)
		return nil, api.BadRequest(api.InvalidOrEmptyToBeDeletedOwner, errMsg)
	}

	tobeDeleteOwnerKey := []string{tobeDeletedOwnerId}
	filter := azurestorage.BuildFilterOnProperty(partitionKey, ToBeDeletedOwnerProperty, tobeDeleteOwnerKey)

	entities, err := a.Datastore.QueryFilter(IPAllocationsTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	rows := make([]IPAllocationEntity, len(entities))
	for i, entity := range entities {
		rows[i] = StorageEntityToIPAllocationEntity(entity)
	}

	return rows, nil
}

// deprecated: use InsertIPAllocationEntities
func (a *AllocationRepo) AddIPAllocations(networkID, subnetName string, rows map[string]map[string]interface{}) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)

	if err := a.Datastore.InsertBatch(IPAllocationsTableName, partitionKey, rows); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if !ok {
			return api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		switch storageErr.StatusCode {
		case http.StatusConflict:
			return api.Conflict(api.AllocationFailed, storageErr.Message)
		default:
			return api.InternalServerError(api.StatusInternalServerError, storageErr.Message)
		}
	}

	return nil
}

func (a *AllocationRepo) InsertIPAllocationEntities(networkID, subnetName string, entities []IPAllocationEntity) *api.JsonErr {
	storageEntities := make([]storage.Entity, len(entities))
	for i, entity := range entities {
		se := entity.ToStorageEntity(networkID, subnetName)
		storageEntities[i] = se
	}

	if err := a.Datastore.InsertEntityBatch(IPAllocationsTableName, storageEntities); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if !ok {
			return api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		switch storageErr.StatusCode {
		case http.StatusConflict:
			return api.Conflict(api.AllocationFailed, storageErr.Message)
		default:
			return api.InternalServerError(api.StatusInternalServerError, storageErr.Message)
		}
	}

	return nil
}

func (a *AllocationRepo) UpdateIPAllocationEntities(networkID, subnetName string, ipAllocationEntities []IPAllocationEntity) *api.JsonErr {
	storageEntities := make([]storage.Entity, len(ipAllocationEntities))
	for i, entity := range ipAllocationEntities {
		se := entity.ToStorageEntity(networkID, subnetName)
		storageEntities[i] = se
	}

	if err := a.Datastore.UpdateEntityBatch(IPAllocationsTableName, storageEntities); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if !ok {
			msgFmt := "could not update batch for vnet id %s subnet %s: %s"
			return api.InternalServerErrorf(api.StatusInternalServerError, msgFmt, networkID, subnetName, err.Error())
		}

		switch storageErr.StatusCode {
		case http.StatusPreconditionFailed:
			msgFmt := "etag conflict when updating batch for vnet id %s subnet %s: %s"
			return api.PreconditionFailedf(api.UpdateIPAllocationFailed, msgFmt, networkID, subnetName, storageErr.Message)
		default:
			msgFmt := "could not update batch for vnet id %s subnet %s: %s"
			return api.InternalServerErrorf(api.StatusInternalServerError, msgFmt, networkID, subnetName, storageErr.Message)
		}
	}

	return nil
}

func (a *AllocationRepo) DeleteIPAllocation(networkID, subnetName, allocatedIPKey string) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)
	if err := a.Datastore.DeleteInPartition(IPAllocationsTableName, partitionKey, allocatedIPKey); err != nil {
		msg := fmt.Sprintf("Cannot delete entry %s from IPAllocations table due to error: %+v", allocatedIPKey, err)
		return api.InternalServerError(api.StatusInternalServerError, msg)
	}

	return nil
}

func (a *AllocationRepo) DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)

	if err := a.Datastore.DeleteBatch(IPAllocationsTableName, partitionKey, keys); err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	return nil
}

// Deletes the entity if etag matches
func (a *AllocationRepo) DeleteIPAllocationEntityWithEtag(networkID, subnetName string, entity IPAllocationEntity) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)
	err := a.Datastore.DeleteInPartitionWithEtag(IPAllocationsTableName, partitionKey, entity.RowKey, entity.ETag)

	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to delete request associated with key %s with etag %s in table %s due to error:{%s}",
			entity.RowKey, entity.ETag, IPAllocationsTableName, err.Error())

	}

	return nil
}

// todo: convert ipcleanup to a struct that can marshall/unmarshall itself to a string
// SubnetKey parameter looks like: "Network_%s_Subnet_%s"
func GetSubnetFromIPAllocationsPartitionKey(key string) (string, *api.JsonErr) {
	entries := strings.Split(key, "_")
	return entries[3], nil
}

// todo: convert ipcleanup to a struct that can marshall/unmarshall itself to a string
// SubnetKey parameter looks like: "Network_%s_Subnet_%s"
func GetNetworkIdFromIPAllocationsPartitionKey(key string) (string, *api.JsonErr) {
	entries := strings.Split(key, "_")
	return entries[1], nil
}

func (a *AllocationRepo) QueryUnreservedAllocations(networkID, subnetName string, count int) ([]IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	filter := azurestorage.BuildFilterOnProperty(partitionKey, IPAllocationProperty, []string{""})

	entities, err := a.Datastore.QueryFilterTopN(IPAllocationsTableName, filter, count)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	var rows []IPAllocationEntity
	for _, entity := range entities {
		row := StorageEntityToIPAllocationEntity(entity)
		rows = append(rows, row)
	}

	return rows, nil
}

func (a *AllocationRepo) QueryIPAllocationsByReservationKeys(networkID, subnetName string, reservationKeys []string) ([]IPAllocationEntity, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	filter := azurestorage.BuildFilterOnProperty(partitionKey, IPAllocationProperty, reservationKeys)

	entities, err := a.Datastore.QueryFilter(IPAllocationsTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}
	var rows []IPAllocationEntity
	for _, entity := range entities {
		row := StorageEntityToIPAllocationEntity(entity)
		rows = append(rows, row)
	}

	return rows, nil
}
