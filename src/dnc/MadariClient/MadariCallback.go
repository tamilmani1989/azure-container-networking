package MadariClient

/*
#include <stdint.h>

struct MADARI_HEALTH;
struct MADARI_APPLICATIONHEALTH;
struct MADARI_NOTIFY;
struct MADARI_RESULT;
*/
import "C"
import "fmt"
import "sync"
import "unsafe"

type MadariTraceLevel int

const (
    MADARI_TRACE_ERROR MadariTraceLevel = iota + 1
    MADARI_TRACE_WARNING
    MADARI_TRACE_VERBOSE
    MADARI_TRACE_PAINFUL
)

type MadariErrorSrc int

const (
    MADARIERRORSRC_UNKNOWN MadariErrorSrc = iota
    MADARIERRORSRC_FRONTENDSERVICE
    MADARIERRORSRC_MADARISERVICE
    MADARIERRORSRC_SUBSCRIBER
)

type MadariServiceType int

const (
    MADARI_SERVICETYPE_UNKNOWN MadariServiceType = iota
    MADARI_SERVICETYPE_HEALTHMONITOR
    MADARI_SERVICETYPE_FRONTEND
    MADARI_SERVICETYPE_NOTIFICATIONSERVICE
    MADARI_SERVICETYPE_SELECTORSERVICE
    MADARI_SERVICETYPE_MADARISERVICE
)

type MadariNotifyType int

const (
    MADARI_NOTIFY_NODE_INITIAL_STATE MadariNotifyType = iota + 1
    MADARI_NOTIFY_NODE_CHILDREN_CHANGED
    MADARI_NOTIFY_NODE_CREATED
    MADARI_NOTIFY_NODE_DATA_CHANGED
    MADARI_NOTIFY_NODE_DELETED
)

type MadariResultType int

const (
    MADARI_RESULT_TYPE_UNDEFINED MadariResultType = iota
    MADARI_RESULT_TYPE_BOOL
    MADARI_RESULT_TYPE_STRING
    MADARI_RESULT_TYPE_STAT
    MADARI_RESULT_TYPE_DATA
    MADARI_RESULT_TYPE_DATAANDSTAT
    MADARI_RESULT_TYPE_CHILDREN
    MADARI_RESULT_TYPE_ACL
    MADARI_RESULT_TYPE_MADARIOPRESULTCOLLECTION
)

type MadariExceptionCode int

const (
    MADARIEXCEPTIONCODE_APIERROR MadariExceptionCode = iota
    MADARIEXCEPTIONCODE_AUTHFAILED
    MADARIEXCEPTIONCODE_BADARGUMENTS
    MADARIEXCEPTIONCODE_BADVERSION
    MADARIEXCEPTIONCODE_CONNECTIONLOSS
    MADARIEXCEPTIONCODE_DATAINCONSISTENCY
    MADARIEXCEPTIONCODE_INVALIDACL
    MADARIEXCEPTIONCODE_INVALIDCALLBACK
    MADARIEXCEPTIONCODE_MARSHALLINGERROR
    MADARIEXCEPTIONCODE_NOAUTH
    MADARIEXCEPTIONCODE_NOCHILDRENFOREPHEMERALS
    MADARIEXCEPTIONCODE_NODEEXISTS
    MADARIEXCEPTIONCODE_NONODE
    MADARIEXCEPTIONCODE_NOTEMPTY
    MADARIEXCEPTIONCODE_OK
    MADARIEXCEPTIONCODE_OPERATIONTIMEOUT
    MADARIEXCEPTIONCODE_RUNTIMEINCONSISTENCY
    MADARIEXCEPTIONCODE_SESSIONEXPIRED
    MADARIEXCEPTIONCODE_SESSIONMOVED
    MADARIEXCEPTIONCODE_SYSTEMERROR
    MADARIEXCEPTIONCODE_UNIMPLEMENTED
    MADARIEXCEPTIONCODE_UNKNOWN
    MADARIEXCEPTIONCODE_TRANSACTIONNOTAGREED
    MADARIEXCEPTIONCODE_WAITQUEUETIMEOUTONSERVER
)

type MadariHealth struct {
    Service MadariServiceType
    Identity string
    HealthyInstances uint32
    TotalInstances uint32
}

