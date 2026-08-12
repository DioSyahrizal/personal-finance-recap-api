package main

import (
	"net/http"
	"strconv"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type application struct {
	store recap.Store
}

func healthHandler(w http.ResponseWriter, r *http.Request) error {
	return writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (app *application) listRecapsHandler(
	w http.ResponseWriter,
	r *http.Request,
) error {
	recaps, err := app.store.List(r.Context())
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, recaps)
}

func (app *application) getRecapHandler(
	w http.ResponseWriter,
	r *http.Request,
) error {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)

	if err != nil || id <= 0 {
		return newAPIError(
			http.StatusBadRequest,
			"invalid recap id",
		)
	}

	recap, err := app.store.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, recap)
}

func (app *application) listRecapItemsHandler(
	w http.ResponseWriter,
	r *http.Request,
) error {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)

	if err != nil || id <= 0 {
		return newAPIError(
			http.StatusBadRequest,
			"invalid recap id",
		)
	}

	_, err = app.store.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	recapItems, err := app.store.ListItemsByRecapID(r.Context(), id)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, recapItems)
}
