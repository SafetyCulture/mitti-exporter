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

// Site represents a row from the sites feed
type Site struct {
	ID             string    `json:"id" csv:"site_id" gorm:"primarykey;column:site_id;size:41"`
	Name           string    `json:"name" csv:"name"`
	CreatorID      string    `json:"creator_id" csv:"creator_id" gorm:"size:37"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	ExportedAt     time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
	Deleted        bool      `json:"deleted" csv:"deleted" gorm:"deleted"`
	SiteUUID       string    `json:"site_uuid" csv:"site_uuid" gorm:"size:36"`
	MetaLabel      string    `json:"meta_label" csv:"meta_label" gorm:"size:36"`
	ParentID       string    `json:"parent_id" csv:"parent_id" gorm:"size:41"`
}

// SiteFeed is a representation of the sites feed
type SiteFeed struct {
	IncludeDeleted       bool
	IncludeFullHierarchy bool
}

// Name is the name of the feed
func (f *SiteFeed) Name() string {
	return "sites"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *SiteFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *SiteFeed) Model() interface{} {
	return Site{}
}

// RowsModel returns the model of feed rows
func (f *SiteFeed) RowsModel() interface{} {
	return &[]*Site{}
}

// PrimaryKey returns the primary key(s)
func (f *SiteFeed) PrimaryKey() []string {
	return []string{"site_id"}
}

// Columns returns the columns of the row
func (f *SiteFeed) Columns() []string {
	return []string{
		"name",
		"creator_id",
		"organisation_id",
		"exported_at",
		"deleted",
		"site_uuid",
		"meta_label",
		"parent_id",
	}
}

// Order returns the ordering when retrieving an export
func (f *SiteFeed) Order() string {
	return "site_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *SiteFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*Site{})
}

// Export exports the feed to the supplied exporter
func (f *SiteFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
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
		var rows []*Site

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*Site) error {
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

	showOnlyLeafNodes := !f.IncludeFullHierarchy
	req := &GetFeedRequest{
		InitialURL: "/feed/sites",
		Params: GetFeedParams{
			IncludeDeleted:    f.IncludeDeleted,
			ShowOnlyLeafNodes: &showOnlyLeafNodes,
		},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*Site{})
}
