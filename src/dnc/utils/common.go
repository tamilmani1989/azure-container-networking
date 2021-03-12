package utils

import (
	"dnc/logger/loggershim"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/google/uuid"
)

const (
	StorageTableApiVersion = "2019-07-07"
	MaxContainersPerNode   = 10
)

var (
	Version = "no version provided"

	// CosmosDbEndpoints - Endpoint Suffix for CosmosDb Table Storage for different clouds/env
	// For RX (USNAt) : https://microsoft.sharepoint.com/teams/LX/unrestricted/SitePages/DNS%20Endpoints%20for%20RX.aspx
	// For EX (USSEC) : https://microsoft.sharepoint.com/teams/LX/unrestricted/SitePages/DNS%20Endpoints.aspx
	// Other reference : https://github.com/Azure/go-autorest/blob/master/autorest/azure/environments.go
	CosmosDbEndpoints = map[string]string{
		"":                  "cosmos.azure.com",
		"publiccloud":       "cosmos.azure.com",
		"usgovernmentcloud": "cosmos.azure.us",
		"chinacloud":        "cosmos.azure.cn",
		"germancloud":       "cosmos.microsoftazure.de",
		"usseccloud":        "cosmos.azure.eaglex.ic.gov",
		"usnatcloud":        "cosmos.azure.microsoft.scloud",
	}

	// TableStorageEndpoints - Endpoints Suffix for Azure Table Storage for different clouds/env
	TableStorageEndpoints = map[string]string{
		"":                  "core.windows.net",
		"publiccloud":       "core.windows.net",
		"usgovernmentcloud": "core.usgovcloudapi.net",
		"chinacloud":        "core.chinacloudapi.cn",
		"germancloud":       "core.cloudapi.de",
		"usseccloud":        "core.eaglex.ic.gov",
		"usnatcloud":        "core.microsoft.scloud",
	}

	// CosmosDbAdditionalHeaders - Headers specific to cosmos db
	CosmosDbAdditionalHeaders = map[string]string{
		"MaxDataServiceVersion": "3.0;NetFx",
		"DataServiceVersion":    "3.0",
	}
)

type Uint16s []uint16

func (s Uint16s) Len() int           { return len(s) }
func (s Uint16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

const (
	LocalIPSubnetPrefixLength = 17
)

// IsValidUUID - Validate that the string passed is a UUID
func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// Sort is a convenience method.
func (s Uint16s) Sort() {
	sort.Sort(s)
}

// onErrorFail prints a failure message and exits the program if err is not nil.
func OnErrorFail(err error, message string) {
	if err != nil {
		loggershim.LS.Errorf("%s: %s", message, err)
		os.Exit(1)
	}
}

// GetLocalIP -Helper to get the local IP
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("Failed to find IP Address")
}

// PrintMapKeys - Print Keys in a map
func PrintMapKeys(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

// PrintMapKeyValues - Print Key Values in a map
func PrintMapKeyValues(m map[string]string) string {
	keyvalues := make([]string, 0, len(m))
	for k, v := range m {
		kv := k + ":" + v
		keyvalues = append(keyvalues, kv)
	}

	return "[" + strings.Join(keyvalues, ", ") + "]"
}

// PrintMapKeyValuesUint16 - Print Key Values in a map
func PrintMapKeyValuesUint16(m map[string]uint16) string {
	keyvalues := make([]string, 0, len(m))
	for k, v := range m {
		kv := fmt.Sprintf("%s:%v", k, v)
		keyvalues = append(keyvalues, kv)
	}

	return "[" + strings.Join(keyvalues, ", ") + "]"
}

// PrintMapValues - Print Values in a map
func PrintMapValues(m map[string]string) string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return "[" + strings.Join(values, ", ") + "]"
}

// TracePrefix - Helper to generate trace prefix
// skip=2 - caller's details
// skip=3 - caller of caller's details
func TracePrefix(skip int) string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	return fmt.Sprintf("In [%s:%d Function:%s]", frame.File, frame.Line, frame.Function)
}

// RecordTime - Helper to generate processing time.
// may be used with TracePrefix i.e. defer RecordTime(time.Now(), TracePrefix(2))
func RecordTime(start time.Time, ctx, ncID string) {
	elapsed := time.Since(start)
	loggershim.LS.Debugf("[%s] Process time: %s for NC: %s", ctx, elapsed, ncID)
}

func AreSetsEqual(set1, set2 []string) bool {
	if set1 == nil {
		set1 = make([]string, 0)
	}

	if set2 == nil {
		set2 = make([]string, 0)
	}

	if len(set1) != len(set2) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(set1))
	for _, item := range set1 {
		diff[item]++
	}
	for _, item := range set2 {
		if _, ok := diff[item]; !ok {
			return false
		}
		diff[item]--
		if diff[item] == 0 {
			delete(diff, item)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

func DecodeResponse(w *http.Response, response interface{}) error {
	var err error

	if w.Body != nil {
		err = json.NewDecoder(w.Body).Decode(&response)
	} else {
		err = fmt.Errorf("Response body is empty")
	}

	return err
}

// GetStorageClient - Gets the Storage Client
func GetStorageClient(accountName string, accountKey string, envName string, isUsingCosmosDb bool) (storage.Client, error) {
	var (
		storageClient  storage.Client
		err            error
		endpointSuffix string
		endpointExists bool
	)

	lowerEnvName := strings.ToLower(envName)
	if isUsingCosmosDb {
		endpointSuffix, endpointExists = CosmosDbEndpoints[lowerEnvName]
	} else {
		endpointSuffix, endpointExists = TableStorageEndpoints[lowerEnvName]
	}

	if !endpointExists {
		return storageClient, fmt.Errorf("Storage Base Endpoint with UsingCosmosDb:%v for Env:'%s' not found", isUsingCosmosDb, envName)
	}

	if isUsingCosmosDb {
		storageClient, err = storage.NewCosmosClient(accountName, accountKey, endpointSuffix, StorageTableApiVersion, true)
	} else {
		storageClient, err = storage.NewClient(accountName, accountKey, endpointSuffix, StorageTableApiVersion, true)
	}

	// CosmosDb Specific headers
	if isUsingCosmosDb && err == nil {
		storageClient.AddAdditionalHeaders(CosmosDbAdditionalHeaders)
	}

	return storageClient, err
}

// PanicRecover try to recover from crash and proceed
// recover() must be called directly from defer function.
// Don't nested it in an anonymous function.
func PanicRecover() {
	if r := recover(); r != nil {
		loggershim.LS.Errorf("[Azure DNC] Recovered from crash due to %+v", r)
		loggershim.LS.Errorf("[Azure DNC] Crash stack trace: %s", string(debug.Stack()))
	}
}

// PanicRecoverAndExitWithStackTrace - recovery from panic, print a failure message and stack trace and exit the program
func PanicRecoverAndExitWithStackTrace(context string) {
	if r := recover(); r != nil {
		loggershim.LS.Errorf("[%s]. %+v", context, r)
		loggershim.LS.Errorf(
			"[%s]. Stack trace: %s", context, string(debug.Stack()))
		os.Exit(1)
	}
}

// Remove hyphen from string
func DehyphenString(guid string) string {
	parts := strings.Split(guid, "-")
	return strings.Join(parts, "")
}

// Reverse byte array
func ReverseByteArray(s []byte) []byte {
	a := make([]byte, len(s))
	copy(a, s)

	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}