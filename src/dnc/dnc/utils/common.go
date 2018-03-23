package utils

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
)

// onErrorFail prints a failure message and exits the program if err is not nil.
func OnErrorFail(err error, message string) {
	if err != nil {
		fmt.Printf("%s: %s\n", message, err)
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
func TracePrefix() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	return fmt.Sprintf("File:%s, Function:%s, Line:%d \n", frame.File, frame.Function, frame.Line)
}
