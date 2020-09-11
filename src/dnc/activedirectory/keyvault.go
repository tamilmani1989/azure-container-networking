package activedirectory

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"
	"strings"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	keyvaultRegex = `^[-A-Za-z0-9]+$`
	blockType     = "CERTIFICATE"
)

type KeyvaultClient struct {
	// the name of the Azure Key Vault instance
	KeyvaultName string
	// the type of azure cloud based on azure go sdk
	AzureCloudEnvironment azure.Environment
	// base keyvault client used to get certificates
	Client kv.BaseClient
	// url of the keyvault
	vaultURL string
}

func NewKeyvaultClient(tf TokenFetcher, env azure.Environment, keyvaultName string) (*KeyvaultClient, error) {
	if keyvaultName == "" {
		return nil, errors.New("keyvault name not specified")
	}

	vaultURL, err := getVaultURL(keyvaultName, env.Name)
	if err != nil {
		return nil, err
	}

	// keyvault resource must have no trailing slash
	kvEpt := strings.TrimSuffix(env.KeyVaultEndpoint, "/")
	spt, err := tf.GetServicePrincipalToken(kvEpt)
	if err != nil {
		return nil, err
	}

	baseKVClient := kv.New()
	baseKVClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &KeyvaultClient{
		KeyvaultName:          keyvaultName,
		AzureCloudEnvironment: env,
		Client:                baseKVClient,
		vaultURL:              vaultURL,
	}, nil
}

// GetCertificate returns a PEM encoded certificate for the given cert name and version
func (kvc *KeyvaultClient) GetCertificate(ctx context.Context, certName, certVersion string) ([]byte, error) {
	return kvc.getCertificate(ctx, certName, certVersion)
}

// GetLatestCertificate returns a PEM encoded certificate for the latest cert version
func (kvc *KeyvaultClient) GetLatestCertificate(ctx context.Context, certName string) ([]byte, error) {
	// if a version is not specified, the latest certificate is returned
	// https://docs.microsoft.com/en-us/rest/api/keyvault/getcertificate/getcertificate
	return kvc.getCertificate(ctx, certName, "")
}

func (kvc *KeyvaultClient) getCertificate(ctx context.Context, certName, certVersion string) ([]byte, error) {
	certBundle, err := kvc.Client.GetCertificate(ctx, kvc.vaultURL, certName, certVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %v", err)
	}

	certBlock := &pem.Block{
		Type:  blockType,
		Bytes: *certBundle.Cer,
	}

	return pem.EncodeToMemory(certBlock), nil
}

var keyvaultRegexp = regexp.MustCompile(keyvaultRegex)

func getVaultURL(keyvaultName, cloudEnv string) (string, error) {
	// Keyvault name must be a 3-24 character string
	if len(keyvaultName) < minKeyVaultName || len(keyvaultName) > maxKeyVaultName {
		return "", fmt.Errorf("invalid vault name: '%s', must be between 3 and 24 chars", keyvaultName)
	}

	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	if !keyvaultRegexp.MatchString(keyvaultName) {
		return "", fmt.Errorf("invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", keyvaultName)
	}

	vaultDNSSuffix, err := getVaultDNSSuffix(cloudEnv)
	if err != nil {
		return "", err
	}

	vaultURL := fmt.Sprintf("https://%s.%s/", keyvaultName, vaultDNSSuffix)
	return vaultURL, nil
}
