// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/configuration"
	"fmt"
	"testing"
)

func TestInit(t *testing.T) {
	var (
		got  error
		want error
	)

	t.Logf("TestInit: Pubsub count 0")
	settings := configuration.PubSubProxySettings{
		NumPubSubs: 0,
	}
	want = fmt.Errorf("Invalid number of pubsubs: 0")

	got = Init(settings, nil)
	if got.Error() != want.Error() {
		t.Fatalf("Failed with error: %v", got)
	}

	t.Logf("TestInit: Mismatch between pubsub count and vip list")
	settings = configuration.PubSubProxySettings{
		NumPubSubs: 3,
		PubsubVips: "52.225.185.199|52.138.65.29",
	}
	want = fmt.Errorf("Mismatch between number of pubsubs: 3 and VIP list count: 2")

	got = Init(settings, nil)
	if got.Error() != want.Error() {
		t.Fatalf("Failed with error: %v", got)
	}
}
