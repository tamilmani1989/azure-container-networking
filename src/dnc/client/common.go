package client

import (
	"encoding/json"
	"fmt"
)

// dnctl actions
const (
	Register   = "Register"
	Unregister = "Unregister"
	Get        = "Get"
	Delete     = "Delete"
	Promote    = "Promote"
)

// dnc resources
const (
	AzStorage        = "Azure Storage"
	Node             = "Node"
	Nodes            = "Nodes"
	Network          = "Network"
	Subnet           = "Subnet"
	Reservation      = "Reservation"
	NetworkContainer = "NetworkContainer"
)

func PrettyPrint(obj interface{}) {
	b, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Print(string(b))
}
