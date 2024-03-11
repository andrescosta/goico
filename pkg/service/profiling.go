//nolint:all
package service

import (
	_ "expvar"
	"net/http"
	_ "net/http/pprof"

	"github.com/gorilla/mux"
)

func AttachProfilingHandlers(router *mux.Router) {
	router.PathPrefix("/debug/").Handler(http.DefaultServeMux)
}
