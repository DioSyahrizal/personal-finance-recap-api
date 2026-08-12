package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func newMockStore(t *testing.T) (*RecapStore, pgxmock.PgxPoolIface) {
	t.Helper()

	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create database mock: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return NewRecapStore(db), db
}

func TestRecapStoreList(t *testing.T) {
	store, db := newMockStore(t)
	createdAt := time.Date(2025, time.December, 30, 3, 25, 45, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)

	rows := pgxmock.NewRows([]string{
		"id",
		"name",
		"status",
		"bank_name",
		"period",
		"created_at",
		"updated_at",
		"deleted_at",
	}).AddRow(
		int64(1),
		"Monthly Expenses September 2025",
		"completed",
		"Bank Central Asia",
		"september",
		createdAt,
		updatedAt,
		nil,
	)

	db.ExpectQuery(`SELECT .* FROM recaps`).WillReturnRows(rows)

	results, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 recap, got %d", len(results))
	}

	if results[0].ID != 1 {
		t.Errorf("expected recap ID %d, got %d", 1, results[0].ID)
	}

	if results[0].Period != "september" {
		t.Errorf("expected period %q, got %q", "september", results[0].Period)
	}

	if results[0].DeletedAt != nil {
		t.Errorf("expected deleted_at to be nil, got %v", results[0].DeletedAt)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestRecapStoreGetByID(t *testing.T) {
	store, db := newMockStore(t)
	createdAt := time.Date(2025, time.December, 30, 3, 25, 45, 0, time.UTC)

	rows := pgxmock.NewRows([]string{
		"id",
		"name",
		"status",
		"bank_name",
		"period",
		"created_at",
		"updated_at",
		"deleted_at",
	}).AddRow(
		int64(1),
		"Monthly Expenses September 2025",
		"completed",
		"Bank Central Asia",
		"september",
		createdAt,
		createdAt,
		nil,
	)

	db.ExpectQuery(`SELECT .* FROM recaps`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	result, err := store.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.ID != 1 {
		t.Errorf("expected recap ID %d, got %d", 1, result.ID)
	}

	if result.Name != "Monthly Expenses September 2025" {
		t.Errorf(
			"expected recap name %q, got %q",
			"Monthly Expenses September 2025",
			result.Name,
		)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestRecapStoreGetByIDNotFound(t *testing.T) {
	store, db := newMockStore(t)

	db.ExpectQuery(`SELECT .* FROM recaps`).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err := store.GetByID(context.Background(), 999)
	if !errors.Is(err, recap.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestRecapStoreListItemsByRecapID(t *testing.T) {
	store, db := newMockStore(t)
	createdAt := time.Date(2025, time.December, 30, 3, 48, 27, 0, time.UTC)
	balance := 175654122.59
	category := "Uncategorized"

	rows := pgxmock.NewRows([]string{
		"id",
		"recap_id",
		"transaction_date",
		"description",
		"amount",
		"balance",
		"created_at",
		"category",
	}).AddRow(
		int64(1),
		int64(1),
		"2025-09-01",
		`"SALDO AWAL"`,
		float64(0),
		&balance,
		createdAt,
		&category,
	)

	db.ExpectQuery(`SELECT .* FROM recap_items`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	items, err := store.ListItemsByRecapID(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 recap item, got %d", len(items))
	}

	if items[0].Balance == nil {
		t.Fatal("expected balance, got nil")
	}

	if *items[0].Balance != 175654122.59 {
		t.Errorf("expected balance %.2f, got %.2f", 175654122.59, *items[0].Balance)
	}

	if items[0].Category == nil || *items[0].Category != "Uncategorized" {
		t.Errorf("expected category %q, got %v", "Uncategorized", items[0].Category)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}
