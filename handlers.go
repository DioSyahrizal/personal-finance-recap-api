package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
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
		app.serverErrorResponse(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, recaps); err != nil {
		log.Printf("failed to write recaps response: %v", err)
	}
}

func (app *application) getRecapHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)

	if err != nil || id <= 0 {
		app.errorResponse(
			w,
			http.StatusBadRequest,
			"invalid recap id",
		)
		return
	}

	recap, err := app.store.GetByID(r.Context(), id)

	if errors.Is(err, ErrRecapNotFound) {
		app.errorResponse(
			w,
			http.StatusNotFound,
			"recap not found",
		)
		return
	}

	if err != nil {
		app.serverErrorResponse(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, recap); err != nil {
		log.Printf("failed to write recap response: %v", err)
	}
}

func (app *application) listRecapItemsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)

	if err != nil || id <= 0 {
		app.errorResponse(
			w,
			http.StatusBadRequest,
			"invalid recap id",
		)
		return
	}

	_, err = app.store.GetByID(r.Context(), id)

	if errors.Is(err, ErrRecapNotFound) {
		app.errorResponse(w, http.StatusNotFound, "recap not found")
		return
	}

	if err != nil {
		app.serverErrorResponse(w, err)
		return
	}

	recapItems, err := app.store.ListItemsByRecapID(r.Context(), id)

	if err != nil {
		app.serverErrorResponse(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, recapItems); err != nil {
		log.Printf("failed to write recap items response: %v", err)
	}
}
