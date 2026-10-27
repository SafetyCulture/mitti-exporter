package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/MickStanciu/go-fn/fn"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
)

// JSONArray holds a JSON array returned by the API as its compact JSON text,
// so the whole array fits in a single text column or CSV cell.
type JSONArray string

// UnmarshalJSON stores the compacted JSON text of the array. A null value is stored as an empty string.
func (a *JSONArray) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*a = ""
		return nil
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		return fmt.Errorf("JSONArray.UnmarshalJSON: compact json array: %w", err)
	}
	*a = JSONArray(buf.String())
	return nil
}

// rowFeed is a generic feed for endpoints that return a flat list of rows of type T.
// Every export is a full refresh: the table is truncated and re-populated, which also means
// records deleted upstream don't linger in the export.
type rowFeed[T any] struct {
	name       string
	initialURL string
	primaryKey []string
	columns    []string
	order      string
	// hasRemaining is true if the endpoint reports remaining_records in the response metadata
	hasRemaining bool
	params       GetFeedParams
	// key returns the primary key of a row. It is used to deduplicate rows returned by the API
	key func(*T) string
}

// Name is the name of the feed
func (f *rowFeed[T]) Name() string {
	return f.name
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *rowFeed[T]) HasRemainingInformation() bool {
	return f.hasRemaining
}

// Model returns the model of the feed row
func (f *rowFeed[T]) Model() interface{} {
	var row T
	return row
}

// RowsModel returns the model of feed rows
func (f *rowFeed[T]) RowsModel() interface{} {
	return &[]*T{}
}

// PrimaryKey returns the primary key(s)
func (f *rowFeed[T]) PrimaryKey() []string {
	return f.primaryKey
}

// Columns returns the columns of the row that are not part of the primary key
func (f *rowFeed[T]) Columns() []string {
	return f.columns
}

// Order returns the ordering when retrieving an export
func (f *rowFeed[T]) Order() string {
	return f.order
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *rowFeed[T]) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*T{})
}

// Export exports the feed to the supplied exporter
func (f *rowFeed[T]) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		Truncate: true,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*T

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		// deduplicate rows because a page containing the same primary key twice causes PK violations
		rows = fn.DeduplicateOrderedList(rows, f.key)

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + len(f.PrimaryKey()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*T) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
				return err
			}
		}

		if f.hasRemaining {
			status.UpdateStatus(f.Name(), resp.Metadata.RemainingRecords, exporter.GetDuration().Milliseconds())
		} else {
			status.IncrementStatus(f.Name(), int64(len(rows)), apiClient.Duration.Milliseconds())
		}

		l.With(
			"estimated_remaining", resp.Metadata.RemainingRecords,
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{
		InitialURL: f.initialURL,
		Params:     f.params,
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	if err := exporter.FinaliseExport(f, &[]*T{}); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("finalise export of feed %q", f.Name()))
	}
	return nil
}
