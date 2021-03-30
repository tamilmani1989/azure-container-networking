package server

import (
	"dnc/logger/loggermock"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		dbQueryResult    Querier
		expectedHTTPCode int
	}{
		{
			name:             "successful db call, row not found",
			dbQueryResult:    &fakeQueryResult{err: storageErr(http.StatusNotFound)},
			expectedHTTPCode: http.StatusOK,
		},
		{
			name:             "successful db call, row found",
			dbQueryResult:    &fakeQueryResult{entity: new(storage.Entity), err: nil},
			expectedHTTPCode: http.StatusOK,
		},
		{
			name:             "transient db errors, db internal error",
			dbQueryResult:    &fakeQueryResult{transientErr: storageErr(http.StatusInternalServerError), err: storageErr(http.StatusNotFound), transientErrUntil: 3},
			expectedHTTPCode: http.StatusOK,
		},
		{
			name:             "transient db errors, other",
			dbQueryResult:    &fakeQueryResult{transientErr: errors.New("some conn err"), err: storageErr(http.StatusNotFound), transientErrUntil: 3},
			expectedHTTPCode: http.StatusOK,
		},
		{
			name:             "db errors over threshold",
			dbQueryResult:    &fakeQueryResult{transientErr: errors.New("some conn err"), err: storageErr(http.StatusNotFound), transientErrUntil: 10},
			expectedHTTPCode: http.StatusInternalServerError,
		},
		{
			name:             "unauthorized",
			dbQueryResult:    &fakeQueryResult{err: storageErr(http.StatusUnauthorized)},
			expectedHTTPCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		ts := tt
		t.Run(ts.name, func(t *testing.T) {
			t.Parallel()
			r := ProbeRoutes{Querier: ts.dbQueryResult, Logger: loggermock.NewDummyLogger()}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

			r.Health(w, req)

			resp := w.Result()
			require.NotNil(t, resp)
			assert.Equal(t, ts.expectedHTTPCode, resp.StatusCode)
		})
	}
}

func storageErr(code int) storage.AzureStorageServiceError {
	return storage.AzureStorageServiceError{StatusCode: code}
}

type fakeQueryResult struct {
	entity            *storage.Entity
	err               error
	callCount         int
	transientErr      error
	transientErrUntil int
}

func (f *fakeQueryResult) Query(_ string, _ string) (*storage.Entity, int, error) {
	if f.callCount < f.transientErrUntil {
		f.callCount++
		return nil, 0, f.transientErr
	}

	return f.entity, 0, f.err
}
