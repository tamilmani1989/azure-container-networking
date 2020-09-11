package activedirectory

import (
	"dnc/logger/loggershim"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	nmiendpoint = "http://localhost:2579/host/token/"
	// Pod Identity podnameheader
	podnameheader = "podname"
	// Pod Identity podnsheader
	podnsheader         = "podns"
	minKeyVaultName     = 3
	maxKeyVaultName     = 24
	maxPodIdentityDelay = 120 * time.Second

	AADPodNameEnv      = "POD_NAME"
	AADPodNamespaceEnv = "POD_NAMESPACE"
)

// PodIdentityManager is used to interact with Keyvault by leveraging AAD Pod Identity
type PodIdentityManager struct {
	// Tenant ID is the id of the AAD tenant where the Managed Identity + Keyvault exists
	TenantID string
	// the type of azure cloud based on azure go sdk
	AzureCloudEnvironment azure.Environment
	// the name of the pod (if using POD AAD Identity)
	PodName string
	// the namespace of the pod (if using POD AAD Identity)
	PodNamespace string
}

// NMIResponse is the response received from aad-pod-identity
type NMIResponse struct {
	Token    adal.Token `json:"token"`
	ClientID string     `json:"clientid"`
}

func NewPodIdentityManager() (*PodIdentityManager, error) {
	var im PodIdentityManager
	im.AzureCloudEnvironment = azure.PublicCloud
	im.PodName = os.Getenv(AADPodNameEnv)
	im.PodNamespace = os.Getenv(AADPodNamespaceEnv)

	if im.PodName == "" {
		return nil, fmt.Errorf("%v environmental variable not supplied", AADPodNameEnv)
	}

	if im.PodNamespace == "" {
		return nil, fmt.Errorf("%v environmental variable not supplied", AADPodNamespaceEnv)
	}

	return &im, nil
}

func (im *PodIdentityManager) GetServicePrincipalToken(resource string) (*adal.ServicePrincipalToken, error) {

	// response object from NMI
	var resp *http.Response
	var spt *adal.ServicePrincipalToken

	oauthConfig, err := adal.NewOAuthConfig(im.AzureCloudEnvironment.ActiveDirectoryEndpoint, im.TenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}

	endpoint := fmt.Sprintf("%s?resource=%s", nmiendpoint, resource)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create NMI request with err : %v", err)
	}

	req.Header.Add(podnsheader, im.PodNamespace)
	req.Header.Add(podnameheader, im.PodName)

	// Sometimes there is a delay when AAD Pod Identity recognizes DNC starts up, retry until successfull or max time
	for delay := 1 * time.Second; delay < maxPodIdentityDelay; delay *= 2 {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("Failed to read NMI response with err : %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			loggershim.LS.Errorf("NMI responded with %v, retrying in %v...\n", resp.StatusCode, delay)
			time.Sleep(delay)
		} else {
			var nmiResp = new(NMIResponse)
			err = json.Unmarshal(bodyBytes, &nmiResp)
			if err != nil {
				return nil, fmt.Errorf("NMI request failed unmarshall with err : %v", err)
			}

			token := nmiResp.Token
			clientID := nmiResp.ClientID

			if token.AccessToken == "" || clientID == "" {
				return nil, fmt.Errorf("nmi did not return expected values in response: token and clientid")
			}

			spt, err = adal.NewServicePrincipalTokenFromManualToken(*oauthConfig, clientID, resource, token, nil)
			if err != nil {
				return nil, fmt.Errorf("Failed to create service principal token from manual token with err : %v", err)
			}
		}
	}

	return spt, nil
}

// ParseAzureEnvironment returns azure environment by name
func parseAzureEnvironment(cloudName string) (azure.Environment, error) {
	if cloudName == "" {
		return azure.PublicCloud, nil
	}
	return azure.EnvironmentFromName(cloudName)
}

func getVaultDNSSuffix(cloudName string) (vaultTld string, err error) {
	environment, err := parseAzureEnvironment(cloudName)
	if err != nil {
		return vaultTld, fmt.Errorf("Failed to parse Azure environment with: %v", err)
	}

	return environment.KeyVaultDNSSuffix, nil
}
