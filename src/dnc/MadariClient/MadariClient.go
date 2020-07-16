package MadariClient

/*
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

enum MADARI_TRACE_LEVEL {
    MADARI_TRACE_ERROR = 1,
    MADARI_TRACE_WARNING = 2,
    MADARI_TRACE_VERBOSE = 3,
    MADARI_TRACE_PAINFUL = 4
};

enum MADARIERRORSRC {
    MADARIERRORSRC_UNKNOWN = 0,
    MADARIERRORSRC_FRONTENDSERVICE = 1,
    MADARIERRORSRC_MADARISERVICE = 2,
    MADARIERRORSRC_SUBSCRIBER = 3
};

enum MADARI_SERVICETYPE {
    MADARI_SERVICETYPE_UNKNOWN = 0,
    MADARI_SERVICETYPE_HEALTHMONITOR = 1,
    MADARI_SERVICETYPE_FRONTEND = 2,
    MADARI_SERVICETYPE_NOTIFICATIONSERVICE = 3,
    MADARI_SERVICETYPE_SELECTORSERVICE = 4,
    MADARI_SERVICETYPE_MADARISERVICE = 5
};

enum MADARI_NOTIFY_TYPE {
    MADARI_NOTIFY_NODE_INITIAL_STATE = 1,
    MADARI_NOTIFY_NODE_CHILDREN_CHANGED = 2,
    MADARI_NOTIFY_NODE_CREATED = 3,
    MADARI_NOTIFY_NODE_DATA_CHANGED = 4,
    MADARI_NOTIFY_NODE_DELETED = 5
};

enum MADARICREATEACTION {
    MADARICREATEACTION_DONTCREATE = 0,
    MADARICREATEACTION_CREATEIFNOTEXISTS = 1
};

enum MADARIOPCODE {
    MADARIOPCODE_UNKNOWN = 0,
    MADARIOPCODE_AUTH = 1,
    MADARIOPCODE_CHECK = 2,
    MADARIOPCODE_CLOSESESSION = 3,
    MADARIOPCODE_CREATE = 4,
    MADARIOPCODE_CREATESESSION = 5,
    MADARIOPCODE_DELETE = 6,
    MADARIOPCODE_ERROR = 7,
    MADARIOPCODE_EXISTS = 8,
    MADARIOPCODE_GETACL = 9,
    MADARIOPCODE_GETCHILDREN = 10,
    MADARIOPCODE_GETCHILDREN2 = 11,
    MADARIOPCODE_GETDATA = 12,
    MADARIOPCODE_MULTI = 13,
    MADARIOPCODE_NOTIFICATION = 14,
    MADARIOPCODE_PING = 15,
    MADARIOPCODE_SASL = 16,
    MADARIOPCODE_SETACL = 17,
    MADARIOPCODE_SETDATA = 18,
    MADARIOPCODE_SETWATCHES = 19,
    MADARIOPCODE_SYNC = 20,
    MADARIOPCODE_MOVE = 21
};

enum MADARIEXCEPTIONCODE {
    MADARIEXCEPTIONCODE_APIERROR = 0,
    MADARIEXCEPTIONCODE_AUTHFAILED = 1,
    MADARIEXCEPTIONCODE_BADARGUMENTS = 2,
    MADARIEXCEPTIONCODE_BADVERSION = 3,
    MADARIEXCEPTIONCODE_CONNECTIONLOSS = 4,
    MADARIEXCEPTIONCODE_DATAINCONSISTENCY = 5,
    MADARIEXCEPTIONCODE_INVALIDACL = 6,
    MADARIEXCEPTIONCODE_INVALIDCALLBACK = 7,
    MADARIEXCEPTIONCODE_MARSHALLINGERROR = 8,
    MADARIEXCEPTIONCODE_NOAUTH = 9,
    MADARIEXCEPTIONCODE_NOCHILDRENFOREPHEMERALS = 10,
    MADARIEXCEPTIONCODE_NODEEXISTS = 11,
    MADARIEXCEPTIONCODE_NONODE = 12,
    MADARIEXCEPTIONCODE_NOTEMPTY = 13,
    MADARIEXCEPTIONCODE_OK = 14,
    MADARIEXCEPTIONCODE_OPERATIONTIMEOUT = 15,
    MADARIEXCEPTIONCODE_RUNTIMEINCONSISTENCY = 16,
    MADARIEXCEPTIONCODE_SESSIONEXPIRED = 17,
    MADARIEXCEPTIONCODE_SESSIONMOVED = 18,
    MADARIEXCEPTIONCODE_SYSTEMERROR = 19,
    MADARIEXCEPTIONCODE_UNIMPLEMENTED = 20,
    MADARIEXCEPTIONCODE_UNKNOWN = 21,
    MADARIEXCEPTIONCODE_TRANSACTIONNOTAGREED = 22,
    MADARIEXCEPTIONCODE_WAITQUEUETIMEOUTONSERVER = 23
};

enum MADARIOPSGETDATAOPTIONS {
    MADARIOPSGETDATAOPTIONS_NONE = 0,
    MADARIOPSGETDATAOPTIONS_FAULTBACKONPARENTDATA = 1,
    MADARIOPSGETDATAOPTIONS_NOSTATREQUIRED = 2,
    MADARIOPSGETDATAOPTIONS_NOWILDCARDSFORPATH = 4,
    MADARIOPSGETDATAOPTIONS_FAULTBACKONPARENTDATAWITHMATCH = 8
};

enum MADARIMOVEMODE {
    MADARIMOVEMODE_NONE = 0,
    MADARIMOVEMODE_ALLOWPATHCREATIONFLAG = 4096,
    MADARIMOVEMODE_ONLYIFSOURCEPATHISEMPTY = 8192
};

enum MADARIOPSEXECUTIONMODE {
    MADARIOPSEXECUTIONMODE_TRANSACTED = 0,
    MADARIOPSEXECUTIONMODE_BATCH = 1
};

enum MADARI_RESULT_TYPE {
    MADARI_RESULT_TYPE_UNDEFINED = 0,
    MADARI_RESULT_TYPE_BOOL = 1,
    MADARI_RESULT_TYPE_STRING = 2,
    MADARI_RESULT_TYPE_STAT = 3,
    MADARI_RESULT_TYPE_DATA = 4,
    MADARI_RESULT_TYPE_DATAANDSTAT = 5,
    MADARI_RESULT_TYPE_CHILDREN = 6,
    MADARI_RESULT_TYPE_ACL = 7,
    MADARI_RESULT_TYPE_MADARIOPRESULTCOLLECTION = 8
};

struct MADARI_HEALTH {
    enum MADARI_SERVICETYPE Service;
    const wchar_t* Identity;
    uint32_t HealthyInstances;
    uint32_t TotalInstances;
};

struct MADARI_APPLICATIONHEALTH {
    uint32_t NumHealthEntries;
    struct MADARI_HEALTH* Health;
};

struct MADARI_NOTIFY {
    const wchar_t* PartitionKey;
    const wchar_t* RelativePath;
    enum MADARI_NOTIFY_TYPE Type;
    const wchar_t** Children;
    uint32_t NumChildren;
    const void* Data;
    uint32_t DataLength;
    uint32_t Version;
    uint32_t ChildrenVersion;
    int64_t Ctime;
    int64_t MTime;
};

struct MADARI_ID {
    const wchar_t* Scheme;
    const wchar_t* Identifier;
};

struct MADARI_STAT {
    uint32_t Aversion;
    int64_t Ctime;
    uint32_t Cversion;
    int64_t Czxid;
    uint32_t DataLength;
    int64_t Mtime;
    int64_t Mzxid;
    uint32_t NumChildren;
    int64_t Pzxid;
    uint32_t Version;
};

struct MADARI_ACL {
    int Permissions;
    struct MADARI_ID Identifier;
};

struct MADARI_OPCREATE {
    uint8_t* Data;
    uint32_t DataLen;
    struct MADARI_ACL* Acls;
    uint32_t AclLen;
    enum MADARICREATEACTION Mode;
};

struct MADARI_OPCHECK {
    int Version;
    int CVersion;
    int AVersion;
    const wchar_t* UniqueIncarnationId;
    _Bool IsExtendedCheck;
};

struct MADARI_OPGETDATA {
    enum MADARIOPSGETDATAOPTIONS Options;
    struct MADARI_OPCHECK* CheckUsedForThisPath;
};

struct MADARI_OPDELETE {
    int Version;
    _Bool Recursive;
};

struct MADARI_OPSETDATA {
    uint8_t* Data;
    uint32_t DataLen;
    int Version;
};

struct MADARI_OPSETACL {
    struct MADARI_ACL* Acls;
    uint32_t AclLen;
    int Version;
};

struct MADARI_OPMOVE {
    int Version;
    const wchar_t* DstPath;
    enum MADARIMOVEMODE Mode;
};

struct MADARI_OP {
    const wchar_t* Path;
    enum MADARIOPCODE OpType;
    union {
        struct MADARI_OPCREATE OpCreate;
        struct MADARI_OPCHECK OpCheck;
        struct MADARI_OPGETDATA OpGetData;
        struct MADARI_OPDELETE OpDelete;
        struct MADARI_OPSETDATA OpSetData;
        struct MADARI_OPSETACL OpSetAcl;
        struct MADARI_OPMOVE OpMove;
    } OpData; /// One of the OpTypes
};

struct MADARI_OPRESULT_CHECK {
};

struct MADARI_OPRESULT_CREATE {
    const wchar_t* Path;
    struct MADARI_STAT Stat;
};

struct MADARI_OPRESULT_MOVE {
    const wchar_t* DstPath;
    struct MADARI_STAT Stat;
};

struct MADARI_OPRESULT_DELETE {
};

struct MADARI_OPRESULT_ERROR {
    int ResultCode;
};

struct MADARI_OPRESULT_GETDATA {
    const wchar_t* Path;
    struct MADARI_STAT Stat;
    uint8_t* Data;
    uint32_t DataLength;
};

struct MADARI_OPRESULT_RUN {
    struct MADARI_OPRESULT* Results;
    uint32_t ResultLength;
};

struct MADARI_OPRESULT_SETACL {
    struct MADARI_STAT Stat;
};

struct MADARI_OPRESULT_SETDATA {
    struct MADARI_STAT Stat;
};

struct MADARI_OPRESULT {
    enum MADARIOPCODE ResultType;
    enum MADARIEXCEPTIONCODE ErrCode;
    union {
        struct MADARI_OPRESULT_CHECK CheckResult;
        struct MADARI_OPRESULT_CREATE CreateResult;
        struct MADARI_OPRESULT_DELETE DeleteResult;
        struct MADARI_OPRESULT_ERROR ErrorResult;
        struct MADARI_OPRESULT_GETDATA GetDataResult;
        struct MADARI_OPRESULT_RUN RunResult;
        struct MADARI_OPRESULT_SETACL SetAclResult;
        struct MADARI_OPRESULT_SETDATA SetDataResult;
        struct MADARI_OPRESULT_MOVE MoveResult;
    } ResultData;
};

struct MADARI_RESULT {
    enum MADARI_RESULT_TYPE Type;
    const wchar_t* PartitionKey;
    const wchar_t* RelativePath;
    _Bool bResult;
    const wchar_t* sResult;
    const wchar_t** Children;
    uint32_t NumChildren;
    const void* Data;
    uint32_t DataLength;
    _Bool StatIsValid;
    struct MADARI_STAT Stat;
    struct MADARI_ACL* Acls;
    uint32_t NumAcls;
    struct MADARI_OPRESULT* MadariOpResults;
    uint32_t NumResults;
};

extern int32_t MadariCreateSession(uintptr_t, uintptr_t, uintptr_t, const wchar_t *, uintptr_t, uintptr_t, uintptr_t, uint32_t, uint32_t, uint32_t, uintptr_t, _Bool);
extern int32_t MadariDestroySession(uintptr_t);
extern int32_t MadariOpen(uintptr_t, const wchar_t *, const wchar_t *, _Bool, const wchar_t *, const wchar_t *);
extern int32_t MadariClose(uintptr_t);
extern int32_t MadariSubscribeSync(uintptr_t, const wchar_t *, const wchar_t *, _Bool);
extern int32_t MadariUnsubscribe(uintptr_t, const wchar_t *, const wchar_t *);

extern int32_t MadariFrontEndCreateSession(uintptr_t, uintptr_t, uintptr_t, const wchar_t *, uintptr_t, uintptr_t, uintptr_t, uint32_t, uint32_t, uintptr_t, uintptr_t);
extern int32_t MadariFrontEndDestroySession(uintptr_t);
extern int32_t MadariFrontEndOpen(uintptr_t, const wchar_t *, const wchar_t *, _Bool, const wchar_t *, const wchar_t *);
extern int32_t MadariFrontEndClose(uintptr_t);
extern int32_t MadariFrontEndAddContextSelector(uintptr_t, const wchar_t *);
extern int32_t MadariFrontEndRemoveContextSelector(uintptr_t, const wchar_t *);
extern int32_t MadariFrontEndUpdateNode(uintptr_t, _Bool, const wchar_t *, const wchar_t *, uintptr_t, uint32_t, uintptr_t, uint32_t, int);
extern int32_t MadariFrontEndRemoveNode(uintptr_t, _Bool, const wchar_t *, const wchar_t *, _Bool);
extern int32_t MadariFrontEndSetPubSubAcl(uintptr_t, _Bool, const wchar_t *, const wchar_t *, uintptr_t, uint32_t, int);
extern int32_t MadariFrontEndGetPubSubAcl(uintptr_t, _Bool, const wchar_t *, const wchar_t *, uintptr_t);
extern int32_t MadariFrontEndExists(uintptr_t, _Bool, const wchar_t *, const wchar_t *);
extern int32_t MadariFrontEndGetData(uintptr_t, _Bool, const wchar_t *, const wchar_t *);
extern int32_t MadariFrontEndGetDataV2(uintptr_t, _Bool, const wchar_t *, const wchar_t *);
extern int32_t MadariFrontEndGetChildren(uintptr_t, _Bool, const wchar_t *, const wchar_t *);
extern int32_t MadariFrontEndExecuteOperationCollection(uintptr_t, _Bool, const wchar_t *, int, uintptr_t, uint32_t, _Bool);

extern void NotifyReceived(intgo, struct MADARI_NOTIFY *);
void MadariSubscriberNotifyReceived(void *Context, struct MADARI_NOTIFY *Notify) {
    NotifyReceived((intgo)Context, Notify);
}
extern void ResultReceived(intgo, struct MADARI_RESULT *);
void MadariSubscriberResultReceived(void *Context, struct MADARI_RESULT *Result) {
    ResultReceived((intgo)Context, Result);
}
extern void FailureReceived(intgo, const wchar_t *, const wchar_t *, const wchar_t *, enum MADARIERRORSRC, int32_t);
void MadariSubscriberFailureReceived(void *Context, const wchar_t *PartitionKey, const wchar_t *RelativePath, const wchar_t *Reason, enum MADARIERRORSRC ErrorSource, int32_t ErrorCode) {
    FailureReceived((intgo)Context, PartitionKey, RelativePath, Reason, ErrorSource, ErrorCode);
}
extern void LoggingMessageReceived(intgo, enum MADARI_TRACE_LEVEL, const wchar_t *);
void MadariSubscriberLoggingMessageReceived(void *Context, enum MADARI_TRACE_LEVEL Level, const wchar_t *Message) {
    LoggingMessageReceived((intgo)Context, Level, Message);
}
extern void HealthNotificationReceived(intgo, struct MADARI_APPLICATIONHEALTH *);
void MadariSubscriberHealthNotificationReceived(void *Context, struct MADARI_APPLICATIONHEALTH *Health) {
    HealthNotificationReceived((intgo)Context, Health);
}
extern void SessionTerminationReceived(intgo);
void MadariSubscriberSessionTerminationReceived(void *Context) {
    SessionTerminationReceived((intgo)Context);
}

size_t SizeofWcharPtrPtr() {
    return sizeof(wchar_t **);
}
size_t SizeOfMadariID() {
    return sizeof(struct MADARI_ID);
}
size_t SizeofMadariHealth() {
    return sizeof(struct MADARI_HEALTH);
}
size_t SizeofMadariAcl() {
    return sizeof(struct MADARI_ACL);
}
size_t SizeofMadariOp() {
    return sizeof(struct MADARI_OP);
}
size_t OffsetMadariOpData() {
    return offsetof(struct MADARI_OP, OpData);
}
size_t SizeofMadariOpResult() {
    return sizeof(struct MADARI_OPRESULT);
}
size_t OffsetMadariOpResultData() {
    return offsetof(struct MADARI_OPRESULT, ResultData);
}
size_t OffsetMadariOpResultCreateStat() {
    return offsetof(struct MADARI_OPRESULT_CREATE, Stat);
}
size_t OffsetMadariOpResultMoveStat() {
    return offsetof(struct MADARI_OPRESULT_MOVE, Stat);
}
size_t OffsetMadariOpResultGetDataStat() {
    return offsetof(struct MADARI_OPRESULT_GETDATA, Stat);
}
size_t OffsetMadariOpResultGetDataData() {
    return offsetof(struct MADARI_OPRESULT_GETDATA, Data);
}
size_t OffsetMadariOpResultGetDataDataLength() {
    return offsetof(struct MADARI_OPRESULT_GETDATA, DataLength);
}
size_t OffsetMadariOpResultRunResultLength() {
    return offsetof(struct MADARI_OPRESULT_RUN, ResultLength);
}
*/
import "C"
import "unsafe"

