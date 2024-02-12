package service

import (
	"context"
	"net/http"
	_ "net/http/pprof" //nolint:gosec

	"github.com/gorilla/mux"
)

func AttachProfilingHandlers(_ context.Context, router *mux.Router) error {
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	return nil
}
