package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/diosyahrizal/finance-recap-api/internal/importer"
	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type application struct {
	store         recap.Store
	importCreator recap.ImportCreator
	fileStore     importer.FileStore
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
	const maxUploadBytes int64 = 10 << 20 // 10 MB

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxUploadBytes,
	)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
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

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return newAPIError(
			http.StatusBadRequest,
			"PDF file is required",
		)
	}
	defer file.Close()

	if !strings.EqualFold(
		filepath.Ext(fileHeader.Filename),
		".pdf",
	) {
		return newAPIError(
			http.StatusBadRequest,
			"file must be a PDF",
		)
	}

	filePath, err := app.fileStore.Save(
		r.Context(),
		file,
	)
	if err != nil {
		return err
	}

	input := recap.CreateInput{
		Name:     name,
		BankName: bankName,
		Period:   period,
	}

	createdRecap, err := app.importCreator.CreateImport(
		r.Context(),
		input,
		filePath,
	)
	if err != nil {
		cleanupContext := context.WithoutCancel(r.Context())

		if deleteErr := app.fileStore.Delete(
			cleanupContext,
			filePath,
		); deleteErr != nil {
			log.Printf(
				"failed to clean up uploaded file %q: %v",
				filePath,
				deleteErr,
			)
		}

		return err
	}

	return writeJSON(
		w,
		http.StatusAccepted,
		createdRecap,
	)
}