type MadariApplicationHealth struct {
    HealthEntries []MadariHealth
}

func CreateMadariApplicationHealthFromBuffer(applicationHealth *C.struct_MADARI_APPLICATIONHEALTH) *MadariApplicationHealth {
    var madariApplicationHealth MadariApplicationHealth

    madariApplicationHealth.HealthEntries = make([]MadariHealth, applicationHealth.NumHealthEntries)
    healthEntryPtr := uintptr(unsafe.Pointer(applicationHealth.Health))
    for i := 0; i < len(madariApplicationHealth.HealthEntries); i++ {
        health := (*C.struct_MADARI_HEALTH)(unsafe.Pointer(healthEntryPtr))
        madariApplicationHealth.HealthEntries[i].Service = (MadariServiceType)(health.Service)
        madariApplicationHealth.HealthEntries[i].Identity = WcharStringPtrToGoString(unsafe.Pointer(health.Identity))
        madariApplicationHealth.HealthEntries[i].HealthyInstances = uint32(health.HealthyInstances)
        madariApplicationHealth.HealthEntries[i].TotalInstances = uint32(health.TotalInstances)
        healthEntryPtr += sizeofMadariHealth()
    }

    return &madariApplicationHealth
}

func (stat *MadariStat) DeserializeFromBuffer(buf *C.struct_MADARI_STAT) {
    stat.Aversion = uint32(buf.Aversion)
    stat.Ctime = int64(buf.Ctime)
    stat.Cversion = uint32(buf.Cversion)
    stat.Czxid = int64(buf.Czxid)
    stat.DataLength = uint32(buf.DataLength)
    stat.Mtime = int64(buf.Mtime)
    stat.Mzxid = int64(buf.Mzxid)
    stat.NumChildren = uint32(buf.NumChildren)
    stat.Pzxid = int64(buf.Pzxid)
    stat.Version = uint32(buf.Version)
}

type MadariNotify struct {
    PartitionKey string
    RelativePath string
    Type MadariNotifyType
    Children []string
    Data []byte
    Version uint32
    ChildrenVersion uint32
    Ctime int64
    Mtime int64
}

func CreateMadariNotifyFromBuffer(notify *C.struct_MADARI_NOTIFY) *MadariNotify {
    var madariNotify MadariNotify

    madariNotify.PartitionKey = WcharStringPtrToGoString(unsafe.Pointer(notify.PartitionKey))
    madariNotify.RelativePath = WcharStringPtrToGoString(unsafe.Pointer(notify.RelativePath))
    madariNotify.Type = (MadariNotifyType)(notify.Type)
    madariNotify.Children = make([]string, notify.NumChildren)
    wcharPtrPtr := uintptr(unsafe.Pointer(notify.Children))
    for i := 0; i < len(madariNotify.Children); i++ {
        madariNotify.Children[i] = WcharStringPtrToGoString(unsafe.Pointer(*(**C.wchar_t)(unsafe.Pointer(wcharPtrPtr))))
        wcharPtrPtr += sizeofWcharPtrPtr()
    }
    madariNotify.Data = C.GoBytes(unsafe.Pointer(notify.Data), (C.int)(notify.DataLength))
    madariNotify.Version = uint32(notify.Version)
    madariNotify.ChildrenVersion = uint32(notify.ChildrenVersion)
    madariNotify.Ctime = int64(notify.Ctime)
    madariNotify.Mtime = int64(notify.MTime)

    return &madariNotify
}

type MadariOpResultData interface {
    DeserializeFromBuffer(buf unsafe.Pointer)
}

type MadariOpResultCreate struct {
    Path string
    Stat MadariStat
}

func (result *MadariOpResultCreate) DeserializeFromBuffer(buf unsafe.Pointer) {
    result.Path = WcharStringPtrToGoString(unsafe.Pointer(*(**C.wchar_t)(buf)))
    result.Stat.DeserializeFromBuffer((*C.struct_MADARI_STAT)(unsafe.Pointer(uintptr(buf) + offsetMadariOpResultCreateStat())))
}

func (result *MadariOpResultCreate) String() string {
    return fmt.Sprintf("{%v %v}", result.Path, result.Stat)
}

type MadariOpResultGetData struct {
    Path string
    Stat MadariStat
    Data []byte
}