const DefaultLoggingContextSubscriber = "GoSubscriber"
const DefaultLoggingContextPublisher = "GoPublisher"

type MadariSubscriber interface {
    Open(host string, port string, cert string, expectedServerSubject string) int
    Close() int
    Subscribe(key string, path string, pathIsPrefix bool) int
    Unsubscribe(key string, path string) int
}

type DefaultMadariSubscriber struct {
    callbackId int
    session uintptr
}

func CreateDefaultMadariSubscriber(callback MadariClientCallback) DefaultMadariSubscriber {
    return CreateDefaultMadariSubscriberWithLoggingContext(callback, DefaultLoggingContextSubscriber)
}

func CreateDefaultMadariSubscriberWithLoggingContext(callback MadariClientCallback, loggingContext string) DefaultMadariSubscriber {
    loggingContextBuffer := GoStringToWcharStringPtr(loggingContext)
    defer DeleteWcharBuffer(loggingContextBuffer)

    callbackId := callbackMapAdd(callback)

    var session uintptr
    C.MadariCreateSession(
        C.uintptr_t(callbackId),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberNotifyReceived))),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberFailureReceived))),
        loggingContextBuffer,
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberLoggingMessageReceived))),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberHealthNotificationReceived))),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberSessionTerminationReceived))),
        10000,
        60000,
        30000,
        C.uintptr_t(uintptr(unsafe.Pointer(&session))),
        true,
    )

    return DefaultMadariSubscriber{callbackId, session}
}

