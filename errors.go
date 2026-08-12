package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type appHandler func(
	http.ResponseWriter,
	*http.Request,
) error

type apiError struct {
	status  int
	message string
}

func (err *apiError) Error() string {
	return err.message
}

func newAPIError(status int, message string) *apiError {
	return &apiError{
		status:  status,
		message: message,
	}
}

func (app *application) handle(
	handler appHandler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			app.handleError(w, r, err)
		}
	}
}

func (app *application) handleError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	var apiErr *apiError

	switch {
	case errors.As(err, &apiErr):
		app.errorResponse(
			w,
			apiErr.status,
			apiErr.message,
		)

	case errors.Is(err, ErrRecapNotFound):
		app.errorResponse(
			w,
			http.StatusNotFound,
			"recap not found",
		)

	default:
		requestID := middleware.GetReqID(r.Context())

		log.Printf(
			"request_id=%s failed to handle request: %v",
			requestID,
			err,
		)

		app.errorResponse(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
	}
}
