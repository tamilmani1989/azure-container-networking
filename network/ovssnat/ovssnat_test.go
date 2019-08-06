package ovssnat

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestAllowInboundFromHostToNC(t *testing.T) {
	client := &OVSSnatClient{
		snatBridgeIP:          "169.254.0.1/16",
		localIP:               "169.254.0.4/16",
		containerSnatVethName: "eth0",
	}

	if err := client.AllowInboundFromHostToNC(); err != nil {
		t.Errorf("Error adding inbound rule: %v", err)
	}

	if err := client.AllowInboundFromHostToNC(); err != nil {
		t.Errorf("Error adding existing inbound rule: %v", err)
	}

	if err := client.DeleteInboundFromHostToNC(); err != nil {
		t.Errorf("Error removing inbound rule: %v", err)
	}
}
