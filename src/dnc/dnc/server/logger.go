package server

import (
	"dnc/api"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dnc/logger/loggershim"
)

var supportedAPIVersions = map[string]bool{"2018-03-01": true}

func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Validate if the API version is supported.
		_, err := getandValidateAPIVersion(r)
		if err != nil {
			setRequestFailed(w, err)
			return
		}

		// Check if the replica is primary. If it is not
		// redirect to the current primary.
		isReplicaPrimary := ctrler.Replica.IsReplicaPrimary()
		if !isReplicaPrimary {
			var err error
			var primaryEP string

			primaryEP, err = ctrler.GetPrimaryIPEndpoint()
			if err != nil {
				jsonErr := api.JsonErr{
					Code: api.PrimaryNotAvailable}
				w.WriteHeader(http.StatusInternalServerError)
				if err := json.NewEncoder(w).Encode(jsonErr); err != nil {
					panic(err)
				}

				return
			}

			replicaEP, err := ctrler.GetReplicaIPEndpoint()
			if err != nil {
				jsonErr := api.JsonErr{
					Code: api.PrimaryNotAvailable}
				w.WriteHeader(http.StatusInternalServerError)
				if err := json.NewEncoder(w).Encode(jsonErr); err != nil {
					panic(err)
				}

				return
			}

			if strings.Compare(primaryEP, replicaEP) == 0 {
				err = fmt.Errorf("Primary replica not ready")
				jsonErr := api.JsonErr{
					Code: api.PrimaryNotAvailable}
				w.WriteHeader(http.StatusInternalServerError)
				if err := json.NewEncoder(w).Encode(jsonErr); err != nil {
					panic(err)
				}

				return
			}

			target := "http://" + primaryEP + r.URL.String()
			loggershim.LS.Log("Redirecting to %s\n", target)
			http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			return
		}

		inner.ServeHTTP(w, r)

		loggershim.LS.Log(
			"%s\t%s\t%s\t%s\n",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start).String(),
		)
	})
}

func getandValidateAPIVersion(request *http.Request) (string, *api.JsonErr) {
	apiVersion := request.URL.Query().Get("api-version")
	if len(apiVersion) == 0 {
		return "", &api.JsonErr{
			Code:       api.APIVersionMissing,
			HTTPStatus: http.StatusBadRequest,
			Text:       "Api version is missing in the request URI"}
	}
	ok, _ := supportedAPIVersions[apiVersion]
	if !ok {
		return "", &api.JsonErr{
			Code:       api.APIVersionNotSupported,
			HTTPStatus: http.StatusBadRequest,
			Text:       "Api version is not supported"}
	}

	return "", nil
}

func setRequestFailed(w http.ResponseWriter, jsonErr *api.JsonErr) {
	w.WriteHeader(jsonErr.HTTPStatus)
	if err := json.NewEncoder(w).Encode(jsonErr); err != nil {
		panic(err)
	}
}
