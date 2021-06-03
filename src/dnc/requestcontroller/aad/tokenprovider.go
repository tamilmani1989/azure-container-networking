package aad

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"dnc/requestcontroller"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/pkcs12"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
)

const (
	expirationDeltaSeconds = 60
)

type TokenProvider struct {
	cache  map[string]adal.Token
	mutex  sync.RWMutex
	logger logrus.FieldLogger
}

type TokenProviderOptions struct {
	Logger logrus.FieldLogger
}

func NewTokenProvider(opts TokenProviderOptions) *TokenProvider {
	return &TokenProvider{
		logger: opts.Logger.WithField("component", "TokenProvider"),
		cache:  make(map[string]adal.Token),
	}
}

func (t *TokenProvider) GetToken(ctx context.Context, tokenOptions requestcontroller.TokenOptions) (adal.Token, error) {
	// inject orchestratorID from context into logger
	var orcehstratorIDKey requestcontroller.ContextKey = requestcontroller.OrchestratorIDContextKey
	orchestratorID := ""
	if orchestratorIDVal := ctx.Value(orcehstratorIDKey); orchestratorIDVal != nil {
		orchestratorID = orchestratorIDVal.(string)
	}

	logger := t.logger.WithFields(logrus.Fields{
		"orchestratorID": orchestratorID,
		"serverAppID":    tokenOptions.ServerAppID,
		"serverTenantID": tokenOptions.ServerTenantID,
		"clientAppID":    tokenOptions.ClientAppID,
	})
	// check the cache to see if we already have a token
	if token, err := t.getTokenFromCache(tokenOptions); err == nil {
		// if the token is not about to expire, just return it
		if !token.WillExpireIn(expirationDeltaSeconds * time.Second) {
			logger.Info("successfully received token")
			return token, nil
		}

		// try to refresh it
		refreshToken, refreshErr := t.refreshToken(tokenOptions, token)
		if refreshErr == nil {
			// cache the refreshed token and return it
			t.putTokenInCache(tokenOptions, refreshToken)
			logger.Info("successfully received token")
			return refreshToken, nil
		}

		logger.Errorf("unable to refresh expiring token, will try to get a new one: %v", refreshErr)
	}

	// no token found in cache or couldn't refresh an expiring token, so get a new one
	logger.Info("getting new token from AAD")

	token, err := t.getTokenFromAAD(tokenOptions)
	if err != nil {
		return adal.Token{}, fmt.Errorf("unable to get token from AAD: %w", err)
	}

	//store the token in the cache
	t.putTokenInCache(tokenOptions, token)

	logger.Info("successfully received token")
	return token, nil
}

