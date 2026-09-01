# Supabase data migration

This command imports the old Supabase `recap_list` and `recap_item` rows into
the Go API's `recaps` and `recap_items` tables. It accepts either a JSON export
or the plain SQL file produced by `supabase db dump`.

For a JSON export, use this shape:

```json
{
  "recaps": [
    {
      "id": 1,
      "name": "Monthly Expenses September 2025",
      "status": "completed",
      "bank_name": "Bank Central Asia",
      "period": "september",
      "created_at": "2025-12-30T03:25:45.862124+00:00",
      "updated_at": "2025-12-30T03:25:45.862124+00:00",
      "deleted_at": null
    }
  ],
  "recap_items": [
    {
      "id": 1,
      "recap_id": 1,
      "date": "2025-09-01",
      "description": "\"SALDO AWAL\"",
      "amount": null,
      "balance": 175654122.59,
      "category": "Uncategorized",
      "created_at": "2025-12-30T03:48:27.984584+00:00"
    }
  ]
}
```

Run [export.sql](export.sql) in the Supabase SQL editor, copy its single JSON result into
`supabase-export.json`, and save it locally:

```sql
select jsonb_build_object(
  'recaps', coalesce((select jsonb_agg(to_jsonb(r) order by r.id)
                      from public.recap_list r), '[]'::jsonb),
  'recap_items', coalesce((select jsonb_agg(to_jsonb(i) order by i.id)
                           from public.recap_item i), '[]'::jsonb)
);
```

Validate without writing anything:

```bash
go run ./cmd/migrate-supabase \
  -file supabase-export.json \
  -dry-run
```

You can also validate the existing `data.sql` directly:

```bash
go run ./cmd/migrate-supabase \
  -file data.sql \
  -dry-run
```

Only the two `public` application tables are read from the SQL dump. Supabase
managed `auth` and `storage` sections are ignored.

Import into the local database (the target should have migrations applied):

```bash
go run ./cmd/migrate-supabase \
  -file supabase-export.json \
  -database-url "$DATABASE_URL"
```

The import preserves source IDs and timestamps, normalizes legacy `F&B` labels
to `Food`, maps unknown/missing categories to `Uncategorized`, and resets the
target identity sequences. It intentionally fails on an ID conflict instead of
silently overwriting existing local data, so run it against an empty target or
review conflicts first.
