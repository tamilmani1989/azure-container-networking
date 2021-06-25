package dns

import (
	"context"
	"sort"
	"strings"
	"testing"

	"dnc/configuration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointDNSLookup(t *testing.T) {
	endpoint := "eastus-prod.sdnpubsub.core.windows.net"

	// Expected values //reddog/builds/branches/dcmt_latest/Configuration/CloudSettings/Settings_LS_Prod.xml
	pubsubvip1 := configuration.PubsubVips{
		AzId: "AZ01",
		VIPs: "13.90.84.247,52.170.152.22,52.170.152.21",
	}
	pubsubvip2 := configuration.PubsubVips{
		AzId: "AZ02",
		VIPs: "13.90.156.100,40.88.27.36,40.88.29.75",
	}
	pubsubvip3 := configuration.PubsubVips{
		AzId: "AZ03",
		VIPs: "52.253.160.61,52.253.160.60",
	}
	expectedPubsubVips := []configuration.PubsubVips{
		pubsubvip1,
		pubsubvip2,
		pubsubvip3,
	}

	translatedEndpoint := TranslateEndpoint(endpoint)
	require.Equal(t, "useast-prod.sdnpubsub.core.windows.net", translatedEndpoint)

	pubsubvips, err := PubsubVipsFromDNS(context.Background(), translatedEndpoint)
	require.Nil(t, err)
	require.NotNil(t, pubsubvips)
	assert.Equal(t, len(pubsubvips), len(expectedPubsubVips))
	for i, v := range pubsubvips {
		assert.Equal(t, v.AzId, expectedPubsubVips[i].AzId)
		assert.True(t, unsortedEqual(v.VIPs, expectedPubsubVips[i].VIPs))
	}
}

func TestEndpointDNSLookupWithAlreadyTranslated(t *testing.T) {
	endpoint := "uswest2-prod.sdnpubsub.core.windows.net"

	pubsubvip1 := configuration.PubsubVips{
		AzId: "AZ01",
		VIPs: "52.175.233.151,52.183.114.19,52.175.236.80",
	}
	pubsubvip2 := configuration.PubsubVips{
		AzId: "AZ02",
		VIPs: "52.143.211.60",
	}
	pubsubvip3 := configuration.PubsubVips{
		AzId: "AZ03",
		VIPs: "40.65.112.111,52.151.40.224,52.151.61.1",
	}
	expectedPubsubVips := []configuration.PubsubVips{
		pubsubvip1,
		pubsubvip2,
		pubsubvip3,
	}

	translatedEndpoint := TranslateEndpoint(endpoint)
	require.Equal(t, "uswest2-prod.sdnpubsub.core.windows.net", translatedEndpoint)

	pubsubvips, err := PubsubVipsFromDNS(context.Background(), translatedEndpoint)
	require.Nil(t, err)
	require.NotNil(t, pubsubvips)
	assert.Equal(t, len(pubsubvips), len(expectedPubsubVips))
	for i, v := range pubsubvips {
		assert.Equal(t, v.AzId, expectedPubsubVips[i].AzId)
		assert.True(t, unsortedEqual(v.VIPs, expectedPubsubVips[i].VIPs))
	}
}

func TestEndpointDNSInvalidEndpointLocation(t *testing.T) {
	endpoint := "jupiter-prod.sdnpubsub.core.windows.net"

	translatedEndpoint := TranslateEndpoint(endpoint)
	require.Equal(t, "jupiter-prod.sdnpubsub.core.windows.net", translatedEndpoint)

	pubsubvips, err := PubsubVipsFromDNS(context.Background(), translatedEndpoint)
	require.NotNil(t, err)
	require.Nil(t, pubsubvips)
}

func TestEndpointTranslation(t *testing.T) {
	endpoint := "southcentralus-prod.sdnpubsub.core.windows.net"

	translatedEndpoint := TranslateEndpoint(endpoint)
	require.Equal(t, "ussouth-prod.sdnpubsub.core.windows.net", translatedEndpoint)
}

// Useful for getting every possible region's pubsubvips at once
type pubsubvipsHolder struct {
	region     string
	pubsubvips []configuration.PubsubVips
}

func TestAllKnownEndpoints(t *testing.T) {
	holder := make([]pubsubvipsHolder, 0)
	for region := range regionTranslations {
		translatedEndpoint := TranslateEndpoint(region + "-prod.sdnpubsub.core.windows.net")

		pubsubvips, err := PubsubVipsFromDNS(context.Background(), translatedEndpoint)
		require.Nil(t, err)

		holder = append(holder, pubsubvipsHolder{region: region, pubsubvips: pubsubvips})
	}
	require.EqualValues(t, len(holder), len(regionTranslations))
}

// There is no guarantee which order the VIPs come back in, so
// 52.253.160.60,52.253.160.61 is effectively equal to 52.253.160.61,52.253.160.60
func unsortedEqual(received, expected string) bool {
	if len(received) != len(expected) {
		return false
	}

	a := strings.Split(received, "")
	b := strings.Split(received, "")
	sort.Strings(a)
	sort.Strings(b)

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
