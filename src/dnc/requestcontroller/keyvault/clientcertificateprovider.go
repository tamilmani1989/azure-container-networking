// Copyright 2020 Microsoft. All rights reserved.

package keyvault

import (
	"context"
	"io/ioutil"

	ad "dnc/activedirectory"

	"github.com/Azure/go-autorest/autorest/azure"
)

type ClientCertificateProvider struct {
	MSIResourceID    string
	AzureEnvironment string
	KeyvaultName     string
}

func (c ClientCertificateProvider) FetchAndStoreLatestPKCS12Secret(ctx context.Context, secretName, certPath string) error {
	tokenFetcher := &ad.MSITokenFetcher{ResourceID: c.MSIResourceID}
	azureEnvironment, err := azure.EnvironmentFromName(c.AzureEnvironment)
	if err != nil {
		return err
	}

	keyvaultClient, err := ad.NewKeyvaultClient(tokenFetcher, azureEnvironment, c.KeyvaultName)
	if err != nil {
		return err
	}

	certBytes, err := keyvaultClient.GetLatestSecretPKCS12(ctx, secretName)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(certPath, certBytes, 0644) // TODO: what file permissions should be used?
}
