package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

const feedPath = "/accounts/history/v2/feed/activity_log_events"
const accountHistorySortingColumn = "event_at"

// AccountHistory represents a row from the account history feed
type AccountHistory struct {
	ID             string    `json:"id" csv:"event_id" gorm:"primarykey;column:event_id;size:41"`
	EventAt        time.Time `json:"event_at" csv:"event_at" gorm:"autoUpdateTime"`
	Type           string    `json:"type" csv:"type"`
	UserID         string    `json:"user_id" csv:"user_id" gorm:"size:37"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	ClientClass    string    `json:"client_class" csv:"client_class" gorm:"client_class"`
	Agent          string    `json:"agent" csv:"agent" gorm:"agent"`
	Initiator      string    `json:"initiator" csv:"initiator" gorm:"size:100"`
	ExportedAt     time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// AccountHistoryFeed is a representation of the account history feed
type AccountHistoryFeed struct {
	ExportedAt  time.Time
	Incremental bool
	Limit       int
}

// Name is the name of the feed
func (f *AccountHistoryFeed) Name() string {
	return "account_histories"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *AccountHistoryFeed) HasRemainingInformation() bool {
	return false
}

// Model returns the model of the feed row
func (f *AccountHistoryFeed) Model() interface{} {
	return AccountHistory{}
}

// RowsModel returns the model of feed rows
func (f *AccountHistoryFeed) RowsModel() interface{} {
	return &[]*AccountHistory{}
}

// PrimaryKey returns the primary key(s)
func (f *AccountHistoryFeed) PrimaryKey() []string {
	return []string{"event_id"}
}

// Columns returns the columns of the row
func (f *AccountHistoryFeed) Columns() []string {
	return []string{
		"event_at",
		"type",
		"user_id",
		"organisation_id",
		"client_class",
		"agent",
		"initiator",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *AccountHistoryFeed) Order() string {
	return "event_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *AccountHistoryFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*AccountHistory{})
}

// Export exports the feed to the supplied exporter
func (f *AccountHistoryFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	s12OrgID := util.ConvertS12ToUUID(orgID)
	if s12OrgID.IsNil() {
		return fmt.Errorf("cannot convert given %q organisation ID to UUID", orgID)
	}

	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	if f.Incremental {
		f.ExportedAt = exporter.LastRecord(f, f.ExportedAt, s12OrgID.String(), accountHistorySortingColumn)
		l.Info("resuming account history feed from ", f.ExportedAt.String())
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*AccountHistory

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		numRows := len(rows)
		if numRows != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*AccountHistory) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
				return err
			}
		}

		status.IncrementStatus(f.Name(), int64(numRows), apiClient.Duration.Milliseconds())

		l.With(
			"downloaded", status.ReadCounter(f.Name()),
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{
		InitialURL: feedPath,
		Params: GetFeedParams{
			Limit:        f.Limit,
			CreatedAfter: f.ExportedAt,
		},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*AccountHistory{})
}