func (t *TokenProvider) getTokenFromAAD(tokenOptions requestcontroller.TokenOptions) (adal.Token, error) {
	// Get the certificate and private key from pfx file
	cert, rsaPrivateKey, err := decodePkcs12(tokenOptions.ClientCertificateKeyPair, "")
	if err != nil {
		return adal.Token{}, fmt.Errorf("failed to decode pkcs12 certificate while creating spt: %w", err)
	}

	oauthConfig, err := t.getOAuthConfig(tokenOptions)
	if err != nil {
		return adal.Token{}, fmt.Errorf("unable to get oauth config %w", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromCertificate(
		*oauthConfig,
		tokenOptions.ClientAppID,
		cert,
		rsaPrivateKey,
		tokenOptions.ServerAppID,
	)
	if err != nil {
		return adal.Token{}, fmt.Errorf("failed to create service principal token using cert: %s", err)
	}

	err = spt.Refresh()
	if err != nil {
		return adal.Token{}, err
	}

	return spt.Token(), nil
}

func (t *TokenProvider) refreshToken(tokenOptions requestcontroller.TokenOptions, token adal.Token) (adal.Token, error) {
	oauthConfig, err := t.getOAuthConfig(tokenOptions)
	if err != nil {
		return adal.Token{}, fmt.Errorf("unable to get oauth config %w", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(
		*oauthConfig,
		tokenOptions.ClientAppID,
		tokenOptions.ServerAppID,
		token,
	)

	if err != nil {
		return adal.Token{}, err
	}

	err = spt.Refresh()
	if err != nil {
		return adal.Token{}, err
	}

	return spt.Token(), nil
}

func (t *TokenProvider) getOAuthConfig(tokenOptions requestcontroller.TokenOptions) (*adal.OAuthConfig, error) {
	environment, err := azure.EnvironmentFromName(tokenOptions.AzureEnvironment)
	if err != nil {
		return nil, fmt.Errorf("failed to get azure environment: %s", err)
	}

	return adal.NewOAuthConfigWithAPIVersion(environment.ActiveDirectoryEndpoint, tokenOptions.ServerTenantID, nil)
}

func (t *TokenProvider) getTokenFromCache(tokenOptions requestcontroller.TokenOptions) (adal.Token, error) {
	key := getCacheKeyForToken(tokenOptions)
	// take a read lock on the cache
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if token, found := t.cache[key]; found {
		return token, nil
	}

	return adal.Token{}, errors.New("token not found")
}

func (t *TokenProvider) putTokenInCache(tokenOptions requestcontroller.TokenOptions, token adal.Token) {
	key := getCacheKeyForToken(tokenOptions)
	// take a lock on the cache
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.cache[key] = token
}

func getCacheKeyForToken(tokenOptions requestcontroller.TokenOptions) string {
	return fmt.Sprintf("clientID:%s_serverID:%s_serverTenantID:%s", tokenOptions.ClientAppID, tokenOptions.ServerAppID, tokenOptions.ServerTenantID)
}

// The following helper functions were taken from https://github.com/Azure/kubelogin

//pem block types
const (
	certificate = "CERTIFICATE"
	privateKey  = "PRIVATE KEY"
)

func decodePkcs12(pkcs []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	blocks, err := pkcs12.ToPEM(pkcs, password)
	if err != nil {
		return nil, nil, err
	}

	var (
		pemData []byte
	)

	for _, b := range blocks {
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}

	return parseKeyPairFromPEMBlock(pemData)
}

func parseKeyPairFromPEMBlock(pemBlock []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, keyPEM := splitPEMBlock(pemBlock)

	privateKey, err := parseRsaPrivateKey(keyPEM)
	if err != nil {
		return nil, nil, err
	}

	found := false
	var cert *x509.Certificate
	for {
		var certBlock *pem.Block
		var err error
		certBlock, certPEM = pem.Decode(certPEM)
		if certBlock == nil {
			break
		}

		cert, err = x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse certificate. %w", err)
		}

		certPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if ok {
			if isPublicKeyEqual(certPublicKey, &privateKey.PublicKey) {
				found = true
				break
			}
		}
	}

	if !found {
		return nil, nil, fmt.Errorf("unable to find a matching public certificate")
	}

	return cert, privateKey, nil
}

func splitPEMBlock(pemBlock []byte) (certPEM []byte, keyPEM []byte) {
	for {
		var derBlock *pem.Block
		derBlock, pemBlock = pem.Decode(pemBlock)
		if derBlock == nil {
			break
		}
		if derBlock.Type == certificate {
			certPEM = append(certPEM, pem.EncodeToMemory(derBlock)...)
		} else if derBlock.Type == privateKey {
			keyPEM = append(keyPEM, pem.EncodeToMemory(derBlock)...)
		}
	}

	return certPEM, keyPEM
}

func parseRsaPrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode a pem block from private key")
	}

	privatePkcs1Key, errPkcs1 := x509.ParsePKCS1PrivateKey(block.Bytes)
	if errPkcs1 == nil {
		return privatePkcs1Key, nil
	}

	privatePkcs8Key, errPkcs8 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if errPkcs8 == nil {
		privatePkcs8RsaKey, ok := privatePkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("pkcs8 contained non-RSA key. Expected RSA key")
		}
		return privatePkcs8RsaKey, nil
	}

	return nil, fmt.Errorf("failed to parse private key as Pkcs#1 or Pkcs#8. (%s). (%s)", errPkcs1, errPkcs8)
}

func isPublicKeyEqual(key1, key2 *rsa.PublicKey) bool {
	if key1.N == nil || key2.N == nil {
		return false
	}
	return key1.E == key2.E && key1.N.Cmp(key2.N) == 0
}
