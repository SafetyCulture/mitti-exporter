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

// Template represents a row from the templates feed
type Template struct {
	ID             string    `json:"id" csv:"template_id" gorm:"primarykey;column:template_id;size:100"`
	Archived       bool      `json:"archived" csv:"archived"`
	Name           string    `json:"name" csv:"name"`
	Description    string    `json:"description" csv:"description"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"index:idx_tml_modified_at;size:37"`
	OwnerName      string    `json:"owner_name" csv:"owner_name"`
	OwnerID        string    `json:"owner_id" csv:"owner_id" gorm:"size:37"`
	AuthorName     string    `json:"author_name" csv:"author_name"`
	AuthorID       string    `json:"author_id" csv:"author_id" gorm:"size:37"`
	CreatedAt      time.Time `json:"created_at" csv:"created_at"`
	ModifiedAt     time.Time `json:"modified_at" csv:"modified_at" gorm:"index:idx_tml_modified_at,sort:desc"`
	ExportedAt     time.Time `json:"exported_at" csv:"exported_at" gorm:"index:idx_tml_modified_at;autoUpdateTime"`
}

// TemplateFeed is a representation of the templates feed
type TemplateFeed struct {
	ModifiedAfter time.Time
	Incremental   bool
}

// Name is the name of the feed
func (f *TemplateFeed) Name() string {
	return "templates"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *TemplateFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *TemplateFeed) Model() interface{} {
	return Template{}
}

// RowsModel returns the model of feed rows
func (f *TemplateFeed) RowsModel() interface{} {
	return &[]*Template{}
}

// PrimaryKey returns the primary key(s)
func (f *TemplateFeed) PrimaryKey() []string {
	return []string{"template_id"}
}

// Columns returns the columns of the row
func (f *TemplateFeed) Columns() []string {
	return []string{
		"archived",
		"name",
		"description",
		"organisation_id",
		"owner_name",
		"owner_id",
		"author_name",
		"author_id",
		"created_at",
		"modified_at",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *TemplateFeed) Order() string {
	return "modified_at ASC, template_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *TemplateFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*Template{})
}

// Export exports the feed to the supplied exporter
func (f *TemplateFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*Template

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*Template) error {
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

	var err error
	f.ModifiedAfter, err = exporter.LastModifiedAt(f, f.ModifiedAfter, orgID)
	if err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDB, false, "unable to load modified after")
	}

	l.With(
		"modified_after", f.ModifiedAfter,
	).Info("exporting")

	req := &GetFeedRequest{
		InitialURL: "/feed/templates",
		Params: GetFeedParams{
			ModifiedAfter: f.ModifiedAfter,
		},
	}
	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*Template{})
}
