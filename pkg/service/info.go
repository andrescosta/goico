package service

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
)

type Info struct {
	Name    string
	BuildId string
	Env     map[string]string
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
