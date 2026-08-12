package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(data)
}

func (app *application) errorResponse(
	w http.ResponseWriter,
	status int,
	message string,
) {
	if err := writeJSON(w, status, map[string]string{"error": message}); err != nil {
		log.Printf("failed to write error response: %v", err)
	}
}
