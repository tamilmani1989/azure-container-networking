package main

import (
	aznmmock "dnc/cmd/aznm/mock"
	"fmt"
)

const (
	port = 444
)

func main ()  {
	fmt.Println("starting mock aznm server ....")

	mockserver  := aznmmock.NewMockServer()
	mockserver.StartGrpcServer(port)
}