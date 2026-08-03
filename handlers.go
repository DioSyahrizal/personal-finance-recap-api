package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type application struct {
	store recapStore
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	err := writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	if err != nil {
		log.Printf("Failed to write Health response: %v", err)
	}
}

func (app *application) listRecapsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	recaps, err := app.store.List(r.Context())
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recaps)
}
