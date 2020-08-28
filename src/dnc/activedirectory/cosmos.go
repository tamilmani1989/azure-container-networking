package activedirectory

import (
	"context"
	"dnc/retry"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	cosmosKeysRequestFmt      = "%s%s/%s/?api-version=2016-03-31" // host, resource id, operation
	cosmosProviderNamespace   = "Microsoft.DocumentDb"
	cosmosAccountResourceType = "databaseAccounts"
	cosmosRWOpType            = "listKeys"
	cosmosReadOnlyOpType      = "readonlykeys"
)

type CosmosReadOnlyKeys struct {
	PrimaryReadonlyMasterKey   string `json:"primaryReadonlyMasterKey"`
	SecondaryReadonlyMasterKey string `json:"secondaryReadonlyMasterKey"`
}

type CosmosRWKeys struct {
	PrimaryMasterKey   string `json:"primaryMasterKey"`
	SecondaryMasterKey string `json:"secondaryMasterKey"`
}

type cosmosKeys struct {
	CosmosRWKeys
	CosmosReadOnlyKeys
}

type CosmosKeyFetcher struct {
	ARMEndpoint string
}

func (c *CosmosKeyFetcher) GetReadOnlyKeys(ctx context.Context, armToken, cosmosResourceID string) (CosmosReadOnlyKeys, error) {
	k, err := c.getCosmosKeys(ctx, armToken, cosmosResourceID, true)
	if err != nil {
		return CosmosReadOnlyKeys{}, err
	}
	return CosmosReadOnlyKeys{
		PrimaryReadonlyMasterKey:   k.PrimaryReadonlyMasterKey,
		SecondaryReadonlyMasterKey: k.SecondaryReadonlyMasterKey,
	}, nil
}

func (c *CosmosKeyFetcher) GetReadWriteKeys(ctx context.Context, armToken, cosmosResourceID string) (CosmosRWKeys, error) {
	k, err := c.getCosmosKeys(ctx, armToken, cosmosResourceID, false)
	if err != nil {
		return CosmosRWKeys{}, err
	}
	return CosmosRWKeys{
		PrimaryMasterKey:   k.PrimaryMasterKey,
		SecondaryMasterKey: k.SecondaryMasterKey,
	}, nil
}

// todo: use autorest.Client with bearer authorizer? this is basically an ARM wrapper
func (c *CosmosKeyFetcher) getCosmosKeys(ctx context.Context, armToken, cosmosResourceID string, readOnly bool) (cosmosKeys, error) {
	res, err := ParseResource(cosmosResourceID)
	if err != nil {
		return cosmosKeys{}, err
	}

	if strings.ToLower(res.ProviderNamespace) != strings.ToLower(cosmosProviderNamespace) ||
		strings.ToLower(res.ResourceType) != strings.ToLower(cosmosAccountResourceType) {
		return cosmosKeys{}, fmt.Errorf("resource id '%s' type %s is not a cosmos resource", cosmosResourceID, res.ResourceType)
	}

	opType := cosmosRWOpType
	if readOnly {
		opType = cosmosReadOnlyOpType
	}

	cosmosURL := fmt.Sprintf(cosmosKeysRequestFmt, c.ARMEndpoint, cosmosResourceID, opType)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cosmosURL, nil)
	if err != nil {
		return cosmosKeys{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", armToken))

	retrier := retry.Retrier{Attempts: 10, Delay: time.Second}

	var keys cosmosKeys
	err = retrier.Do(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		bs, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(bs, &keys)
	})
	if err != nil {
		return cosmosKeys{}, err
	}
	return keys, nil
}
