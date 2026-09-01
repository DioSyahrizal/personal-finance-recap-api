# Finance Recap API — Project Plan

Updated: 2026-09-01

This project replaces Supabase finance-data operations with a Go/PostgreSQL
backend while keeping the frontend and authentication migration incremental.

## Completed

### Go API foundation

- [x] Create the initial Go HTTP server.
- [x] Add health endpoint.
- [x] Add JSON responses and centralized error handling.
- [x] Add Chi routing.
- [x] Add request ID, request logging, panic recovery, and request timeout middleware.
- [x] Split the backend into `internal/recap`, `internal/postgres`, and `internal/importer` packages.

### PostgreSQL and domain

- [x] Create `recaps` and `recap_items` tables.
- [x] Create import-job table and status lifecycle.
- [x] Add nullable transaction amounts for opening-balance rows.
- [x] Define recap, item, store, import-job, and import-creator interfaces.
- [x] Implement PostgreSQL recap, item, and import-job stores.
- [x] Use transactions for recap imports and bulk item inserts.
- [x] Add canonical transaction categories.
- [x] Normalize legacy categories and opening-balance data.

### PDF import pipeline

- [x] Accept multipart PDF uploads.
- [x] Validate upload size, required fields, and file extension.
- [x] Store uploaded PDFs locally.
- [x] Queue imports instead of processing inside the HTTP request.
- [x] Implement the background worker and job claiming.
- [x] Add processing timeout and failed/completed job updates.
- [x] Upload PDFs to OpenAI and extract structured transactions.
- [x] Require strict transaction JSON output.
- [x] Generate categories during PDF extraction.
- [x] Delete temporary local and OpenAI files after processing.

### Supabase data migration

- [x] Export the Supabase database data.
- [x] Inspect the old `recap_list` and `recap_item` schema.
- [x] Add a migration command for Supabase JSON exports.
- [x] Extend the migration command to read the actual plain `pg_dump` format.
- [x] Import only the application tables from the dump.
- [x] Ignore Supabase-managed `auth` and `storage` sections.
- [x] Preserve source IDs and timestamps.
- [x] Reset PostgreSQL identity sequences after import.
- [x] Validate the migration with a dry run.

### Frontend API cutover

- [x] Add a shared frontend finance API client.
- [x] Read dashboard recap data from the Go API.
- [x] Read history and recap details from the Go API.
- [x] Read analytics source data from the Go API.
- [x] Send uploads to the Go API.
- [x] Show `Pending`, `Processing`, `Completed`, and `Failed` statuses.
- [x] Remove upload polling from the frontend.
- [x] Keep Supabase authentication and settings temporarily.

## Next

### Analytics

- [ ] Design the `GET /api/v1/analytics` response.
- [ ] Aggregate monthly category totals in Go/PostgreSQL.
- [ ] Add the analytics route and handler.
- [ ] Update the frontend analytics page to use the new endpoint.

### Authentication and authorization

- [ ] Decide whether the Go API will validate Supabase JWTs or use a new auth system.
- [ ] Pass the frontend auth token to the Go API.
- [ ] Add JWT validation middleware.
- [ ] Associate recaps with users.
- [ ] Protect recap list, detail, upload, and analytics endpoints.

### Settings and model selection

- [ ] Decide whether model selection is server-controlled or user-configurable.
- [ ] Remove or connect the frontend model selector to the Go API.
- [ ] Migrate user settings away from Supabase when authentication is ready.

### Reliability and testing

- [ ] Add unit tests for category normalization.
- [ ] Add migration parser tests using a small fixture dump.
- [ ] Add worker retry and timeout tests.
- [ ] Add analytics aggregation tests.
- [ ] Add an end-to-end upload test against local PostgreSQL.
- [ ] Add graceful shutdown for the HTTP server and worker.
- [ ] Add structured production logging and basic metrics.

### Production VM migration

- [ ] Provision PostgreSQL on the production VM or choose a managed PostgreSQL provider.
- [ ] Provision the Go API and worker runtime.
- [ ] Apply all database migrations in order.
- [ ] Transfer the migrated recap data to production.
- [ ] Configure production environment variables securely.
- [ ] Configure upload storage and cleanup directories.
- [ ] Configure a process supervisor or container restart policy.
- [ ] Configure the production frontend API URL.
- [ ] Configure HTTPS, reverse proxying, and firewall rules.
- [ ] Run production smoke tests for listing, upload, worker processing, and analytics.
- [ ] Set up backups and a rollback procedure.

## Notes

- `data.sql` and `schema.sql` are local migration artifacts and must not be committed.
- The current Go API uses the server-side `OPENAI_MODEL` setting.
- New PDF imports are categorized during extraction; a separate legacy re-categorization job can be added later.
- Supabase remains temporarily responsible for authentication and settings only.
