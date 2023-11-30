package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

type Info struct {
	Name            string
	BuildId         string
	LastStartupTime time.Time
	Env             map[string]string
}

func InfoResource(r *chi.Mux, info Info) {
	r.Get("/info", func(w http.ResponseWriter, r *http.Request) {
		res, err := json.Marshal(info)
		if err != nil {
			http.Error(w, "error getting info", http.StatusInternalServerError)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res)
	})
}