func DeleteDefaultMadariSubscriber(subscriber DefaultMadariSubscriber) int {
    callbackMapDelete(subscriber.callbackId)
    return (int)(C.MadariDestroySession(C.uintptr_t(subscriber.session)))
}

func (subscriber DefaultMadariSubscriber) Open(host string, port string, cert string, expectedServerSubject string) int {
    hostBuffer := GoStringToWcharStringPtr(host)
    defer DeleteWcharBuffer(hostBuffer)
    portBuffer := GoStringToWcharStringPtr(port)
    defer DeleteWcharBuffer(portBuffer)
    certBuffer := GoStringToWcharStringPtr(cert)
    defer DeleteWcharBuffer(certBuffer)
    expectedServerSubjectBuffer := GoStringToWcharStringPtr(expectedServerSubject)
    defer DeleteWcharBuffer(expectedServerSubjectBuffer)

    return (int)(C.MadariOpen(C.uintptr_t(subscriber.session), hostBuffer, portBuffer, true, certBuffer, expectedServerSubjectBuffer))
}

func (subscriber DefaultMadariSubscriber) Close() int {
    return (int)(C.MadariClose(C.uintptr_t(subscriber.session)))
}

func (subscriber DefaultMadariSubscriber) Subscribe(key string, path string, pathIsPrefix bool) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariSubscribeSync(C.uintptr_t(subscriber.session), keyBuffer, pathBuffer, C._Bool(pathIsPrefix)))
}

