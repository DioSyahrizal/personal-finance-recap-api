// Command migrate-supabase imports a JSON export of the old Supabase tables
// (recap_list and recap_item) into the new PostgreSQL schema.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type exportData struct {
	Recaps []sourceRecap `json:"recaps"`
	Items  []sourceItem  `json:"recap_items"`
}

type sourceRecap struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	BankName  string     `json:"bank_name"`
	Period    string     `json:"period"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type sourceItem struct {
	ID          int64     `json:"id"`
	RecapID     int64     `json:"recap_id"`
	Date        string    `json:"date"`
	Description string    `json:"description"`
	Amount      *float64  `json:"amount"`
	Balance     *float64  `json:"balance"`
	CreatedAt   time.Time `json:"created_at"`
	Category    *string   `json:"category"`
}

func main() {
	filePath := flag.String("file", "", "path to the Supabase JSON export")
	databaseURL := flag.String(
		"database-url",
		os.Getenv("DATABASE_URL"),
		"target PostgreSQL URL (defaults to DATABASE_URL)",
	)
	dryRun := flag.Bool("dry-run", false, "validate the export without writing to PostgreSQL")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("-file is required")
	}

	data, err := readExport(*filePath)
	if err != nil {
		log.Fatal(err)
	}

	if err := validateExport(data); err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"validated Supabase export: recaps=%d items=%d",
		len(data.Recaps),
		len(data.Items),
	)

	if *dryRun {
		log.Println("dry run: no database changes made")
		return
	}

	if *databaseURL == "" {
		log.Fatal("-database-url or DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("create database pool: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping target database: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatalf("begin migration transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	if err := importRows(ctx, tx, data); err != nil {
		log.Fatalf("import Supabase data: %v", err)
	}

	if err := resetSequences(ctx, tx); err != nil {
		log.Fatalf("reset identity sequences: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit migration: %v", err)
	}

	log.Println("Supabase data imported successfully")
}

func readExport(path string) (exportData, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return exportData{}, fmt.Errorf("open export: %w", err)
	}

	var data exportData
	if err := json.Unmarshal(contents, &data); err == nil {
		return data, nil
	}

	data, err = parseCopyDump(string(contents))
	if err != nil {
		return exportData{}, fmt.Errorf("decode export (JSON or pg_dump SQL): %w", err)
	}
	return data, nil
}

// parseCopyDump reads only the two application tables from a plain pg_dump
// file. Supabase's auth, storage, and other managed schemas are intentionally
// ignored.
func parseCopyDump(contents string) (exportData, error) {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	var data exportData
	section := ""
	seenRecaps := false
	seenItems := false

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, `COPY "public"."recap_list"`):
			section = "recaps"
			seenRecaps = true
			continue
		case strings.HasPrefix(line, `COPY "public"."recap_item"`):
			section = "items"
			seenItems = true
			continue
		case line == `\.`:
			section = ""
			continue
		}

		if section == "" || strings.TrimSpace(line) == "" {
			continue
		}

		fields, err := parseCopyFields(line)
		if err != nil {
			return exportData{}, fmt.Errorf("parse %s row: %w", section, err)
		}

		switch section {
		case "recaps":
			row, err := parseSourceRecap(fields)
			if err != nil {
				return exportData{}, err
			}
			data.Recaps = append(data.Recaps, row)
		case "items":
			row, err := parseSourceItem(fields)
			if err != nil {
				return exportData{}, err
			}
			data.Items = append(data.Items, row)
		}
	}

	if err := scanner.Err(); err != nil {
		return exportData{}, fmt.Errorf("read pg_dump SQL: %w", err)
	}
	if !seenRecaps || !seenItems {
		return exportData{}, fmt.Errorf(
			"pg_dump does not contain both public.recap_list and public.recap_item",
		)
	}

	return data, nil
}

func parseCopyFields(line string) ([]*string, error) {
	rawFields := strings.Split(line, "\t")
	fields := make([]*string, len(rawFields))

	for index, raw := range rawFields {
		if raw == `\N` {
			continue
		}
		value, err := decodeCopyValue(raw)
		if err != nil {
			return nil, fmt.Errorf("field %d: %w", index, err)
		}
		fields[index] = &value
	}

	return fields, nil
}

func decodeCopyValue(value string) (string, error) {
	var decoded strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' {
			decoded.WriteByte(value[index])
			continue
		}
		if index+1 >= len(value) {
			return "", fmt.Errorf("trailing escape")
		}

		index++
		switch value[index] {
		case 'b':
			decoded.WriteByte('\b')
		case 'f':
			decoded.WriteByte('\f')
		case 'n':
			decoded.WriteByte('\n')
		case 'r':
			decoded.WriteByte('\r')
		case 't':
			decoded.WriteByte('\t')
		case 'v':
			decoded.WriteByte('\v')
		case '\\':
			decoded.WriteByte('\\')
		default:
			decoded.WriteByte(value[index])
		}
	}
	return decoded.String(), nil
}

func parseSourceRecap(fields []*string) (sourceRecap, error) {
	if len(fields) != 8 {
		return sourceRecap{}, fmt.Errorf("recap row has %d fields, expected 8", len(fields))
	}
	id, err := requiredInt64(fields, 0, "recap id")
	if err != nil {
		return sourceRecap{}, err
	}
	name, err := requiredString(fields, 1, "recap name")
	if err != nil {
		return sourceRecap{}, err
	}
	bankName, err := optionalString(fields, 3)
	if err != nil {
		return sourceRecap{}, fmt.Errorf("recap %d bank_name: %w", id, err)
	}
	period, err := optionalString(fields, 4)
	if err != nil {
		return sourceRecap{}, fmt.Errorf("recap %d period: %w", id, err)
	}
	createdAt, err := requiredTime(fields, 5, "recap created_at")
	if err != nil {
		return sourceRecap{}, err
	}
	updatedAt, err := optionalTime(fields, 6)
	if err != nil {
		return sourceRecap{}, fmt.Errorf("recap %d updated_at: %w", id, err)
	}
	deletedAt, err := optionalTime(fields, 7)
	if err != nil {
		return sourceRecap{}, fmt.Errorf("recap %d deleted_at: %w", id, err)
	}

	status, err := optionalString(fields, 2)
	if err != nil {
		return sourceRecap{}, fmt.Errorf("recap %d status: %w", id, err)
	}

	return sourceRecap{
		ID:        id,
		Name:      name,
		Status:    status,
		BankName:  bankName,
		Period:    period,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}, nil
}

func parseSourceItem(fields []*string) (sourceItem, error) {
	if len(fields) != 8 {
		return sourceItem{}, fmt.Errorf("item row has %d fields, expected 8", len(fields))
	}
	id, err := requiredInt64(fields, 0, "item id")
	if err != nil {
		return sourceItem{}, err
	}
	recapID, err := requiredInt64(fields, 1, "item recap_id")
	if err != nil {
		return sourceItem{}, err
	}
	date, err := requiredString(fields, 2, "item date")
	if err != nil {
		return sourceItem{}, err
	}
	description, err := requiredString(fields, 3, "item description")
	if err != nil {
		return sourceItem{}, err
	}
	amount, err := optionalFloat(fields, 4)
	if err != nil {
		return sourceItem{}, fmt.Errorf("item %d amount: %w", id, err)
	}
	balance, err := optionalFloat(fields, 5)
	if err != nil {
		return sourceItem{}, fmt.Errorf("item %d balance: %w", id, err)
	}
	createdAt, err := requiredTime(fields, 6, "item created_at")
	if err != nil {
		return sourceItem{}, err
	}
	category, err := optionalStringPtr(fields, 7)
	if err != nil {
		return sourceItem{}, fmt.Errorf("item %d category: %w", id, err)
	}

	return sourceItem{
		ID:          id,
		RecapID:     recapID,
		Date:        date,
		Description: description,
		Amount:      amount,
		Balance:     balance,
		CreatedAt:   createdAt,
		Category:    category,
	}, nil
}

func requiredString(fields []*string, index int, label string) (string, error) {
	if index >= len(fields) || fields[index] == nil || strings.TrimSpace(*fields[index]) == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return *fields[index], nil
}

func optionalString(fields []*string, index int) (string, error) {
	if index >= len(fields) || fields[index] == nil {
		return "", nil
	}
	return *fields[index], nil
}

func optionalStringPtr(fields []*string, index int) (*string, error) {
	if index >= len(fields) || fields[index] == nil {
		return nil, nil
	}
	value := *fields[index]
	return &value, nil
}

func requiredInt64(fields []*string, index int, label string) (int64, error) {
	value, err := requiredString(fields, index, label)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s %q: %w", label, value, err)
	}
	return parsed, nil
}

func optionalFloat(fields []*string, index int) (*float64, error) {
	if index >= len(fields) || fields[index] == nil {
		return nil, nil
	}
	value, err := strconv.ParseFloat(*fields[index], 64)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func requiredTime(fields []*string, index int, label string) (time.Time, error) {
	if index >= len(fields) || fields[index] == nil {
		return time.Time{}, fmt.Errorf("%s is required", label)
	}
	value, err := parseDumpTime(*fields[index])
	if err != nil {
		return time.Time{}, fmt.Errorf("%s %q: %w", label, *fields[index], err)
	}
	return value, nil
}

func optionalTime(fields []*string, index int) (*time.Time, error) {
	if index >= len(fields) || fields[index] == nil {
		return nil, nil
	}
	value, err := parseDumpTime(*fields[index])
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseDumpTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07",
		"2006-01-02 15:04:05.999999999",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format")
}

func validateExport(data exportData) error {
	recapIDs := make(map[int64]struct{}, len(data.Recaps))
	for index, item := range data.Recaps {
		if item.ID <= 0 {
			return fmt.Errorf("recaps[%d]: id must be positive", index)
		}
		if _, exists := recapIDs[item.ID]; exists {
			return fmt.Errorf("recaps[%d]: duplicate id %d", index, item.ID)
		}
		recapIDs[item.ID] = struct{}{}
		if strings.TrimSpace(item.Name) == "" {
			return fmt.Errorf("recaps[%d]: name is required", index)
		}
		if strings.TrimSpace(item.BankName) == "" {
			return fmt.Errorf("recaps[%d]: bank_name is required", index)
		}
		if strings.TrimSpace(item.Period) == "" {
			return fmt.Errorf("recaps[%d]: period is required", index)
		}
		if item.Status != "" && !validRecapStatus(item.Status) {
			return fmt.Errorf("recaps[%d]: unsupported status %q", index, item.Status)
		}
		if !validPeriod(item.Period) {
			return fmt.Errorf("recaps[%d]: unsupported period %q", index, item.Period)
		}
		if item.CreatedAt.IsZero() {
			return fmt.Errorf("recaps[%d]: created_at is required", index)
		}
	}

	itemIDs := make(map[int64]struct{}, len(data.Items))
	for index, item := range data.Items {
		if item.ID <= 0 {
			return fmt.Errorf("recap_items[%d]: id must be positive", index)
		}
		if _, exists := itemIDs[item.ID]; exists {
			return fmt.Errorf("recap_items[%d]: duplicate id %d", index, item.ID)
		}
		itemIDs[item.ID] = struct{}{}
		if _, exists := recapIDs[item.RecapID]; !exists {
			return fmt.Errorf(
				"recap_items[%d]: recap_id %d is missing from recaps",
				index,
				item.RecapID,
			)
		}
		if item.Date == "" {
			return fmt.Errorf("recap_items[%d]: date is required", index)
		}
		if _, err := time.Parse("2006-01-02", item.Date); err != nil {
			return fmt.Errorf("recap_items[%d]: invalid date %q", index, item.Date)
		}
		if strings.TrimSpace(item.Description) == "" {
			return fmt.Errorf("recap_items[%d]: description is required", index)
		}
		if item.CreatedAt.IsZero() {
			return fmt.Errorf("recap_items[%d]: created_at is required", index)
		}
	}

	return nil
}

func validRecapStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "processing", "completed", "failed":
		return true
	default:
		return false
	}
}

func validPeriod(period string) bool {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december":
		return true
	default:
		return false
	}
}

func importRows(ctx context.Context, tx pgx.Tx, data exportData) error {
	for _, source := range data.Recaps {
		updatedAt := source.CreatedAt
		if source.UpdatedAt != nil {
			updatedAt = *source.UpdatedAt
		}

		status := strings.ToLower(strings.TrimSpace(source.Status))
		if status == "" {
			status = "pending"
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO recaps (
				id, name, status, bank_name, period, created_at, updated_at, deleted_at
			)
			OVERRIDING SYSTEM VALUE
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			source.ID,
			source.Name,
			status,
			source.BankName,
			strings.ToLower(strings.TrimSpace(source.Period)),
			source.CreatedAt,
			updatedAt,
			source.DeletedAt,
		)
		if err != nil {
			return fmt.Errorf("insert recap %d: %w", source.ID, err)
		}
	}

	for _, source := range data.Items {
		category := recap.CategoryUncategorized
		if source.Category != nil {
			category = recap.NormalizeCategory(*source.Category)
		}
		if isOpeningBalance(source.Description) {
			source.Amount = nil
			category = recap.CategoryUncategorized
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO recap_items (
				id,
				recap_id,
				transaction_date,
				description,
				amount,
				balance,
				category,
				created_at,
				updated_at
			)
			OVERRIDING SYSTEM VALUE
			VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $8)
		`,
			source.ID,
			source.RecapID,
			source.Date,
			source.Description,
			source.Amount,
			source.Balance,
			string(category),
			source.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert recap item %d: %w", source.ID, err)
		}
	}

	return nil
}

func isOpeningBalance(description string) bool {
	normalized := strings.ToLower(strings.TrimSpace(description))
	normalized = strings.ReplaceAll(normalized, `\`, "")
	normalized = strings.Trim(normalized, `"`)
	return normalized == "saldo awal"
}

func resetSequences(ctx context.Context, tx pgx.Tx) error {
	for _, table := range []string{"recaps", "recap_items"} {
		_, err := tx.Exec(ctx, fmt.Sprintf(`
			SELECT setval(
				pg_get_serial_sequence('%s', 'id'),
				COALESCE((SELECT MAX(id) FROM %s), 1),
				(SELECT COUNT(*) > 0 FROM %s)
			)
		`, table, table, table))
		if err != nil {
			return fmt.Errorf("reset %s sequence: %w", table, err)
		}
	}
	return nil
}
