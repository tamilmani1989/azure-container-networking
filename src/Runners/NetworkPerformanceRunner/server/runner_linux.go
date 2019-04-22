package main

import (
	"encoding/json"
	"log"
	"math"
	"strconv"
	"strings"
)

func parseLogs(output []byte, report *NetworkPerformanceResults) (err error) {
	var iperf3Json Iperf3JSON
	log.Printf("Before parsing logs...")
	splitOutput := strings.Split(string(output), "PING")
	if len(splitOutput) > 0 {
		if err = json.Unmarshal([]byte(splitOutput[0]), &iperf3Json); err != nil {
			log.Printf("unmarshal error: %+v", err)
		} else {
			log.Printf("Iperf error: %+v", iperf3Json.Error)
			report.Throughput = int64(iperf3Json.End.SumReceived.Throughput)
			report.CPUSender = int(math.Round(iperf3Json.End.CPUUtilization.Sender))
			report.CPUReceiver = int(math.Round(iperf3Json.End.CPUUtilization.Receiver))
		}
	}

	latencyLine := compiledLatencyRegex.FindString(string(output))
	if len(latencyLine) > 0 {
		log.Printf("%+v", latencyLine)
		// split on spaces from -> min/avg/max/mdev = 0.532/1.047/7.639/1.030
		// and then split on the / in -> 0.532/1.047/7.639/1.030
		latency := strings.Split(strings.Split(latencyLine, " ")[2], "/")
		min, _ := strconv.ParseFloat(latency[0], 64)
		avg, _ := strconv.ParseFloat(latency[1], 64)
		max, _ := strconv.ParseFloat(latency[2], 64)
		// store latency as microseconds
		report.LatencyMin = int(min * 1000)
		report.LatencyAvg = int(avg * 1000)
		report.LatencyMax = int(max * 1000)
	}

	log.Printf("After parsing logs...")
	return
}
