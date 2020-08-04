package activedirectory

import (
	"context"
	"encoding/pem"
	"fmt"
	"regexp"
	"strings"
	"time"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	keyvaultRegex = `^[-A-Za-z0-9]+$`
)

type KeyvaultClient struct {
	// the identiy Manager that
	IdentityManager IdentityManager
	// the name of the Azure Key Vault instance
	KeyvaultName string
	// the type of azure cloud based on azure go sdk
	AzureCloudEnvironment azure.Environment
	// base keyvault client used to get certificates
	Client kv.BaseClient
	// url of the keyvault
	vaultURL string
}

func NewKeyvaultClient(im IdentityManager, keyvaultName string) (KeyvaultClient, error) {
	var kvc KeyvaultClient
	kvc.KeyvaultName = keyvaultName
	kvc.IdentityManager = im
	kvc.AzureCloudEnvironment = azure.PublicCloud
	kvc.Client = kv.New()

	if kvc.KeyvaultName == "" {
		return kvc, fmt.Errorf("Keyvault name not specified")
	}

	authorizer, err := kvc.GetKeyvaultAuthorizer()
	if err != nil {
		return kvc, err
	}

	kvc.Client.Authorizer = authorizer

	err = kvc.setVaultURL()
	if err != nil {
		return kvc, err
	}

	return kvc, err
}

func (kvc *KeyvaultClient) setVaultURL() (err error) {
	// Key Vault name must be a 3-24 character string
	if len(kvc.KeyvaultName) < minKeyVaultName || len(kvc.KeyvaultName) > maxKeyVaultName {
		return fmt.Errorf("Invalid vault name: %q, must be between 3 and 24 chars", kvc.KeyvaultName)
	}

	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	isValid := regexp.MustCompile(keyvaultRegex).MatchString
	if !isValid(kvc.KeyvaultName) {
		return fmt.Errorf("Invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", kvc.KeyvaultName)
	}

	vaultDNSSuffix, err := getVaultDNSSuffix(kvc.AzureCloudEnvironment.Name)
	if err != nil {
		return err
	}

	vaultDNSSuffixValue := vaultDNSSuffix
	vaultURI := "https://" + kvc.KeyvaultName + "." + vaultDNSSuffixValue + "/"
	kvc.vaultURL = vaultURI
	return nil
}

// GetKeyvaultAuthorizer retrieves a new authorizer to access keyvault
func (kvc *KeyvaultClient) GetKeyvaultAuthorizer() (authorizer autorest.Authorizer, err error) {
	kvEndPoint := strings.TrimSuffix(kvc.AzureCloudEnvironment.KeyVaultEndpoint, "/")

	servicePrincipalToken, err := kvc.IdentityManager.GetServicePrincipalToken(kvEndPoint)
	if err != nil {
		return nil, err
	}

	authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	return authorizer, nil
}

func (kvc *KeyvaultClient) getLatestCertificateVersion(ctx context.Context, certificateName string) (string, error) {
	// List certificate versions
	list, err := kvc.Client.GetCertificateVersionsComplete(ctx, kvc.vaultURL, certificateName, nil)
	if err != nil {
		return "", err
	}

	// Iterate through the list and get the last version
	var lastItemDate time.Time
	var lastItemVersion string
	for list.NotDone() {
		item := list.Value()

		// Filter only enabled items
		if item.Attributes != nil && item.Attributes.Enabled != nil && *item.Attributes.Enabled {

			// Get the most recent element
			updatedTime := time.Time(*item.Attributes.Updated)
			if lastItemDate.IsZero() || updatedTime.After(lastItemDate) {
				lastItemDate = updatedTime

				// Get the ID
				parts := strings.Split(*item.ID, "/")
				lastItemVersion = parts[len(parts)-1]
			}
		}

		// Iterate to next
		err = list.NextWithContext(ctx)
		if err != nil {
			return "", err
		}
	}

	return lastItemVersion, nil
}

func (kvc *KeyvaultClient) GetCertificate(ctx context.Context, certname string) (cert []byte, err error) {

	// for object type "cert" the certificate is written to the file in PEM format
	if certname == "" {
		err = fmt.Errorf("Keyvault Certificate name not specified")
		return
	}

	certversion, err := kvc.getLatestCertificateVersion(ctx, certname)
	if err != nil {
		err = fmt.Errorf("Failed to get latest certificate version with err: %v", err)
		return
	}

	certbundle, err := kvc.Client.GetCertificate(ctx, kvc.vaultURL, certname, certversion)
	if err != nil {
		err = fmt.Errorf("Failed to get certificate with err : %v", err)
		return
	}

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: *certbundle.Cer,
	}

	cert = pem.EncodeToMemory(certBlock)
	return
}
