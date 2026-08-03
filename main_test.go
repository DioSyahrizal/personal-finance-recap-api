package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubRecapStore struct {
	recaps []Recap
	err    error
}

func (store *stubRecapStore) List(
	ctx context.Context,
) ([]Recap, error) {
	return store.recaps, store.err
}

func assertJSONResponse(
	t *testing.T,
	response *http.Response,
	expectedStatus int,
) {
	t.Helper()

	if response.StatusCode != expectedStatus {
		t.Errorf(
			"expected status %d, got %d",
			expectedStatus,
			response.StatusCode,
		)
	}

	contentType := response.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf(
			"expected Content-Type %q, got %q",
			"application/json",
			contentType,
		)
	}
}

func TestHealthHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	healthHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var body map[string]string

	err := json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status %q, got %q", "ok", body["status"])
	}
}

func TestListRecapsHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/recaps", nil)
	recorder := httptest.NewRecorder()

	store := &stubRecapStore{
		recaps: []Recap{
			{
				ID:   1,
				Name: "January Recap",
			},
		},
	}

	app := &application{
		store: store,
	}

	app.listRecapsHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var recaps []Recap

	err := json.NewDecoder(response.Body).Decode(&recaps)

	if err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if len(recaps) != 1 {
		t.Fatalf("expected 1 recap, got %d instead", len(recaps))
	}

	if recaps[0].Name != "January Recap" {
		t.Errorf("expected name of array 0 is %q, got %q instead", "January Recap", recaps[0].Name)
	}
}

func TestListRecapsHandlerStoreError(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps",
		nil,
	)
	recorder := httptest.NewRecorder()

	store := &stubRecapStore{
		err: errors.New("database unavailable"),
	}

	app := &application{
		store: store,
	}

	app.listRecapsHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusInternalServerError {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			response.StatusCode,
		)
	}
}
