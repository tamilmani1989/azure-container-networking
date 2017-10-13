// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package common

import (
	"fmt"
	"net"
	"os/exec"

	"azure-overlay/log"
)

// LogNetworkInterfaces logs the host's network interfaces in the default namespace.
func LogNetworkInterfaces() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Failed to query network interfaces, err:%v", err)
		return
	}

	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		log.Printf("[net] Network interface: %+v with IP addresses: %+v", iface, addrs)
	}
}
func System(cmd string, arg ...string) error {
	out, err := exec.Command(cmd, arg...).Output()
	if err != nil {
		log.Printf("error:%v", err)
	}
	fmt.Println(string(out))
	return err
}
