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
	recaps       []Recap
	err          error
	getByIDCalls int
	lastID       int64
}

func (store *stubRecapStore) List(
	ctx context.Context,
) ([]Recap, error) {
	return store.recaps, store.err
}

func (store *stubRecapStore) GetByID(
	ctx context.Context,
	id int64,
) (Recap, error) {

	store.getByIDCalls++
	store.lastID = id

	if store.err != nil {
		return Recap{}, store.err
	}

	if len(store.recaps) == 0 {
		return Recap{}, ErrRecapNotFound
	}

	return store.recaps[0], nil
}

func (store *stubRecapStore) ListItemsByRecapID(
	ctx context.Context,
	recapID int64,
) ([]RecapItem, error) {
	return nil, store.err
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

	assertJSONResponse(
		t,
		response,
		http.StatusInternalServerError,
	)

	defer response.Body.Close()

	var body map[string]string

	err := json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if body["error"] != "internal server error" {
		t.Errorf(
			"expected error %q, got %q",
			"internal server error",
			body["error"],
		)
	}
}

func TestGetRecapHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/recaps/1", nil)
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()

	store := &stubRecapStore{
		recaps: []Recap{
			{
				ID:       1,
				Name:     "January Recap",
				Period:   "2026-01-01",
				BankName: "BCA",
				Status:   "completed",
			},
		},
	}

	app := &application{
		store: store,
	}

	app.getRecapHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var recap Recap

	if err := json.NewDecoder(response.Body).Decode(&recap); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if recap.ID != 1 {
		t.Errorf("expected recap ID %d, got %d", 1, recap.ID)
	}

	if recap.Name != "January Recap" {
		t.Errorf(
			"expected recap name %q, got %q",
			"January Recap",
			recap.Name,
		)
	}

}

func TestGetRecapHandlerInvalidID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/not-a-number",
		nil,
	)
	request.SetPathValue("id", "not-a-number")

	recorder := httptest.NewRecorder()
	store := &stubRecapStore{}

	app := &application{
		store: store,
	}

	app.getRecapHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(
		t,
		response,
		http.StatusBadRequest,
	)

	var body map[string]string

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if body["error"] != "invalid recap id" {
		t.Errorf(
			"expected error %q, got %q",
			"invalid recap id",
			body["error"],
		)
	}

	if store.getByIDCalls != 0 {
		t.Errorf(
			"expected store not to be called, got %d calls",
			store.getByIDCalls,
		)
	}
}

func TestGetRecapHandlerNotFound(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/999",
		nil,
	)
	request.SetPathValue("id", "999")

	recorder := httptest.NewRecorder()

	// No recaps means the stub returns ErrRecapNotFound.
	store := &stubRecapStore{}

	app := &application{
		store: store,
	}

	app.getRecapHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(
		t,
		response,
		http.StatusNotFound,
	)

	var body map[string]string

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if body["error"] != "recap not found" {
		t.Errorf(
			"expected error %q, got %q",
			"recap not found",
			body["error"],
		)
	}

	if store.getByIDCalls != 1 {
		t.Errorf(
			"expected store to be called once, got %d calls",
			store.getByIDCalls,
		)
	}

	if store.lastID != 999 {
		t.Errorf(
			"expected store to receive ID %d, got %d",
			999,
			store.lastID,
		)
	}
}

func TestGetRecapHandlerStoreError(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/1",
		nil,
	)
	request.SetPathValue("id", "1")

	recorder := httptest.NewRecorder()

	store := &stubRecapStore{
		err: errors.New("database unavailable"),
	}

	app := &application{
		store: store,
	}

	app.getRecapHandler(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(
		t,
		response,
		http.StatusInternalServerError,
	)

	var body map[string]string

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if body["error"] != "internal server error" {
		t.Errorf(
			"expected error %q, got %q",
			"internal server error",
			body["error"],
		)
	}

	if store.getByIDCalls != 1 {
		t.Errorf(
			"expected store to be called once, got %d calls",
			store.getByIDCalls,
		)
	}

	if store.lastID != 1 {
		t.Errorf(
			"expected store to receive ID %d, got %d",
			1,
			store.lastID,
		)
	}
}
