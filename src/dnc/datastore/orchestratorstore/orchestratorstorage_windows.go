package orchestratorstorage

import (
	"dnc/configuration"
	"dnc/utils"
	"encoding/json"
	"errors"
	"net/http"
	"syscall"
	"unsafe"
)

// SwiftErrorCodes :- Return codes from SwiftPersistenceDllClr.dll
const (
	SwiftSuccess = iota
	SwiftGenericFailure
	SwiftNoMoreRecords
	SwiftInsufficientBuffer
	SwiftKeyNotFound
	SwiftInvalidParameter
)

// SwiftEntityType :- Entity types from SwiftPersistenceDllClr.dll
const (
	SwiftVirtualNetwork = iota
	SwiftSubnet
	SwiftReservation
	SwiftNode
	SwiftSTTag
	SwiftNetworkContainer
	SwiftPendingItems
	SwiftAllEntityTypes = 0xffff
)

// OrchestratorStorage :- Orchestrator's storage object to manage tables in partition
type OrchestratorStorage struct {
	configuration.OrchestratorStorageInfo
	procInsertOrUpdate uintptr
	procQuery          uintptr
	procQueryAll       uintptr
	procKeyExists      uintptr
	procDeleteRow      uintptr
	procEventLog       uintptr
	dll                syscall.Handle
}

// Init :- Initialize orchestrator storage
func (orchestratorstorage *OrchestratorStorage) Init(tableToColumnMap map[string]string) {
	var err error

	// Need to release dll somewhere...defer syscall.FreeLibrary(dll)
	if orchestratorstorage.dll, err = syscall.LoadLibrary(orchestratorstorage.OrchestratorStorageInfo.Dll); err != nil {
		utils.OnErrorFail(err, "Failed to load dll")
	}

	if orchestratorstorage.procInsertOrUpdate, err = syscall.GetProcAddress(orchestratorstorage.dll, "InsertOrUpdate"); err != nil {
		utils.OnErrorFail(err, "Failed to find InsertOrUpdate function")
	}

	if orchestratorstorage.procQuery, err = syscall.GetProcAddress(orchestratorstorage.dll, "Query"); err != nil {
		utils.OnErrorFail(err, "Failed to find Query function")
	}

	if orchestratorstorage.procQueryAll, err = syscall.GetProcAddress(orchestratorstorage.dll, "QueryAll"); err != nil {
		utils.OnErrorFail(err, "Failed to find QueryAll function")
	}

	if orchestratorstorage.procKeyExists, err = syscall.GetProcAddress(orchestratorstorage.dll, "KeyExists"); err != nil {
		utils.OnErrorFail(err, "Failed to find KeyExists function")
	}

	if orchestratorstorage.procDeleteRow, err = syscall.GetProcAddress(orchestratorstorage.dll, "DeleteRow"); err != nil {
		utils.OnErrorFail(err, "Failed to find DeleteRow function")
	}

	// Logger provided along with dll here, but perhaps not always the case??
	if orchestratorstorage.procEventLog, err = syscall.GetProcAddress(orchestratorstorage.dll, "EventLog"); err != nil {
		utils.OnErrorFail(err, "Failed to find EventLog function")
	}
}

// InsertOrUpdate :- Insert or update an element using rowKey and tablName.
func (orchestratorstorage *OrchestratorStorage) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	ret, _, _ := syscall.Syscall(uintptr(orchestratorstorage.procInsertOrUpdate), uintptr(3), getEntityEnum(tableName), convertStrToPass(rowKey), convertStrToPass(value.(string)))
	return parseSwiftRet(ret)
}

// Delete :- Remove an element using rowKey and tableName.
func (orchestratorstorage *OrchestratorStorage) Delete(tableName, rowKey string) error {
	ret, _, _ := syscall.Syscall(uintptr(orchestratorstorage.procDeleteRow), uintptr(2), getEntityEnum(tableName), convertStrToPass(rowKey), 0)
	return parseSwiftRet(ret)
}

// Query :- Query an element using rowKey and tableName.
func (orchestratorstorage *OrchestratorStorage) Query(tableName, rowKey string) (interface{}, int, error) {
	var (
		status    = http.StatusNotFound
		buffer    = make([]uint16, 1024)
		cchBuffer = cap(buffer)
	)

	ret, _, _ := syscall.Syscall6(uintptr(orchestratorstorage.procQuery), uintptr(4), getEntityEnum(tableName), convertStrToPass(rowKey), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&cchBuffer)), 0, 0)

	if ret == SwiftSuccess {
		status = http.StatusOK
	}

	return syscall.UTF16ToString(buffer), status, parseSwiftRet(ret)
}

// QueryAll - Query a row in a given table.
func (orchestratorstorage *OrchestratorStorage) QueryAll(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	var (
		status    = http.StatusNotFound
		buffer    = make([]uint16, 1024)
		cchBuffer = cap(buffer)
		next      = context.(int)
		data      interface{}
		err       error
	)

	if ret, _, _ := syscall.Syscall6(uintptr(orchestratorstorage.procQueryAll), uintptr(5), getEntityEnum(tableName), uintptr(next), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&cchBuffer)), uintptr(unsafe.Pointer(&next)), 0); ret == SwiftSuccess {
		err = json.Unmarshal([]byte(syscall.UTF16ToString(buffer)), &data)
		status = http.StatusOK
	} else {
		err = parseSwiftRet(ret)
	}

	return data.(map[string]interface{}), false, next, status, err
}

// KeyExist :- Check if key exists in a given table.
func (orchestratorstorage *OrchestratorStorage) KeyExist(tableName, rowKey string) (bool, error) {
	ret, _, _ := syscall.Syscall(uintptr(orchestratorstorage.procDeleteRow), uintptr(2), getEntityEnum(tableName), convertStrToPass(rowKey), 0)
	return ret == SwiftSuccess, parseSwiftRet(ret)
}

func convertStrToPass(s string) uintptr {
	return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s)))
}

func getEntityEnum(entity string) uintptr {
	var ret uintptr

	switch entity {
	case "SubnetConfigurationTable":
		ret = uintptr(SwiftSubnet)
	case "NodeConfigurationTable":
		ret = uintptr(SwiftNode)
	case "IPReservationsTable":
		ret = uintptr(SwiftReservation)
	case "NetworkContainerTable":
		ret = uintptr(SwiftNetworkContainer)
	case "NetworkConfigurationTable":
		ret = uintptr(SwiftVirtualNetwork)
	case "PendingWorkItemsTable":
		ret = uintptr(SwiftPendingItems)
	case "AllTables": // will need to add this functionality in the future, currently no db dump
		ret = uintptr(SwiftAllEntityTypes)
	}

	return ret
}

func parseSwiftRet(ret uintptr) error {
	switch int(ret) {
	case SwiftGenericFailure:
		return errors.New("SwiftGenericFailure")
	case SwiftNoMoreRecords:
		return errors.New("SwiftNoMoreRecords")
	case SwiftInsufficientBuffer:
		return errors.New("SwiftInsufficientBuffer")
	case SwiftKeyNotFound:
		return errors.New("SwiftKeyNotFound")
	case SwiftInvalidParameter:
		return errors.New("SwiftInvalidParameter")
	}
	// Remaining SwiftErrorCodes -> SwiftSuccess
	return nil
}
