package main

import (
	"net/http"
	"strconv"
	"strings"

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

func (app *application) createRecapHandler(
	w http.ResponseWriter,
	r *http.Request,
) error {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		return newAPIError(
			http.StatusBadRequest,
			"invalid multipart form",
		)
	}

	defer r.MultipartForm.RemoveAll()

	name := strings.TrimSpace(r.FormValue("name"))
	bankName := strings.TrimSpace(r.FormValue("bank"))
	period := strings.ToLower(strings.TrimSpace(r.FormValue("period")))

	if name == "" {
		return newAPIError(
			http.StatusBadRequest,
			"name is required",
		)
	}

	if bankName == "" {
		return newAPIError(
			http.StatusBadRequest,
			"bank is required",
		)
	}

	if period == "" {
		return newAPIError(
			http.StatusBadRequest,
			"period is required",
		)
	}

	input := recap.CreateInput{
		Name:     name,
		BankName: bankName,
		Period:   period,
	}

	createdRecap, err := app.store.Create(r.Context(), input)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusCreated, createdRecap)
}
