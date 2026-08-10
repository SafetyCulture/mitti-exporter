package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

// User represents a row from the users feed
type User struct {
	ID             string     `json:"id" csv:"user_id" gorm:"primarykey;column:user_id;size:37"`
	OrganisationID string     `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	Email          string     `json:"email" csv:"email" gorm:"size:256"`
	Firstname      string     `json:"firstname" csv:"firstname"`
	Lastname       string     `json:"lastname" csv:"lastname"`
	Active         bool       `json:"active" csv:"active"`
	LastSeenAt     *time.Time `json:"last_seen_at" csv:"last_seen_at"`
	ExportedAt     time.Time  `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
	SeatType       string     `json:"seat_type" csv:"seat_type" gorm:"seat_type"`
	CreatedAt      *time.Time `json:"created_at" csv:"created_at" gorm:"autoCreateTime"`
}

// UserFeed is a representation of the users feed
type UserFeed struct{}

// Name is the name of the feed
func (f *UserFeed) Name() string {
	return "users"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *UserFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *UserFeed) Model() interface{} {
	return User{}
}

// RowsModel returns the model of feed rows
func (f *UserFeed) RowsModel() interface{} {
	return &[]*User{}
}

// PrimaryKey returns the primary key(s)
func (f *UserFeed) PrimaryKey() []string {
	return []string{"user_id"}
}

// Columns returns the columns of the row
func (f *UserFeed) Columns() []string {
	return []string{
		"organisation_id",
		"email",
		"firstname",
		"lastname",
		"active",
		"last_seen_at",
		"exported_at",
		"seat_type",
		"created_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *UserFeed) Order() string {
	return "user_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *UserFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*User{})
}

// Export exports the feed to the supplied exporter
func (f *UserFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: !exporter.SupportsUpsert(),
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*User

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*User) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
				return err
			}
		}

		status.UpdateStatus(f.Name(), resp.Metadata.RemainingRecords, exporter.GetDuration().Milliseconds())

		l.With(
			"estimated_remaining", resp.Metadata.RemainingRecords,
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{InitialURL: "/feed/users", Params: GetFeedParams{}}
	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*User{})
}
