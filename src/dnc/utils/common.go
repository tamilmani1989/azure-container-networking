package utils

import (
	"dnc/logger/loggershim"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Uint16s []uint16

func (s Uint16s) Len() int           { return len(s) }
func (s Uint16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Sort is a convenience method.

func (s Uint16s) Sort() {
	sort.Sort(s)
}

// onErrorFail prints a failure message and exits the program if err is not nil.
func OnErrorFail(err error, message string) {
	if err != nil {
		loggershim.LS.Errorf("%s: %s\n", message, err)
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
func RecordTime(start time.Time, ctx string) {
	elapsed := time.Since(start)
	loggershim.LS.Debugf("[%s] Process time: %s", ctx, elapsed)
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

	loggershim.LS.Debugf("DecodeResponse received response: %+v", response)

	if w.Body != nil {
		err = json.NewDecoder(w.Body).Decode(&response)
		loggershim.LS.Debugf("Decoded response:%v", response)
	} else {
		err = fmt.Errorf("Response body is empty")
	}

	if w.StatusCode != http.StatusOK {
		err = fmt.Errorf("Request failed with HTTP error %d", w.StatusCode)
	}

	return err
}
