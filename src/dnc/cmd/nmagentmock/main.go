package main

import (
	"dnc/test/nmagentmock"
	"flag"

	"github.com/sirupsen/logrus"
)

const defaultAddr = "localhost:9001"

func main() {
	addrFlag := flag.String("address", defaultAddr, "host:port")
	flag.Parse()

	if err := nmagentmock.NewServer(*addrFlag).Start(); err != nil {
		logrus.Fatalf("could not start nmagentmock server: %v", err)
	}
}
