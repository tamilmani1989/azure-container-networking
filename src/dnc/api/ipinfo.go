package api

import "fmt"

// IPInfo holds metadata for an IP resource
type IPInfo struct {
	IPAddress string `json:"Address"`
}

func (i *IPInfo) String() string {
	return fmt.Sprintf("IPInfo:[IPAddress:%s]", i.IPAddress)
}
