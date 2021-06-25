package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"dnc/configuration"
)

// The dns package relies on information provided by //reddog/builds/branches/dcmt_latest/Configuration/CloudSettings/Settings_LS_Prod.xml
// It contains the information needed to provide the ARM -> DCMT region translations and the number of AZs per region
// In addition to the AZs, the specific ring information is also provided
//
//	Example 1:
//      <EnvironmentSetting Name="Region" Value="germanywestcentral" />
//		<EnvironmentSetting Name="PubSub_REGION_NAME" Value="germanywc" />
//		<EnvironmentSetting Name="SdnPubSubVips" Value="AZ01=52.253.169.60,52.253.169.61;
//		AZ01-02=51.116.168.50,51.116.239.211;AZ02=52.253.170.60,52.253.170.61;AZ03=52.253.171.60,52.253.171.61" />
//
//		Primary ring endpoints:
//			germanywc01-prod.sdnpubsub.core.windows.net
//			germanywc02-prod.sdnpubsub.core.windows.net
//			germanywc03-prod.sdnpubsub.core.windows.net
//		Secondary ring endpoints:
//			germanywc01R02-prod.sdnpubsub.core.windows.net
//
//	Example 2:
//		<EnvironmentSetting Name="Region" Value="switzerlandwest" />
//		<EnvironmentSetting Name="PubSub_REGION_NAME" Value="switzerlandw" />
//		<EnvironmentSetting Name="SdnPubSubVips" Value="RG=52.253.166.60,52.253.166.61;RG-02=51.107.232.38,51.107.235.192" />
//
//		Primary ring endpoints:
//			switzerlandw-prod.sdnpubsub.core.windows.net
//		Secondary ring endpoints:
//			switzerlandwR02-prod.sdnpubsub.core.windows.net
//

// There is some values that are the same but keeping them to avoid confusion
// TODO: this is interim solution, will eventually use cname to replace this
var regionTranslations = map[string]string{
	"australiacentral":   "australiac",
	"australiacentral2":  "australiac2",
	"australiaeast":      "australiaeast",
	"australiasoutheast": "australiasoutheast",
	"brazilsouth":        "brazilsouth",
	"brazilsoutheast":    "brazilse",
	"centralus":          "uscentral",
	"canadacentral":      "canadacentral",
	"canadaeast":         "canadaeast",
	"centralindia":       "indiacentral",
	"centraluseuap":      "uscentraleuap",
	"eastasia":           "asiaeast",
	"eastus":             "useast",
	"eastus2":            "useast2",
	"eastus2euap":        "useast2euap",
	"eastusslv":          "eusslv",
	"eastusstg":          "usstagee",
	"francecentral":      "francec",
	"francesouth":        "frances",
	"germanynorth":       "germanyn",
	"germanywestcentral": "germanywc",
	"japaneast":          "japaneast",
	"japanwest":          "japanwest",
	"jioindiacentral":    "jioinc",
	"jioindiawest":       "jioinw",
	"koreacentral":       "koreacentral",
	"koreasouth":         "koreasouth",
	"northcentralus":     "usnorth",
	"northeurope":        "europenorth",
	"norwayeast":         "norwaye",
	"norwaywest":         "norwayw",
	"southafricanorth":   "southafrican",
	"southafricawest":    "southafricaw",
	"southcentralus":     "ussouth",
	"southcentralusstg":  "usstagesc",
	"southeastasia":      "asiasoutheast",
	"southindia":         "indiasouth",
	"swedencentral":      "swedenc",
	"swedensouth":        "swedens",
	"switzerlandnorth":   "switzerlandn",
	"switzerlandwest":    "switzerlandw",
	"uaecentral":         "uaec",
	"uaenorth":           "uaen",
	"uksouth":            "uksouth",
	"ukwest":             "ukwest",
	"westcentralus":      "uswestcentral",
	"westeurope":         "europewest",
	"westindia":          "indiawest",
	"westus":             "uswest",
	"westus2":            "uswest2",
	"westus3":            "uswest3",
}