func (subscriber DefaultMadariSubscriber) Unsubscribe(key string, path string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariUnsubscribe(C.uintptr_t(subscriber.session), keyBuffer, pathBuffer))
}

type MadariCreateAction int

const (
    MADARICREATEACTION_DONTCREATE MadariCreateAction = iota
    MADARICREATEACTION_CREATEIFNOTEXISTS
)

type MadariOpCode int

const (
    MADARIOPCODE_UNKNOWN MadariOpCode = iota
    MADARIOPCODE_AUTH
    MADARIOPCODE_CHECK
    MADARIOPCODE_CLOSESESSION
    MADARIOPCODE_CREATE
    MADARIOPCODE_CREATESESSION
    MADARIOPCODE_DELETE
    MADARIOPCODE_ERROR
    MADARIOPCODE_EXISTS
    MADARIOPCODE_GETACL
    MADARIOPCODE_GETCHILDREN
    MADARIOPCODE_GETCHILDREN2
    MADARIOPCODE_GETDATA
    MADARIOPCODE_MULTI
    MADARIOPCODE_NOTIFICATION
    MADARIOPCODE_PING
    MADARIOPCODE_SASL
    MADARIOPCODE_SETACL
    MADARIOPCODE_SETDATA
    MADARIOPCODE_SETWATCHES
    MADARIOPCODE_SYNC
    MADARIOPCODE_MOVE
)

