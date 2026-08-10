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

// Group represents a row from the groups feed
type Group struct {
	ID             string    `json:"id" csv:"group_id" gorm:"primarykey;column:group_id;size:37"`
	Name           string    `json:"name" csv:"name"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	ExportedAt     time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// GroupFeed is a representation of the groups feed
type GroupFeed struct{}

// Name is the name of the feed
func (f *GroupFeed) Name() string {
	return "groups"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *GroupFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *GroupFeed) Model() interface{} {
	return Group{}
}

// RowsModel returns the model of feed rows
func (f *GroupFeed) RowsModel() interface{} {
	return &[]*Group{}
}

// PrimaryKey returns the primary key(s)
func (f *GroupFeed) PrimaryKey() []string {
	return []string{"group_id"}
}

// Columns returns the columns of the row
func (f *GroupFeed) Columns() []string {
	return []string{
		"name",
		"organisation_id",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *GroupFeed) Order() string {
	return "group_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *GroupFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*Group{})
}

// Export exports the feed to the supplied exporter
func (f *GroupFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
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
		var rows []*Group

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*Group) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
				return err
			}
		}

		status.UpdateStatus(f.Name(), resp.Metadata.RemainingRecords, apiClient.Duration.Milliseconds())

		l.With(
			"estimated_remaining", resp.Metadata.RemainingRecords,
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{
		InitialURL: "/feed/groups",
		Params:     GetFeedParams{},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*Group{})
}
