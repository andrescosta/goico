package obs

import (
	"net/http"

	"github.com/rs/zerolog"
)

func GetLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		logger.Debug().Msg(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
