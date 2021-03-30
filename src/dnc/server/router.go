package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(reverseProxyEnabled, useHttps bool, probeRoutes *ProbeRoutes) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// todo: routes should not be global and depend on other globals. they should be instantiated and have dependencies injected.
	for _, route := range routes {
		handler := Logger(route.HandlerFunc, reverseProxyEnabled, useHttps)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	// replica specific methods
	router.Methods(http.MethodGet).Path("/healthz").Handler(http.HandlerFunc(probeRoutes.Health))

	return router
}