type MadariOpsExecutionMode int

const (
    MADARIOPSEXECUTIONMODE_TRANSACTED MadariOpsExecutionMode = iota
    MADARIOPSEXECUTIONMODE_BATCH
)

type MadariOpsGetDataOptions int

const (
    MADARIOPSGETDATAOPTIONS_NONE MadariOpsGetDataOptions = 0
    MADARIOPSGETDATAOPTIONS_FAULTBACKONPARENTDATA = 1
    MADARIOPSGETDATAOPTIONS_NOSTATREQUIRED = 2
    MADARIOPSGETDATAOPTIONS_NOWILDCARDSFORPATH = 4
    MADARIOPSGETDATAOPTIONS_FAULTBACKONPARENTDATAWITHMATCH = 8
)

type MadariMoveMode int

const (
    MADARIMOVEMODE_NONE MadariMoveMode = 0
    MADARIMOVEMODE_ALLOWPATHCREATIONFLAG = 4096
    MADARIMOVEMODE_ONLYIFSOURCEPATHISEMPTY = 8192
)

type MadariId struct {
    Scheme string
    Identifier string
}

type MadariAcl struct {
    Permissions int
    Identifier MadariId
}

type MadariStat struct {
    Aversion uint32
    Ctime int64
    Cversion uint32
    Czxid int64
    DataLength uint32
    Mtime int64
    Mzxid int64
    NumChildren uint32
    Pzxid int64
    Version uint32
}

type MadariOpData interface {
    SerializeToBuffer(buf unsafe.Pointer)
    FreeInBuffer(buf unsafe.Pointer)
}

type MadariOpCreate struct {
    Data []byte
    Acls []MadariAcl
    Action MadariCreateAction
}

func (op MadariOpCreate) SerializeToBuffer(buf unsafe.Pointer) {
    madariOpCreate := (*C.struct_MADARI_OPCREATE)(buf)
    madariOpCreate.Data = (*C.uint8_t)(C.CBytes(op.Data))
    madariOpCreate.DataLen = C.uint32_t(len(op.Data))
    madariOpCreate.Acls = (*C.struct_MADARI_ACL)(newMadariAclBuf(op.Acls))
    madariOpCreate.AclLen = C.uint32_t(len(op.Acls))
    madariOpCreate.Mode = uint32(op.Action)
}

func (op MadariOpCreate) FreeInBuffer(buf unsafe.Pointer) {
    madariOpCreate := (*C.struct_MADARI_OPCREATE)(buf)
    C.free(unsafe.Pointer(madariOpCreate.Data))
    deleteMadariAclBuf(unsafe.Pointer(madariOpCreate.Acls), op.Acls)
}

type MadariOpGetData struct {
    Options MadariOpsGetDataOptions
}

func (op MadariOpGetData) SerializeToBuffer(buf unsafe.Pointer) {
    madariOpGetData := (*C.struct_MADARI_OPGETDATA)(buf)
    madariOpGetData.Options = uint32(op.Options)
    madariOpGetData.CheckUsedForThisPath = (*C.struct_MADARI_OPCHECK)(unsafe.Pointer(uintptr(0)))
}

