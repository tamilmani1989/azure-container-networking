package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func triggerDatapathTests(w http.ResponseWriter) (err error) {
	report := DncDataPath{
		ServiceTunnel: false,
		Nic2Nic:       false,
		Nic2VM:        false,
		Vnet2VnetGW:   false,
		LastError:     ""}

	// wait for previous container to finish deletion (up to 2 minutes)
	for start := time.Now(); time.Since(start) < time.Minute*2; time.Sleep(5 * time.Second) {
		output, _ := exec.Command("kubectl", "get", "pod", "-l", "app=container1", "-o", "jsonpath=\"{.items[0].metadata.name}\"").CombinedOutput()
		if result := string(output); !strings.Contains(strings.ToLower(result), "error") {
			break
		}
	}

	// create container and run datapath tests
	if err = createnc(false); err == nil {
		for i := 0; i < 2 && !report.Healthy(); time.Sleep(15 * time.Second) {
			// sleep and wait for pod creation
			time.Sleep(15 * time.Second)
			// get container ID and remove wrapping quotes
			output, _ := exec.Command("kubectl", "get", "pod", "-l", "app=container1", "-o", "jsonpath=\"{.items[0].metadata.name}\"").CombinedOutput()
			containerID := string(output[1 : len(output)-1])
			// wait for container to become ready (up to 2 minutes)
			for start := time.Now(); time.Since(start) < time.Minute*2; time.Sleep(5 * time.Second) {
				output, _ = exec.Command("kubectl", "get", "pods", containerID, "-o", "jsonpath=\"{.status.phase}\"").CombinedOutput()
				if result := string(output[1 : len(output)-1]); result == running {
					break
				}
			}
			// test storage endpoint (service tunneling)
			_, err = exec.Command("kubectl", "exec", containerID, "--", "wget", blobEP).CombinedOutput()
			if err == nil {
				report.ServiceTunnel = true
			}
			// test nic to nic connection
			_, err = exec.Command("kubectl", "exec", containerID, "--", "ping", "-n", "3", "10.0.1.4").CombinedOutput()
			if err == nil {
				report.Nic2Nic = true
			}
			// test nic to vm connection
			_, err = exec.Command("kubectl", "exec", containerID, "--", "ping", "-n", "3", "10.0.0.4").CombinedOutput()
			if err == nil {
				report.Nic2VM = true
			}
			// test gateway (vnet to vnet)
			_, err = exec.Command("kubectl", "exec", containerID, "--", "ping", "-n", "3", "20.0.0.4").CombinedOutput()
			if err == nil {
				report.Vnet2VnetGW = true
			}
			// test peering (vnet to vnet)
			_, err = exec.Command("kubectl", "exec", containerID, "--", "ping", "-n", "3", "30.0.0.4").CombinedOutput()
			if err == nil {
				report.Vnet2VnetPeering = true
			}
		}
	}

	report.LastError = err.Error()
	// create json report
	jsonData, err := json.Marshal(report)
	if err == nil {
		jsonFile, err := os.Create("./datapathReport.json")
		if err == nil {
			jsonFile.Write(jsonData)
			jsonFile.Close()
		}
	}

	// delete container if all tests have passed
	if report.Healthy() {
		deletenc(false)
	}

	testLock.Lock()
	testRunning = false
	testLock.Unlock()

	log.Printf("%+v", err)
	return
}
