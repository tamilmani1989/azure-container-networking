package activedirectory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCosmosKeyFetching(t *testing.T) {
	expectedRW := CosmosRWKeys{PrimaryMasterKey: "primaryRW", SecondaryMasterKey: "secondaryRW"}
	expectedRO := CosmosReadOnlyKeys{PrimaryReadonlyMasterKey: "primaryRO", SecondaryReadonlyMasterKey: "secondaryRO"}
	handler := func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		switch {
		case strings.Contains(url, cosmosRWOpType):
			_ = json.NewEncoder(w).Encode(expectedRW)
		case strings.Contains(url, cosmosReadOnlyOpType):
			_ = json.NewEncoder(w).Encode(expectedRO)
		default:
			t.Fatalf("did not pass a valid operation type in url %s", url)
		}
	}
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	fetcher := CosmosKeyFetcher{ARMEndpoint: ts.URL}

	t.Run("fetches readwrite keys", func(t *testing.T) {
		resID := "/subscriptions/some-sub-id/resourceGroups/some-rg/providers/Microsoft.DocumentDb/databaseAccounts/cosmos-test"
		rwKeys, err := fetcher.GetReadWriteKeys(context.Background(), "someToken", resID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if rwKeys != expectedRW {
			t.Fatalf("expected %+v, got %+v", expectedRW, rwKeys)
		}
	})

	t.Run("fetches readonly keys", func(t *testing.T) {
		resID := "/subscriptions/some-sub-id/resourceGroups/some-rg/providers/Microsoft.DocumentDb/databaseAccounts/cosmos-test"
		roKeys, err := fetcher.GetReadOnlyKeys(context.Background(), "someToken", resID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if roKeys != expectedRO {
			t.Fatalf("expected %+v, got %+v", expectedRO, roKeys)
		}
	})

	t.Run("wrong resource type", func(t *testing.T) {
		resID := "/subscriptions/some-sub-id/resourceGroups/some-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/some-identity"
		_, err := fetcher.GetReadWriteKeys(context.Background(), "someToken", resID)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