func (op MadariOpGetData) FreeInBuffer(buf unsafe.Pointer) {
}

type MadariOpDelete struct {
    Version int
    Recursive bool
}

func (op MadariOpDelete) SerializeToBuffer(buf unsafe.Pointer) {
    madariOpDelete := (*C.struct_MADARI_OPDELETE)(buf)
    madariOpDelete.Version = C.int(op.Version)
    madariOpDelete.Recursive = C._Bool(op.Recursive)
}

func (op MadariOpDelete) FreeInBuffer(buf unsafe.Pointer) {
}

type MadariOpSetData struct {
    Data []byte
    Version int
}

func (op MadariOpSetData) SerializeToBuffer(buf unsafe.Pointer) {
    madariOpSetData := (*C.struct_MADARI_OPSETDATA)(buf)
    madariOpSetData.Data = (*C.uint8_t)(C.CBytes(op.Data))
    madariOpSetData.DataLen = C.uint32_t(len(op.Data))
    madariOpSetData.Version = C.int(op.Version)
}

func (op MadariOpSetData) FreeInBuffer(buf unsafe.Pointer) {
    madariOpSetData := (*C.struct_MADARI_OPSETDATA)(buf)
    C.free(unsafe.Pointer(madariOpSetData.Data))
}

type MadariOpSetAcl struct {
    Acls []MadariAcl
    Version int
}

func (op MadariOpSetAcl) SerializeToBuffer(buf unsafe.Pointer) {
    madariOpSetAcl := (*C.struct_MADARI_OPSETACL)(buf)
    madariOpSetAcl.Acls = (*C.struct_MADARI_ACL)(newMadariAclBuf(op.Acls))
    madariOpSetAcl.AclLen = C.uint32_t(len(op.Acls))
    madariOpSetAcl.Version = C.int(op.Version)
}

func (op MadariOpSetAcl) FreeInBuffer(buf unsafe.Pointer) {
    madariOpSetAcl := (*C.struct_MADARI_OPSETACL)(buf)
    deleteMadariAclBuf(unsafe.Pointer(madariOpSetAcl.Acls), op.Acls)
}

type MadariOp struct {
    Path string
    Type MadariOpCode
    Data MadariOpData
}

type MadariPublisher interface {
    Open(host string, port string, cert string, expectedServerSubject string) int
    Close() int
    AddContextSelector(key string) int
    RemoveContextSelector(key string) int
    UpdateNode(key string, path string, data []byte, acls []MadariAcl, action MadariCreateAction) int
    RemoveNode(key string, path string) int
    SetAcls(key string, path string, acls []MadariAcl, version int) int
    GetAcls(key string, path string, stat *MadariStat) int
    Exists(key string, path string) int
    GetData(key string, path string) int
    GetDataV2(key string, path string) int
    GetChildren(key string, path string) int
    ExecuteOperationCollection(key string, mode MadariOpsExecutionMode, ops []MadariOp, completeSynchronously bool) int
}

type DefaultMadariPublisher struct {
    madariId uintptr
    callbackId int
    session uintptr
}

func CreateDefaultMadariPublisher(scheme string, identifier string, callback MadariClientCallback) DefaultMadariPublisher {
    return CreateDefaultMadariPublisherWithLoggingContext(scheme, identifier, callback, DefaultLoggingContextPublisher)
}

func CreateDefaultMadariPublisherWithLoggingContext(scheme string, identifier string, callback MadariClientCallback, loggingContext string) DefaultMadariPublisher {
    madariId := (*C.struct_MADARI_ID)(C.malloc(C.SizeOfMadariID()))
    madariId.Scheme = GoStringToWcharStringPtr(scheme)
    madariId.Identifier = GoStringToWcharStringPtr(identifier)

    loggingContextBuffer := GoStringToWcharStringPtr(loggingContext)
    defer DeleteWcharBuffer(loggingContextBuffer)

    callbackId := callbackMapAdd(callback)

    var session uintptr
    C.MadariFrontEndCreateSession(
        C.uintptr_t(callbackId),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberResultReceived))),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberFailureReceived))),
        loggingContextBuffer,
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberLoggingMessageReceived))),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberHealthNotificationReceived))),
        C.uintptr_t(uintptr(unsafe.Pointer(C.MadariSubscriberSessionTerminationReceived))),
        10000,
        30000,
        C.uintptr_t(uintptr(unsafe.Pointer(madariId))),
        C.uintptr_t(uintptr(unsafe.Pointer(&session))),
    )

    return DefaultMadariPublisher{uintptr(unsafe.Pointer(madariId)), callbackId, session}
}

