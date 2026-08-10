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

// Issue represents a row from the issues feed
type Issue struct {
	ID              string     `json:"id" csv:"id" gorm:"primarykey;column:id;size:36"`
	Title           string     `json:"title" csv:"title"`
	Description     string     `json:"description" csv:"description"`
	CreatorID       string     `json:"creator_id" csv:"creator_id"`
	CreatorUserName string     `json:"creator_user_name" csv:"creator_user_name"`
	CreatedAt       time.Time  `json:"created_at" csv:"created_at"`
	DueAt           *time.Time `json:"due_at,omitempty" csv:"due_at"`
	Priority        string     `json:"priority" csv:"priority"`
	Status          string     `json:"status" csv:"status"`
	TemplateID      string     `json:"template_id" csv:"template_id"`
	InspectionID    string     `json:"inspection_id" csv:"inspection_id"`
	InspectionName  string     `json:"inspection_name" csv:"inspection_name"`
	SiteID          string     `json:"site_id" csv:"site_id"`
	SiteName        string     `json:"site_name" csv:"site_name"`
	LocationName    string     `json:"location_name" csv:"location_name"`
	CategoryID      string     `json:"category_id" csv:"category_id"`
	CategoryLabel   string     `json:"category_label" csv:"category_label"`
	ModifiedAt      time.Time  `json:"modified_at" csv:"modified_at"`
	CompletedAt     *time.Time `json:"completed_at" csv:"completed_at"`
	AssetID         string     `json:"asset_id" csv:"asset_id" gorm:"size:36"`
	UniqueID        string     `json:"unique_id" csv:"unique_id"`
	OccurredAt      *time.Time `json:"occurred_at" csv:"occurred_at"`
}

// IssueFeed is a representation of the issues feed
type IssueFeed struct {
	Limit       int
	Incremental bool
}

// Name returns the name of the feed
func (f *IssueFeed) Name() string {
	return "issues"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *IssueFeed) HasRemainingInformation() bool {
	return false
}

// Model returns the model of the feed row
func (f *IssueFeed) Model() interface{} {
	return Issue{}
}

// RowsModel returns the model of the feed rows
func (f *IssueFeed) RowsModel() interface{} {
	return &[]*Issue{}
}

// PrimaryKey return the primary key
func (f *IssueFeed) PrimaryKey() []string {
	return []string{"id"}
}

// Columns returns the columns of the row
func (f *IssueFeed) Columns() []string {
	return []string{
		"id", "title", "description", "creator_id", "creator_user_name",
		"created_at", "due_at", "priority", "status", "template_id",
		"inspection_id", "inspection_name", "site_id", "site_name",
		"location_name", "category_id", "category_label", "modified_at",
		"completed_at", "asset_id", "unique_id", "occurred_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *IssueFeed) Order() string {
	return "id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *IssueFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*Issue{})
}

// Export exports the feed to the supplied exporter
func (f *IssueFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	var drainFn = func(resp *GetFeedResponse) error {
		var rows []*Issue

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		numRows := len(rows)
		if numRows != 0 {
			// Calculate the size of the batch we can insert into the DB at once.
			// Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*Issue) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
				return err
			}
		}

		// note: this feed api doesn't return remaining items
		status.IncrementStatus(f.Name(), int64(numRows), apiClient.Duration.Milliseconds())

		l.With(
			"downloaded", status.ReadCounter(f.Name()),
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	var req = &GetFeedRequest{
		InitialURL: "/feed/issues",
		Params: GetFeedParams{
			Limit: f.Limit,
		},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*Issue{})
}
