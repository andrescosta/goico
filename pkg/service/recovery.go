package service

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

func TryToRecover() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := zerolog.Ctx(r.Context())
			defer func() {
				if p := recover(); p != nil {
					logger.Error().Msgf("cannot recover http handler due to %v", p)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