func (result *MadariOpResultGetData) DeserializeFromBuffer(buf unsafe.Pointer) {
    result.Path = WcharStringPtrToGoString(unsafe.Pointer(*(**C.wchar_t)(buf)))
    result.Stat.DeserializeFromBuffer((*C.struct_MADARI_STAT)(unsafe.Pointer(uintptr(buf) + offsetMadariOpResultGetDataStat())))
    length := int32(*((*int32)(unsafe.Pointer(uintptr(buf) + offsetMadariOpResultGetDataDataLength()))))
    result.Data = C.GoBytes(unsafe.Pointer(*(**C.uint8_t)(unsafe.Pointer(uintptr(buf) + offsetMadariOpResultGetDataData()))), (C.int)(length))
}

func (result *MadariOpResultGetData) String() string {
    return fmt.Sprintf("{%v %v %v}", result.Path, result.Stat, result.Data)
}

type MadariOpResultDelete struct {
}

func (result *MadariOpResultDelete) DeserializeFromBuffer(buf unsafe.Pointer) {
}

func (result *MadariOpResultDelete) String() string {
    return "{}"
}

type MadariOpResultSetData struct {
    Stat MadariStat
}

func (result *MadariOpResultSetData) DeserializeFromBuffer(buf unsafe.Pointer) {
    result.Stat.DeserializeFromBuffer((*C.struct_MADARI_STAT)(buf))
}

func (result *MadariOpResultSetData) String() string {
    return fmt.Sprintf("{%v}", result.Stat)
}

type MadariOpResultSetAcl struct {
    Stat MadariStat
}

func (result *MadariOpResultSetAcl) DeserializeFromBuffer(buf unsafe.Pointer) {
    result.Stat.DeserializeFromBuffer((*C.struct_MADARI_STAT)(buf))
}

func (result *MadariOpResultSetAcl) String() string {
    return fmt.Sprintf("{%v}", result.Stat)
}

type MadariOpResultError struct {
    resultCode int
}

func (result *MadariOpResultError) DeserializeFromBuffer(buf unsafe.Pointer) {
    result.resultCode = int(*((*int32)(buf)))
}

func (result *MadariOpResultError) String() string {
    return fmt.Sprintf("{%v}", result.resultCode)
}

type MadariOpResult struct {
    ResultType MadariOpCode
    ErrorCode MadariExceptionCode
    Data MadariOpResultData
}

type MadariResult struct {
    Type MadariResultType
    PartitionKey string
    RelativePath string
    Result bool
    StringResult string
    Children []string
    Data []byte
    StatIsValid bool
    Stat MadariStat
    Acls []MadariAcl
    OpResults []MadariOpResult
}