// Number of AZ's per region, we use this as another method to know when to stop looping through possible endpoints
// The primaryRingAZCount values = (total AZs in region) - (non-primary ring AZs in region)
// For example useast2euap has 4 AZ (AZ01, AZ02, AZ03, AZ03-02) but only 3 in the primary ring
var primaryRingAZCount = map[string]int{
	"asiaeast":           1,
	"asiasoutheast":      3,
	"australiac":         1,
	"australiac2":        1,
	"australiaeast":      3,
	"australiasoutheast": 1,
	"brazilse":           1,
	"brazilsouth":        3,
	"canadacentral":      3,
	"canadaeast":         1,
	"europenorth":        3,
	"europewest":         3,
	"eusslv":             1,
	"francec":            3,
	"frances":            1,
	"germanyn":           1,
	"germanywc":          3,
	"indiacentral":       3,
	"indiasouth":         1,
	"indiawest":          1,
	"japaneast":          3,
	"japanwest":          1,
	"jioinc":             1,
	"jioinw":             1,
	"koreacentral":       3,
	"koreasouth":         1,
	"norwaye":            3,
	"norwayw":            1,
	"southafrican":       3,
	"southafricaw":       1,
	"swedenc":            1,
	"swedens":            1,
	"switzerlandn":       1,
	"switzerlandw":       1,
	"uaec":               1,
	"uaen":               1,
	"uksouth":            3,
	"ukwest":             1,
	"uscentral":          3,
	"uscentraleuap":      1,
	"useast2":            3,
	"useast2euap":        3,
	"useast":             3,
	"usnorth":            1,
	"ussouth":            3,
	"usstagee":           1,
	"usstagesc":          1,
	"uswestcentral":      1,
	"uswest":             1,
	"uswest2":            3,
	"uswest3":            3,
}

const (
	endpointSuffix = "-prod.sdnpubsub.core.windows.net"
	lookupTimeout  = time.Second * 15
	retryDuration  = time.Second * 3
)

// PubsubVipsFromDNS takes in a sdnpubsub endpoint (ie. <region>-prod.sdnpubsub.core.windows.net) and returns the corresponding
// PubsubVips by performing a DNS lookup for each availability zone in that region
func PubsubVipsFromDNS(ctx context.Context, connectionEndpoint string) ([]configuration.PubsubVips, error) {
	endpoint := strings.SplitN(connectionEndpoint, "-", 2)
	if len(endpoint) != 2 || len(endpoint[0]) == 0 || !strings.EqualFold("-"+endpoint[1], endpointSuffix) {
		return nil, fmt.Errorf("invalid PubSub connection endpoint given: %s", connectionEndpoint)
	}

	region := endpoint[0]
	azMax, ok := primaryRingAZCount[region]
	if !ok {
		// default to 1 for unknown (new) regions
		azMax = 1
	}

	// Loop through all AZ's and get their corresponding VIPs
	// AZ endpoint looks like: <region><AZ>-prod.sdnpubsub.core.windows.net with a padded AZ number (01, 02, ...)
	// If a DNS lookup for a subsequent AZ fails, it indicates that there are no more AZs remaining
	azNum := 1
	var pubsubvips []configuration.PubsubVips
	for azNum <= azMax {
		azNumLeadingZero := fmt.Sprintf("%02d", azNum)

		// TODO: Switch to using TXT record which will have exact number of AZs to expect

		dctx, cancel := context.WithTimeout(ctx, lookupTimeout)
		defer cancel()

		// If error received and it is not the stopping signal (IsNotFound), retry until successful or until timeout
		var dnsError *net.DNSError
		ips, err := net.LookupIP(region + azNumLeadingZero + endpointSuffix)
		for err != nil && !(errors.As(err, &dnsError) && dnsError.IsNotFound) {
			select {
			case <-dctx.Done():
				return nil, dctx.Err()
			case <-time.After(retryDuration):
				ips, err = net.LookupIP(region + azNumLeadingZero + endpointSuffix)
			}
		}

		// IsNotFound is a secondary stopping signal to know we found all AZs
		if err != nil {
			break
		}

		vips := make([]string, len(ips))
		for i := range ips {
			vips[i] = ips[i].String()
		}

		pubsubvips = append(pubsubvips, configuration.PubsubVips{AzId: "AZ" + azNumLeadingZero, VIPs: strings.Join(vips, ",")})
		azNum++
	}

	// If the first DNS lookup failed, try just the regional endpoint <region>-prod.sdnpubsub.core.windows.net
	if azNum == 1 {
		dctx, cancel := context.WithTimeout(ctx, lookupTimeout)
		defer cancel()

		ips, err := net.LookupIP(region + endpointSuffix)
		for err != nil {
			select {
			case <-dctx.Done():
				return nil, dctx.Err()
			case <-time.After(retryDuration):
				ips, err = net.LookupIP(region + endpointSuffix)
			}
		}

		vips := make([]string, len(ips))
		for i := range ips {
			vips[i] = ips[i].String()
		}

		pubsubvips = append(pubsubvips, configuration.PubsubVips{AzId: "AZ01", VIPs: strings.Join(vips, ",")})
	}

	return pubsubvips, nil
}

// TranslateEndpoint gets and translate the regional endpoint from ARM -> DCMT format
// ie. uswest2-prod.sdnpubsub.core.windows.net -> westus2-prod.sdnpubsub.core.windows.net
// if no valid translation was found (possibly already in DCMT format) will return what was given
func TranslateEndpoint(connectionEndpoint string) string {
	region := strings.TrimSuffix(connectionEndpoint, endpointSuffix)
	translated, ok := regionTranslations[region]
	if !ok {
		return connectionEndpoint
	}
	return translated + endpointSuffix
}
