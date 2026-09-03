package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type stubRecapStore struct {
	recaps              []recap.Recap
	err                 error
	getByIDCalls        int
	lastID              int64
	items               []recap.Item
	itemsErr            error
	listItemsCalls      int
	lastItemsByRecapID  int64
	panicOnList         bool
	createdRecap        recap.Recap
	createErr           error
	createCalls         int
	lastCreateInput     recap.CreateInput
	analytics           recap.Analytics
	analyticsErr        error
	getAnalyticsCalls   int
	lastAnalyticsFilter recap.AnalyticsFilter
}

func (store *stubRecapStore) GetAnalytics(
	ctx context.Context,
	filter recap.AnalyticsFilter,
) (recap.Analytics, error) {
	store.getAnalyticsCalls++
	store.lastAnalyticsFilter = filter

	return store.analytics, store.analyticsErr
}

func (store *stubRecapStore) List(
	ctx context.Context,
) ([]recap.Recap, error) {
	if store.panicOnList {
		panic("unexpected failure")
	}

	return store.recaps, store.err
}

func (store *stubRecapStore) GetByID(
	ctx context.Context,
	id int64,
) (recap.Recap, error) {

	store.getByIDCalls++
	store.lastID = id

	if store.err != nil {
		return recap.Recap{}, store.err
	}

	if len(store.recaps) == 0 {
		return recap.Recap{}, recap.ErrNotFound
	}

	return store.recaps[0], nil
}

func (store *stubRecapStore) ListItemsByRecapID(
	ctx context.Context,
	recapID int64,
) ([]recap.Item, error) {
	store.listItemsCalls++
	store.lastItemsByRecapID = recapID

	return store.items, store.itemsErr
}

func (store *stubRecapStore) Create(
	ctx context.Context,
	input recap.CreateInput,
) (recap.Recap, error) {
	store.createCalls++
	store.lastCreateInput = input

	return store.createdRecap, store.createErr
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

	app := &application{}
	app.handle(healthHandler)(recorder, request)

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
		recaps: []recap.Recap{
			{
				ID:   1,
				Name: "January Recap",
			},
		},
	}

	app := &application{
		store: store,
	}

	app.handle(app.listRecapsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var recaps []recap.Recap

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

	app.handle(app.listRecapsHandler)(recorder, request)

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
		recaps: []recap.Recap{
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

	app.handle(app.getRecapHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var recap recap.Recap

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

	app.handle(app.getRecapHandler)(recorder, request)

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

	app.handle(app.getRecapHandler)(recorder, request)

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

	app.handle(app.getRecapHandler)(recorder, request)

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

func TestListRecapItemsHandler(t *testing.T) {
	amount := 1000.0

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/1/items",
		nil,
	)
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()

	store := &stubRecapStore{
		recaps: []recap.Recap{
			{
				ID:   1,
				Name: "January Recap",
			},
		},
		items: []recap.Item{
			{
				ID:          1,
				RecapID:     1,
				Date:        "2022-01-01",
				Description: "Payment to John Doe",
				Amount:      &amount,
				Balance:     nil,
				Category:    nil,
			},
		},
	}

	app := &application{
		store: store,
	}

	app.handle(app.listRecapItemsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var items []recap.Item

	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 recap item, got %d", len(items))
	}

	if items[0].Description != "Payment to John Doe" {
		t.Errorf(
			"expected description %q, got %q",
			"Payment to John Doe",
			items[0].Description,
		)
	}

	if store.getByIDCalls != 1 {
		t.Errorf("expected GetByID to be called once, got %d calls", store.getByIDCalls)
	}

	if store.listItemsCalls != 1 {
		t.Errorf(
			"expected ListItemsByRecapID to be called once, got %d calls",
			store.listItemsCalls,
		)
	}

	if store.lastItemsByRecapID != 1 {
		t.Errorf(
			"expected ListItemsByRecapID to receive ID %d, got %d",
			1,
			store.lastItemsByRecapID,
		)
	}
}

func TestListRecapItemsHandlerEmpty(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/1/items",
		nil,
	)
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()

	store := &stubRecapStore{
		recaps: []recap.Recap{{ID: 1}},
		items:  []recap.Item{},
	}

	app := &application{store: store}
	app.handle(app.listRecapItemsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	var items []recap.Item
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if items == nil {
		t.Fatal("expected an empty JSON array, got null")
	}

	if len(items) != 0 {
		t.Errorf("expected 0 recap items, got %d", len(items))
	}
}

func TestListRecapItemsHandlerInvalidID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/not-a-number/items",
		nil,
	)
	request.SetPathValue("id", "not-a-number")
	recorder := httptest.NewRecorder()
	store := &stubRecapStore{}

	app := &application{store: store}
	app.handle(app.listRecapItemsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusBadRequest)

	if store.getByIDCalls != 0 {
		t.Errorf("expected GetByID not to be called, got %d calls", store.getByIDCalls)
	}

	if store.listItemsCalls != 0 {
		t.Errorf(
			"expected ListItemsByRecapID not to be called, got %d calls",
			store.listItemsCalls,
		)
	}
}

