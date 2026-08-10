package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MickStanciu/go-fn/fn"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

// GroupUser represents a row from the group_users feed
type GroupUser struct {
	UserID         string    `json:"user_id" csv:"user_id" gorm:"primaryKey;size:37"`
	GroupID        string    `json:"group_id" csv:"group_id" gorm:"primaryKey;size:37"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	ExportedAt     time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// GroupUserFeed is a representation of the group_users feed
type GroupUserFeed struct{}

// Name is the name of the feed
func (f *GroupUserFeed) Name() string {
	return "group_users"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *GroupUserFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *GroupUserFeed) Model() interface{} {
	return GroupUser{}
}

// RowsModel returns the model of feed rows
func (f *GroupUserFeed) RowsModel() interface{} {
	return &[]*GroupUser{}
}

// PrimaryKey returns the primary key(s)
func (f *GroupUserFeed) PrimaryKey() []string {
	return []string{"user_id", "group_id"}
}

// Columns returns the columns of the row
func (f *GroupUserFeed) Columns() []string {
	return []string{
		"user_id",
		"group_id",
		"organisation_id",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *GroupUserFeed) Order() string {
	return "group_id, user_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *GroupUserFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*GroupUser{})
}

// Export exports the feed to the supplied exporter
func (f *GroupUserFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: false,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	// Delete the actions if already exist
	if err := exporter.DeleteRowsIfExist(f, "organisation_id = ?", orgID); err != nil {
		return events.WrapEventError(err, "delete row")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*GroupUser

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		// deduplicate rows (hotfix) because the feed Api GetUserGroups returns duplicates and this creates PK violations issues
		deDupedRows := fn.DeduplicateOrderedList(rows, func(row *GroupUser) string {
			return fmt.Sprintf("pk__%s_%s", row.UserID, row.GroupID)
		})

		if len(deDupedRows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, deDupedRows, func(batch []*GroupUser) error {
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

	req := &GetFeedRequest{
		InitialURL: "/feed/group_users",
		Params:     GetFeedParams{},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*GroupUser{})
}