func CreateMadariResultFromBuffer(result *C.struct_MADARI_RESULT) *MadariResult {
    var madariResult MadariResult

    madariResult.Type = (MadariResultType)(result.Type)
    madariResult.PartitionKey = WcharStringPtrToGoString(unsafe.Pointer(result.PartitionKey))
    madariResult.RelativePath = WcharStringPtrToGoString(unsafe.Pointer(result.RelativePath))
    madariResult.Result = bool(result.bResult)
    madariResult.StringResult = WcharStringPtrToGoString(unsafe.Pointer(result.sResult))
    madariResult.Children = make([]string, result.NumChildren)
    wcharPtrPtr := uintptr(unsafe.Pointer(result.Children))
    for i := 0; i < len(madariResult.Children); i++ {
        madariResult.Children[i] = WcharStringPtrToGoString(unsafe.Pointer(*(**C.wchar_t)(unsafe.Pointer(wcharPtrPtr))))
        wcharPtrPtr += sizeofWcharPtrPtr()
    }
    madariResult.Data = C.GoBytes(unsafe.Pointer(result.Data), (C.int)(result.DataLength))
    madariResult.StatIsValid = bool(result.StatIsValid)
    madariResult.Stat.DeserializeFromBuffer(&result.Stat)
    madariResult.Acls = make([]MadariAcl, result.NumAcls)
    aclPtr := uintptr(unsafe.Pointer(result.Acls))
    for i:= 0; i < len(madariResult.Acls); i++ {
        madariAcl := ((*C.struct_MADARI_ACL)(unsafe.Pointer(aclPtr)))
        madariResult.Acls[i].Permissions = int(madariAcl.Permissions)
        madariResult.Acls[i].Identifier.Scheme = WcharStringPtrToGoString(unsafe.Pointer(madariAcl.Identifier.Scheme))
        madariResult.Acls[i].Identifier.Identifier = WcharStringPtrToGoString(unsafe.Pointer(madariAcl.Identifier.Identifier))
        aclPtr += sizeofMadariAcl()
    }
    madariResult.OpResults = make([]MadariOpResult, result.NumResults)
    opResultPtr := uintptr(unsafe.Pointer(result.MadariOpResults))
    for i := 0; i < len(madariResult.OpResults); i++ {
        madariOpResult := ((*C.struct_MADARI_OPRESULT)(unsafe.Pointer(opResultPtr)))
        madariResult.OpResults[i].ResultType = (MadariOpCode)(madariOpResult.ResultType)
        madariResult.OpResults[i].ErrorCode = (MadariExceptionCode)(madariOpResult.ErrCode)
        var data MadariOpResultData
        switch madariResult.OpResults[i].ResultType {
        case MADARIOPCODE_CREATE:
            data = &MadariOpResultCreate{}
        case MADARIOPCODE_GETDATA:
            data = &MadariOpResultGetData{}
        case MADARIOPCODE_DELETE:
            data = &MadariOpResultDelete{}
        case MADARIOPCODE_SETDATA:
            data = &MadariOpResultSetData{}
        case MADARIOPCODE_SETACL:
            data = &MadariOpResultSetAcl{}
        case MADARIOPCODE_ERROR:
            data = &MadariOpResultError{}
        default:
        }

        if nil != data {
            data.DeserializeFromBuffer(unsafe.Pointer(opResultPtr + offsetMadariOpResultData()))
            madariResult.OpResults[i].Data = data
        }
        opResultPtr += sizeofMadariOpResult()
    }

    return &madariResult
}

type MadariClientCallback interface {
    OnNotificationReceived(notify *MadariNotify)
    OnResultReceived(result *MadariResult)
    OnFailureReceived(partitionKey string, relativePath string, reason string, errorSource MadariErrorSrc, errorCode int32)
    OnLoggingMessageReceived(traceLevel MadariTraceLevel, message string)
    OnHealthNotificationReceived(applicationHealth *MadariApplicationHealth)
    OnSessionTerminationReceived()
}

type DefaultMadariClientCallback struct {
}

func (callback DefaultMadariClientCallback) OnNotificationReceived(notify *MadariNotify) {
    fmt.Printf("OnNotificationReceived: %v\r\n", *notify)
}

func (callback DefaultMadariClientCallback) OnResultReceived(result *MadariResult) {
    fmt.Printf("OnResultReceived: %v\r\n", *result)
}

func (callback DefaultMadariClientCallback) OnFailureReceived(partitionKey string, relativePath string, reason string, errorSource MadariErrorSrc, errorCode int32) {
    fmt.Printf("OnFailureReceived: {%v %v %v %v %v}\r\n", partitionKey, relativePath, reason, errorSource, errorCode)
}

func (callback DefaultMadariClientCallback) OnLoggingMessageReceived(traceLevel MadariTraceLevel, message string) {
    fmt.Printf("OnLoggingMessageReceived: %v %v", traceLevel, message)
}

func (callback DefaultMadariClientCallback) OnHealthNotificationReceived(applicationHealth *MadariApplicationHealth) {
    fmt.Printf("OnHealthNotificationReceived: %v\r\n", *applicationHealth)
}

func (callback DefaultMadariClientCallback) OnSessionTerminationReceived() {
    fmt.Println("Session terminated")
}

//export NotifyReceived
func NotifyReceived(callbackId int, notify *C.struct_MADARI_NOTIFY) {
    callback := callbackMapLookup(callbackId)
    if nil != callback {
        if clientCallback, ok := callback.(MadariClientCallback); ok {
            madariNotify := CreateMadariNotifyFromBuffer(notify)

            go func() {
                defer panicHandlerInCallback("NotifyReceived", fmt.Sprintf("%v", *madariNotify))
                clientCallback.OnNotificationReceived(madariNotify)
            }()
        }
    }
}