func TestListRecapItemsHandlerRecapNotFound(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/999/items",
		nil,
	)
	request.SetPathValue("id", "999")
	recorder := httptest.NewRecorder()
	store := &stubRecapStore{}

	app := &application{store: store}
	app.handle(app.listRecapItemsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusNotFound)

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON response, got %v", err)
	}

	if body["error"] != "recap not found" {
		t.Errorf("expected error %q, got %q", "recap not found", body["error"])
	}

	if store.getByIDCalls != 1 {
		t.Errorf("expected GetByID to be called once, got %d calls", store.getByIDCalls)
	}

	if store.listItemsCalls != 0 {
		t.Errorf(
			"expected ListItemsByRecapID not to be called, got %d calls",
			store.listItemsCalls,
		)
	}
}

func TestListRecapItemsHandlerRecapStoreError(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/1/items",
		nil,
	)
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()
	store := &stubRecapStore{
		err: errors.New("database unavailable"),
	}

	app := &application{store: store}
	app.handle(app.listRecapItemsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusInternalServerError)

	if store.getByIDCalls != 1 {
		t.Errorf("expected GetByID to be called once, got %d calls", store.getByIDCalls)
	}

	if store.listItemsCalls != 0 {
		t.Errorf(
			"expected ListItemsByRecapID not to be called, got %d calls",
			store.listItemsCalls,
		)
	}
}

func TestListRecapItemsHandlerItemsStoreError(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/1/items",
		nil,
	)
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()
	store := &stubRecapStore{
		recaps:   []recap.Recap{{ID: 1}},
		itemsErr: errors.New("database unavailable"),
	}

	app := &application{store: store}
	app.handle(app.listRecapItemsHandler)(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusInternalServerError)

	if store.getByIDCalls != 1 {
		t.Errorf("expected GetByID to be called once, got %d calls", store.getByIDCalls)
	}

	if store.listItemsCalls != 1 {
		t.Errorf(
			"expected ListItemsByRecapID to be called once, got %d calls",
			store.listItemsCalls,
		)
	}
}

func TestRoutes(t *testing.T) {
	store := &stubRecapStore{
		recaps: []recap.Recap{
			{
				ID:   1,
				Name: "January Recap",
			},
		},
	}

	app := &application{store: store}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps/1",
		nil,
	)
	recorder := httptest.NewRecorder()

	app.routes().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	assertJSONResponse(t, response, http.StatusOK)

	if store.getByIDCalls != 1 {
		t.Errorf(
			"expected GetByID to be called once, got %d calls",
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

func TestRecoverer(t *testing.T) {
	store := &stubRecapStore{
		panicOnList: true,
	}

	app := &application{store: store}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/recaps",
		nil,
	)
	recorder := httptest.NewRecorder()

	app.routes().ServeHTTP(recorder, request)

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
