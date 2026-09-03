package main

import (
	"net/http"
	"strings"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

func (app *application) getAnalyticsHandler(
	w http.ResponseWriter,
	r *http.Request,
) error {

	query := r.URL.Query()

	filter := recap.AnalyticsFilter{
		From: strings.TrimSpace(query.Get("from")),
		To:   strings.TrimSpace(query.Get("to")),
		Bank: strings.TrimSpace(query.Get("bank")),
	}

	result, err := app.store.GetAnalytics(r.Context(), filter)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, result)

}
