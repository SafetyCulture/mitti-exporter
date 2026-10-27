---
name: add-data-feed
description: Add or extend a data feed (a new exported table) in the Mitti Exporter CLI and desktop app. Use when asked to export a new Mitti API feed endpoint, add a table, change how an existing feed syncs (incremental, deletes, limits), or check which documented feeds the exporter is missing.
---

# Adding a data feed to the Mitti Exporter

Feeds live in `cli/pkg/internal/feed/`. A feed turns one paginated `GET .../feed/...` endpoint into one table (SQL) or one set of CSV files. The reference for what exists is https://developer.mitti.com/llms.txt (Data Feeds section); each page's `.md` variant contains the OpenAPI definition.

## 1. Read the endpoint spec first

Fetch `https://developer.mitti.com/reference/<page>.md` and read the `json` block. Do not trust the HTML page: some pages are stubs with no schema (for example `feedservice_feeduserdocuments`). Write down:

- exact path, and which query params exist (`limit`, `modified_after`, `template`, `start_date`, ...)
- the row schema, and which fields are not plain strings (arrays, ints, bools)
- the metadata object: does it include `remaining_records`? (`FeedMetadataSimple` does not)
- the default page size and any default filter window (the scheduling occurrences feed defaults to now-5 months..now+1 month)

## 2. Choose the implementation

**Flat list, full refresh, nothing special: use the generic `rowFeed[T]`** (`feed_row.go`). This covers most new feeds. Each family file (for example `feed_investigation.go`) holds the row structs plus one `NewXxxFeed(...)` constructor.

Write a bespoke feed (copy `feed_action_assignees.go` or `feed_action.go`) only if you need one of:

| Need | Existing example | Notes |
|---|---|---|
| Incremental sync via `modified_after` | `feed_action.go`, `feed_inspection.go` | `LastModifiedAt` requires `organisation_id` and `modified_at` columns |
| Incremental without `modified_at` | `feed_action_timeline_item.go`, `feed_account_history.go` | uses `LastRecord(feed, fallback, orgID, sortColumn)`, which also filters on `organisation_id` |
| Soft deletes | `feed_inspection.go`, `feed_action.go` | read activity-log delete events, then `UpdateRows(deleted=true)` |
| Child rows replaced per parent | `feed_action_assignees.go` | `DeleteRowsIfExist("action_id IN ?", ids)` before `WriteRows` |
| Replace everything for the org | `feed_group_user.go` | one up-front `DeleteRowsIfExist("organisation_id = ?", orgID)` |
| Windowed or resumable download | `feed_schedule_occurrence.go` | experimental; caps each run to a 31-day window |
| Media download | `feed_inspection_item.go` | goroutines, semaphore of 10 |
| JSON reshaping | `feed_sheqsy_*.go` | gjson/sjson flattening; SHEQSY feeds are not paged by `next_page` |

## 3. Define the row struct

- Tags: `json:"<api field>" csv:"<column>" gorm:"..."`. The csv tag is the column name.
- **Every string primary-key column needs `size:N`** or MySQL / SQL Server reject the key. Use `size:375` for long composite-style IDs. Keep composite keys short (MySQL key limit, SQL Server 900 bytes).
- Fields that may be empty or absent (dates like `issue_date`, optional timestamps): use `string`, not `time.Time`. An empty string fails `time.Time` JSON parsing and fails the whole feed. Use `time.Time` only for fields the spec guarantees, and `*int32` / `*time.Time` for nullable numbers and dates.
- Arrays: use `JSONArray` (`feed_row.go`). It stores the compact JSON text in one column / CSV cell.
- **Add `func (X) TableName() string` returning the feed name.** `AutoMigrate` and truncate use gorm's name derived from the struct type, while writes use `feed.Name()`. If they differ you get `no such table` (`ScheduleV2` becomes `schedule_v2s`). `TestRowFeeds_columns_should_match_row_model` enforces this for rowFeeds.

## 4. Write the constructor

```go
func NewThingFeed(limit int) Feed {
	return &rowFeed[Thing]{
		name:         "things",                  // table name, must equal TableName()
		initialURL:   "/things/v1/feed/things",
		primaryKey:   []string{"thing_id"},
		columns:      []string{"title", ...},    // every column that is NOT in primaryKey
		order:        "thing_id",                // used by the CSV exporter to page rows; columns must exist
		hasRemaining: false,                     // true only if metadata has remaining_records
		params:       GetFeedParams{Limit: limit},
		key:          func(r *Thing) string { return r.ThingID }, // dedup key, join composite keys with "|"
	}
}
```

