package server

import (
	"dnc/logger/loggershim"
	"dnc/retry"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// ProbeRoutes provides http handler funcs for probing DNC status
type ProbeRoutes struct {
	Querier Querier
	Logger  loggershim.LoggerInterface
}

type Querier interface {
	Query(tableName string, rowKey string) (*storage.Entity, int, error)
}

type DNCStatusJSON struct {
	StorageHealthy      bool
	StorageErrorMessage string
}

// todo: configurable?
var retrier = retry.Retrier{
	Attempts:   5,
	Delay:      time.Second,
	ExpBackoff: false,
}

func (h *ProbeRoutes) Health(w http.ResponseWriter, r *http.Request) {
	if h.Querier == nil {
		h.Logger.Errorf("[health probe] data store not initialized")
		h.writeResponse(w, http.StatusInternalServerError, DNCStatusJSON{StorageHealthy: false, StorageErrorMessage: "data store not initialized"})
		return
	}

	var (
		entity     *storage.Entity
		storageErr storage.AzureStorageServiceError
	)

	retryFn := func() error {
		e, _, err := h.Querier.Query("NetworkConfigurationTable", "dummy")
		if err != nil {
			stErr, ok := err.(storage.AzureStorageServiceError)
			if !ok || stErr.StatusCode == http.StatusInternalServerError {
				return err // retry transient or unknown errors
			}

			storageErr = stErr // normal expectation is a not found error
			return nil
		}

		entity = e // dummy row not expected to be found, but sign of healthy connection
		return nil
	}

	// this only errors after repeated transient errors or a cancelled request
	if err := retrier.Do(r.Context(), retryFn); err != nil {
		h.Logger.Errorf("[health probe] unexpected error: %v", err)
		h.writeResponse(w, http.StatusInternalServerError, DNCStatusJSON{StorageHealthy: false, StorageErrorMessage: err.Error()})
		return
	}

	if entity != nil {
		h.Logger.Debugf("[health probe] unexpected row found: %+v", *entity)
		h.writeResponse(w, http.StatusOK, DNCStatusJSON{StorageHealthy: true})
		return
	}

	if storageErr.StatusCode != http.StatusNotFound {
		h.Logger.Errorf("[health probe] error from storage: %s", storageErr.Error())
		status := DNCStatusJSON{StorageHealthy: false, StorageErrorMessage: storageErr.Error()}
		h.writeResponse(w, http.StatusInternalServerError, status)
		return
	}

	h.Logger.Debugf("[health probe] storage connection healthy")
	h.writeResponse(w, http.StatusOK, DNCStatusJSON{StorageHealthy: true})
	return
}

func (h *ProbeRoutes) writeResponse(w http.ResponseWriter, status int, rep DNCStatusJSON) {
	w.Header().Add(contentTypeStr, defaultContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(rep); err != nil {
		h.Logger.Errorf("[health probe] could not encode status response: %v", err)
	}
}