//export ResultReceived
func ResultReceived(callbackId int, result *C.struct_MADARI_RESULT) {
    callback := callbackMapLookup(callbackId)
    if nil != callback {
        if clientCallback, ok := callback.(MadariClientCallback); ok {
            madariResult := CreateMadariResultFromBuffer(result)

            go func() {
                defer panicHandlerInCallback("ResultReceived", fmt.Sprintf("%v", *madariResult))
                clientCallback.OnResultReceived(madariResult)
            }()
        }
    }
}

//export FailureReceived
func FailureReceived(callbackId int, partitionKey *C.wchar_t, relativePath *C.wchar_t, reason *C.wchar_t, errorSource int, errorCode int32) {
    callback := callbackMapLookup(callbackId)
    if nil != callback {
        if clientCallback, ok := callback.(MadariClientCallback); ok {
            partitionKeyStr := WcharStringPtrToGoString(unsafe.Pointer(partitionKey))
            relativePathStr := WcharStringPtrToGoString(unsafe.Pointer(relativePath))
            reasonStr := WcharStringPtrToGoString(unsafe.Pointer(reason))
            errorSourceEnum := (MadariErrorSrc)(errorSource)

            go func() {
                defer panicHandlerInCallback("FailureReceived", fmt.Sprintf("{%v %v %v %v %v}", partitionKeyStr, relativePathStr, reasonStr, errorSourceEnum, errorCode))
                clientCallback.OnFailureReceived(partitionKeyStr, relativePathStr, reasonStr, errorSourceEnum, errorCode)
            }()
        }
    }
}

//export LoggingMessageReceived
func LoggingMessageReceived(callbackId int, traceLevel int, message *C.wchar_t) {
    callback := callbackMapLookup(callbackId)
    if nil != callback {
        if clientCallback, ok := callback.(MadariClientCallback); ok {
            messageStr := WcharStringPtrToGoString(unsafe.Pointer(message))
            traceLevelEnum := (MadariTraceLevel)(traceLevel)

            go func() {
                defer panicHandlerInCallback("LoggingMessageReceived", fmt.Sprintf("{%v %v}", traceLevelEnum, messageStr))
                clientCallback.OnLoggingMessageReceived(traceLevelEnum, messageStr)
            }()
        }
    }
}

//export HealthNotificationReceived
func HealthNotificationReceived(callbackId int, applicationHealth *C.struct_MADARI_APPLICATIONHEALTH) {
    callback := callbackMapLookup(callbackId)
    if nil != callback {
        if clientCallback, ok := callback.(MadariClientCallback); ok {
            madariApplicationHealth := CreateMadariApplicationHealthFromBuffer(applicationHealth)

            go func() {
                defer panicHandlerInCallback("HealthNotificationReceived", fmt.Sprintf("%v", *madariApplicationHealth))
                clientCallback.OnHealthNotificationReceived(madariApplicationHealth)
            }()
        }
    }
}

//export SessionTerminationReceived
func SessionTerminationReceived(callbackId int) {
    callback := callbackMapLookup(callbackId)
    if nil != callback {
        if clientCallback, ok := callback.(MadariClientCallback); ok {
            go func() {
                defer panicHandlerInCallback("SessionTerminationReceived", "")
                clientCallback.OnSessionTerminationReceived()
            }()
        }
    }
}

func panicHandlerInCallback(callbackName string, context string) {
    if r := recover(); r != nil {
        fmt.Println("Panic in", callbackName, "context", context, "error message", r)
    }
}

var callbackTrack struct {
    sync.RWMutex
    m map[int]interface{}
    c int
}

func callbackMapAdd(v interface{}) int {
    callbackTrack.Lock()
    defer callbackTrack.Unlock()
    if callbackTrack.m == nil {
        callbackTrack.m = make(map[int]interface{})
    }
    callbackTrack.c++
    ret := callbackTrack.c
    callbackTrack.m[ret] = v
    return ret
}

func callbackMapLookup(c int) interface{} {
    callbackTrack.RLock()
    defer callbackTrack.RUnlock()
    return callbackTrack.m[c]
}

func callbackMapDelete(c int) {
    callbackTrack.Lock()
    defer callbackTrack.Unlock()
    if callbackTrack.m[c] != nil {
        delete(callbackTrack.m, c)
    }
}