func DeleteDefaultMadariPublisher(publisher DefaultMadariPublisher) int {
    madariIdPoiter := unsafe.Pointer(publisher.madariId)
    madariId := (*C.struct_MADARI_ID)(madariIdPoiter)
    defer C.free(madariIdPoiter)
    defer DeleteWcharBuffer(madariId.Scheme)
    defer DeleteWcharBuffer(madariId.Identifier)

    defer callbackMapDelete(publisher.callbackId)
    return (int)(C.MadariFrontEndDestroySession(C.uintptr_t(publisher.session)))
}

func (publisher DefaultMadariPublisher) Open(host string, port string, cert string, expectedServerSubject string) int {
    hostBuffer := GoStringToWcharStringPtr(host)
    defer DeleteWcharBuffer(hostBuffer)
    portBuffer := GoStringToWcharStringPtr(port)
    defer DeleteWcharBuffer(portBuffer)
    certBuffer := GoStringToWcharStringPtr(cert)
    defer DeleteWcharBuffer(certBuffer)
    expectedServerSubjectBuffer := GoStringToWcharStringPtr(expectedServerSubject)
    defer DeleteWcharBuffer(expectedServerSubjectBuffer)

    return (int)(C.MadariFrontEndOpen(C.uintptr_t(publisher.session), hostBuffer, portBuffer, true, certBuffer, expectedServerSubjectBuffer))
}

func (publisher DefaultMadariPublisher) Close() int {
    return (int)(C.MadariFrontEndClose(C.uintptr_t(publisher.session)))
}

func (publisher DefaultMadariPublisher) AddContextSelector(key string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)

    return (int)(C.MadariFrontEndAddContextSelector(C.uintptr_t(publisher.session), keyBuffer))
}

func (publisher DefaultMadariPublisher) RemoveContextSelector(key string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)

    return (int)(C.MadariFrontEndRemoveContextSelector(C.uintptr_t(publisher.session), keyBuffer))
}

func (publisher DefaultMadariPublisher) UpdateNode(key string, path string, data []byte, acls []MadariAcl, action MadariCreateAction) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    dataBuffer := C.CBytes(data)
    defer C.free(dataBuffer)

    madariAcls := newMadariAclBuf(acls)
    defer deleteMadariAclBuf(madariAcls, acls)

    return (int)(C.MadariFrontEndUpdateNode(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer, C.uintptr_t(uintptr(dataBuffer)), C.uint32_t(len(data)), C.uintptr_t(uintptr(madariAcls)), C.uint32_t(len(acls)), C.int(action)))
}

func (publisher DefaultMadariPublisher) RemoveNode(key string, path string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariFrontEndRemoveNode(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer, false))
}

func (publisher DefaultMadariPublisher) SetAcls(key string, path string, acls []MadariAcl, version int) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    madariAcls := newMadariAclBuf(acls)
    defer deleteMadariAclBuf(madariAcls, acls)

    return (int)(C.MadariFrontEndSetPubSubAcl(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer, C.uintptr_t(uintptr(madariAcls)), C.uint32_t(len(acls)), C.int(version)))
}

func (publisher DefaultMadariPublisher) GetAcls(key string, path string, stat *MadariStat) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    if nil != stat {
        var madariStat C.struct_MADARI_STAT
        madariStat.Aversion = C.uint32_t(stat.Aversion)
        madariStat.Ctime = C.int64_t(stat.Ctime)
        madariStat.Cversion = C.uint32_t(stat.Cversion)
        madariStat.Czxid = C.int64_t(stat.Czxid)
        madariStat.DataLength = C.uint32_t(stat.DataLength)
        madariStat.Mtime = C.int64_t(stat.Mtime)
        madariStat.Mzxid = C.int64_t(stat.Mzxid)
        madariStat.NumChildren = C.uint32_t(stat.NumChildren)
        madariStat.Pzxid = C.int64_t(stat.Pzxid)
        madariStat.Version = C.uint32_t(stat.Version)
        return (int)(C.MadariFrontEndGetPubSubAcl(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer, C.uintptr_t(uintptr(unsafe.Pointer(&madariStat)))))
    }

    return (int)(C.MadariFrontEndGetPubSubAcl(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer, C.uintptr_t(0)))
}

func (publisher DefaultMadariPublisher) Exists(key string, path string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariFrontEndExists(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer))
}

func (publisher DefaultMadariPublisher) GetData(key string, path string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariFrontEndGetData(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer))
}

func (publisher DefaultMadariPublisher) GetDataV2(key string, path string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariFrontEndGetDataV2(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer))
}

func (publisher DefaultMadariPublisher) GetChildren(key string, path string) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)
    pathBuffer := GoStringToWcharStringPtr(path)
    defer DeleteWcharBuffer(pathBuffer)

    return (int)(C.MadariFrontEndGetChildren(C.uintptr_t(publisher.session), false, keyBuffer, pathBuffer))
}

func (publisher DefaultMadariPublisher) ExecuteOperationCollection(key string, mode MadariOpsExecutionMode, ops []MadariOp, completeSynchronously bool) int {
    keyBuffer := GoStringToWcharStringPtr(key)
    defer DeleteWcharBuffer(keyBuffer)

    madariOps := newMadariOpBuf(ops)
    defer deleteMadariOpBuf(madariOps, ops)

    return (int)(C.MadariFrontEndExecuteOperationCollection(C.uintptr_t(publisher.session), false, keyBuffer, C.int(mode), C.uintptr_t(uintptr(madariOps)), C.uint32_t(len(ops)), C._Bool(completeSynchronously)))
}

