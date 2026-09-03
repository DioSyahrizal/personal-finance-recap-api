package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))

	router.Get("/health", app.handle(healthHandler))

	router.Route("/api/v1", func(router chi.Router) {
		router.Get("/recaps", app.handle(app.listRecapsHandler))
		router.Post("/recaps", app.handle(app.createRecapHandler))

		router.Get("/recaps/{id}", app.handle(app.getRecapHandler))
		router.Get(
			"/recaps/{id}/items",
			app.handle(app.listRecapItemsHandler),
		)

		router.Get("/analytics", app.handle(app.getAnalyticsHandler))
	})

	return router
}
