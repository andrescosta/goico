package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func WriteJSONBody(b *bytes.Buffer, body any, status int, altError string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(b).Encode(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, altError, http.StatusText(http.StatusInternalServerError))
		return
	}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/JSON")
	_, _ = b.WriteTo(w)
}
