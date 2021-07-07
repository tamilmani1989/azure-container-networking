package main

import (
	"dnc/test/macallocatormock"
	"flag"

	"github.com/sirupsen/logrus"
)

const defaultAddr = "0.0.0.0:5012"

func main() {
	addrFlag := flag.String("address", defaultAddr, "host:port")
	flag.Parse()

	if err := macallocatormock.NewServer(*addrFlag).Start(); err != nil {
		logrus.Fatalf("could not start macallocatormock server: %v", err)
	}
}
