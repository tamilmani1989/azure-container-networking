package server

import (
	"dnc/api"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"dnc/logger/loggershim"
)

var supportedAPIVersions = map[string]bool{"2018-03-01": true}

func Logger(inner http.Handler, name string, reverseProxyEnabled bool, connectionTimeout int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate if the API version is supported.
		_, err := getandValidateAPIVersion(r)
		if err != nil {
			setRequestFailed(w, err)
			return
		}

		if strings.Contains(r.RequestURI, "/azurestoragesettings") ||
			strings.Contains(r.RequestURI, "/azurestoragesettingsv2") ||
			strings.Contains(r.RequestURI, "/azurestoragesettingsv3") {
			if ctrler.IsReplicaReady() {
				return
			}
		} else {
			if !ctrler.IsReplicaReady() {
				setRequestFailed(w, &api.JsonErr{
					Code:       api.StorageNotInitialized,
					HTTPStatus: http.StatusBadRequest,
					Text:       "Azure Storage has not been initialized."})
				return
			}
		}

		// Check if the operation is a replica operation.
		if !isReplicaOperation(r) {
			// Check if the replica is primary. If it is not
			// redirect to the current primary.
			isReplicaPrimary := ctrler.Replica.IsReplicaPrimary()
			if !isReplicaPrimary {
				var err error
				var primaryEP string

				primaryEP, err = ctrler.GetPrimaryIPEndpoint()
				if err != nil {
					setRequestFailed(w, &api.JsonErr{
						Code:       api.PrimaryNotAvailable,
						Text:       api.PrimaryNotAvailable.String(),
						HTTPStatus: http.StatusInternalServerError})

					return
				}

				replicaEP, err := ctrler.GetReplicaIPEndpoint()
				if err != nil {
					setRequestFailed(w, &api.JsonErr{
						Code:       api.PrimaryNotAvailable,
						Text:       api.PrimaryNotAvailable.String(),
						HTTPStatus: http.StatusInternalServerError})

					return
				}

				if strings.Compare(primaryEP, replicaEP) == 0 {
					loggershim.LS.Errorf("Primary replica not ready. primaryEp: %s, replicaEp: %s",
						primaryEP, replicaEP)
					setRequestFailed(w, &api.JsonErr{
						Code:       api.PrimaryNotAvailable,
						Text:       api.PrimaryNotAvailable.String(),
						HTTPStatus: http.StatusInternalServerError})

					return
				}

				target := "http://" + primaryEP + r.URL.String()
				if reverseProxyEnabled {
					loggershim.LS.Debugf("Forwarding to %s", target)
					forwardToPrimary(target, primaryEP, w, r, connectionTimeout)
				} else {
					loggershim.LS.Debugf("Redirecting to %s", target)
					http.Redirect(w, r, target, http.StatusTemporaryRedirect)
				}
				return
			}
		}

		inner.ServeHTTP(w, r)
	})
}

func forwardToPrimary(target, primaryEP string, w http.ResponseWriter, r *http.Request, connectionTimeout int) {
	url, _ := url.ParseRequestURI(target)
	p := httputil.NewSingleHostReverseProxy(url)
	p.Director = func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
		req.Host = primaryEP
	}
	p.Transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: time.Duration(connectionTimeout) * time.Second,
		}).Dial,
	}
	p.ServeHTTP(w, r)
}

func isReplicaOperation(r *http.Request) bool {
	// ReliqnusihPrimaryStatus API is a replica call and this operation
	// should be performed on the replica on which it is invoked.
	return strings.Contains(r.RequestURI, "/relinquishprimarystatus")
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
	// Validate if the HTTPStatus is valid. w.WriteHeader needs it to be valid otherwise it panics.
	if len(http.StatusText(jsonErr.HTTPStatus)) == 0 {
		loggershim.LS.Errorf("Invalid HTTP status code set for setRequestFailed. httpStatus: %d",
			jsonErr.HTTPStatus)
		jsonErr.HTTPStatus = http.StatusInternalServerError
	}

	// log the response for the requests that fail with 500 error.
	// shouldn't be chatty as we don't expect to return this error code, often.
	if jsonErr.HTTPStatus == http.StatusInternalServerError {
		loggershim.LS.Errorf("500 (Internal server error) returned with response : %v", jsonErr)
	}

	w.WriteHeader(jsonErr.HTTPStatus)
	if err := json.NewEncoder(w).Encode(jsonErr); err != nil {
		panic(err)
	}
}