func newMadariAclBuf(acls []MadariAcl) unsafe.Pointer {
    aclLen := len(acls)
    if 0 == aclLen {
        return nil
    }

    madariAcls := C.malloc(C.size_t(aclLen) * C.SizeofMadariAcl())
    madariAclPtr := uintptr(unsafe.Pointer(madariAcls))
    sizeMadariAcl := uintptr(C.SizeofMadariAcl())
    for i := 0; i < aclLen; i++ {
        madariAcl := (*C.struct_MADARI_ACL)(unsafe.Pointer(madariAclPtr))
        madariAcl.Permissions = C.int(acls[i].Permissions)
        madariAcl.Identifier.Scheme = GoStringToWcharStringPtr(acls[i].Identifier.Scheme)
        madariAcl.Identifier.Identifier = GoStringToWcharStringPtr(acls[i].Identifier.Identifier)

        madariAclPtr += sizeMadariAcl
    }
    return madariAcls
}

func deleteMadariAclBuf(madariAcls unsafe.Pointer, acls []MadariAcl) {
    if nil != madariAcls {
        madariAclPtr := uintptr(madariAcls)
        sizeMadariAcl := uintptr(C.SizeofMadariAcl())
        if 0x0 != madariAclPtr {
            for i := 0; i < len(acls); i++ {
                madariAcl := (*C.struct_MADARI_ACL)(unsafe.Pointer(madariAclPtr))
                DeleteWcharBuffer(madariAcl.Identifier.Scheme)
                DeleteWcharBuffer(madariAcl.Identifier.Identifier)
                madariAclPtr += sizeMadariAcl
            }
            C.free(madariAcls)
        }
    }
}

func newMadariOpBuf(ops []MadariOp) unsafe.Pointer {
    opLen := len(ops)
    if 0 == opLen {
        return nil
    }

    madariOps := C.malloc(C.size_t(opLen) * C.SizeofMadariOp())
    madariOpPtr := uintptr(unsafe.Pointer(madariOps))
    sizeMadariOp := uintptr(C.SizeofMadariOp())
    offsetMadariOpData := uintptr(C.OffsetMadariOpData())
    for i := 0; i < opLen; i++ {
        madariOp := (*C.struct_MADARI_OP)(unsafe.Pointer(madariOpPtr))
        madariOp.Path = GoStringToWcharStringPtr(ops[i].Path)
        madariOp.OpType = C.enum_MADARIOPCODE(ops[i].Type)
        ops[i].Data.SerializeToBuffer(unsafe.Pointer(madariOpPtr + offsetMadariOpData))
        madariOpPtr += sizeMadariOp
    }
    return madariOps
}

func deleteMadariOpBuf(madariOps unsafe.Pointer, ops []MadariOp) {
    if nil != madariOps {
        madariOpPtr := uintptr(madariOps)
        sizeMadariOp := uintptr(C.SizeofMadariOp())
        offsetMadariOpData := uintptr(C.OffsetMadariOpData())
        if 0x0 != madariOpPtr {
            for i := 0; i < len(ops); i++ {
                madariOp := (*C.struct_MADARI_OP)(unsafe.Pointer(madariOpPtr))
                DeleteWcharBuffer(madariOp.Path)
                ops[i].Data.FreeInBuffer(unsafe.Pointer(madariOpPtr + offsetMadariOpData))
                madariOpPtr += sizeMadariOp
            }
            C.free(madariOps)
        }
    }
}

func sizeofWcharPtrPtr() uintptr {
    return uintptr(C.SizeofWcharPtrPtr())
}

func sizeofMadariHealth() uintptr {
    return uintptr(C.SizeofMadariHealth())
}

func sizeofMadariAcl() uintptr {
    return uintptr(C.SizeofMadariAcl())
}

func sizeofMadariOpResult() uintptr {
    return uintptr(C.SizeofMadariOpResult())
}

func offsetMadariOpResultData() uintptr {
    return uintptr(C.OffsetMadariOpResultData())
}

func offsetMadariOpResultCreateStat() uintptr {
    return uintptr(C.OffsetMadariOpResultCreateStat())
}

func offsetMadariOpResultMoveStat() uintptr {
    return uintptr(C.OffsetMadariOpResultMoveStat())
}

func offsetMadariOpResultGetDataStat() uintptr {
    return uintptr(C.OffsetMadariOpResultGetDataStat())
}

func offsetMadariOpResultGetDataData() uintptr {
    return uintptr(C.OffsetMadariOpResultGetDataData())
}

func offsetMadariOpResultGetDataDataLength() uintptr {
    return uintptr(C.OffsetMadariOpResultGetDataDataLength())
}

func offsetMadariOpResultRunResultLength() uintptr {
    return uintptr(C.OffsetMadariOpResultRunResultLength())
}
