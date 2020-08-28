package activedirectory

import (
	"context"
	"dnc/retry"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
)

const (
	msiDelay       = time.Second
	msiTimeout     = time.Minute
	msiMaxAttempts = 10
)

func GetARMOAuthTokenViaMSI(ctx context.Context, armEndpoint, msiResourceID string) (string, error) {
	msiEndpoint, _ := adal.GetMSIVMEndpoint() // error is always nil

	// this error is only validation, not network, so no need to retry
	spt, err := adal.NewServicePrincipalTokenFromMSIWithIdentityResourceID(msiEndpoint, armEndpoint, msiResourceID)
	if err != nil {
		return "", fmt.Errorf("could not build a token provider: %v", err)
	}

	retrier := retry.Retrier{Attempts: msiMaxAttempts, Delay: msiDelay}
	var oauthToken string

	ctx, cancel := context.WithTimeout(ctx, msiTimeout)
	defer cancel()

	err = retrier.Do(ctx, func() error {
		if err = spt.RefreshWithContext(ctx); err != nil {
			return fmt.Errorf("could not fetch a token: %v", err)
		}
		token := spt.Token()
		if token.IsZero() {
			return fmt.Errorf("zero token for resource id %s, msi endpoint %s", msiResourceID, msiEndpoint)

		}
		oauthToken = token.OAuthToken()
		return nil
	})

	if err != nil {
		return "", err
	}
	return oauthToken, nil
}
