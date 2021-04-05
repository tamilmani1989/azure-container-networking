package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
)

func NewRouter(reverseProxyEnabled, useHttps bool, probeRoutes *ProbeRoutes, debugRoutes *DebugRoutes) *mux.Router {
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

	// debug methods
	router.Methods(http.MethodPost).Path("/debug/rehydrateallocations").Handler(http.HandlerFunc(debugRoutes.RehydrateSubnetAllocations))

	router.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	router.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	router.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	router.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
	router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

	return router
}