- `columns` is what gets updated on conflict. A column left out is never refreshed on update.
- `hasRemaining` must match the API. It decides whether progress uses `UpdateStatus(remaining)` or `IncrementStatus`.
- If you need a new query param, add it to `GetFeedParams` in `drain_feed.go` with a `url:"...,omitempty"` tag. Booleans that must be sent as `false` need `*bool`.
- rowFeed always truncates and re-writes (full refresh). That is also why deleted records disappear.

## 5. Register it

1. `feed_exporter.go` `GetFeeds()`: append the constructor. Pass a limit from config if a related feed has one (`ExportActionLimit`, `ExportIssueLimit`, `ExportAssetLimit`), otherwise `defaultFeedLimit` (100).
2. **UI, both lists**, or the desktop app will not export or offer the table:
   - `ui/frontend/src/lib/utils.js` `allTables` (this is what the app sends when nothing is selected)
   - `ui/frontend/src/routes/DataSetFilter.svelte` `data` (left/right pairs; `null` is allowed for an odd last cell)
3. If the feed needs a **new config option**, plumb it through all of these (several existing options skip steps, do not copy that):
   - field + default in `ExporterConfiguration` / `BuildConfigurationWithDefaults` (`cli/pkg/api/configuration_manager.go`), and a cap in `ApplySafetyGuards` if the API has a max page size (actions and issues are capped at 100, course progress at 1000)
   - `ToExporterConfig()` and `ExporterFeedCfg`
   - `MapViperConfigToExporterConfiguration` in `cli/cmd/mitti-exporter/cmd/export/export.go` (otherwise CLI flags and viper config are silently ignored)
   - optional flag in `cmd/mitti-exporter/cmd/root.go` (`BindPFlag`) and the template `cmd/configure/mitti-exporter.yaml`

## 6. Tests and fixtures (required, existing tests fail without them)

- **Schema fixture:** `cli/pkg/internal/feed/fixtures/schemas/formatted/<feed name>.txt`. `TestSchemaWriter_should_write_schema` reads one for every registered feed. Generate it from the real output of `exporter.CreateSchema` + `WriteSchema`, then review it.
- **HTTP mock:** every export test registers gock mocks and fails on an unmatched request. Add the endpoint to `newFeedFixtures` in `cli/pkg/api/feed_row_test.go` (path + one-row body). That drives both the per-feed export test and the empty-page mocks in `mock_feeds_test.go`.
- The column test (`feed_row_test.go`) checks that `primaryKey + columns` equals the struct's csv tags and that the count of rowFeeds matches. Update the expected count when you add one.
- Run: `cd cli && go test ./...` (the `pkg/api` package takes about 75s).

## 7. Verify end to end

1. `cd cli && go build -o ../bin/mitti-exporter-<name> ./cmd/mitti-exporter`
2. Run `sqlite` or `csv` with `--tables <your table>` and a config file. **Set the API URL in the config file** (`api.url`), because `--api-url` is not mapped by `MapViperConfigToExporterConfiguration` and is currently ignored.
3. Check row counts against the Mitti web app, then run it twice to confirm no duplicates.
4. Desktop app: `cd ui && wails build`. Settings live in `~/Library/Application Support/mitti-exporter/` (or the legacy `safetyculture-exporter/` directory).

## Known pitfalls

- Table selection is by exact feed name. Unknown `--tables` values are silently ignored; an empty selection exports everything.
- Feeds run concurrently (default 10) and writes are serialised by a mutex in the SQL exporter.
- The CSV exporter is SQLite plus a file rewrite step, so upsert behaviour is identical to SQL.
- Bulk inserts are batched with `ParameterLimit() / (columns + 4)`. Very wide rows on SQL Server (limit 2100 parameters) get small batches.
- Legacy and new endpoints can coexist (scheduling). Use a separate table name (`_v2`) when the new feed's IDs or semantics differ, so orgs on the old experience keep working.
- Pagination follows `metadata.next_page` as a full URL. A feed that paginates differently needs its own drain loop (see `drain_account_activity_log.go`, `feed_sheqsy_activity.go`).
