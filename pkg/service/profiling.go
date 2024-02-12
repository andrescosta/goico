package service

import (
	"net/http"
	_ "net/http/pprof" //nolint:gosec //controlled by config.

	"github.com/gorilla/mux"
)

func AttachProfilingHandlers(router *mux.Router) {
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
}
