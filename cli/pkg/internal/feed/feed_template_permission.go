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

// TemplatePermission represents a row from the template_permissions feed
type TemplatePermission struct {
	ID             string `json:"id" csv:"permission_id" gorm:"primarykey;column:permission_id;size:375"`
	TemplateID     string `json:"template_id" csv:"template_id" gorm:"size:100"`
	Permission     string `json:"permission" csv:"permission" gorm:"size:10"`
	AssigneeID     string `json:"assignee_id" csv:"assignee_id" gorm:"size:256"`
	AssigneeType   string `json:"assignee_type" csv:"assignee_type" gorm:"size:10"`
	OrganisationID string `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
}

// TemplatePermissionFeed is a representation of the template_permissions feed
type TemplatePermissionFeed struct {
	ModifiedAfter time.Time
	Incremental   bool
}

// Name is the name of the feed
func (f *TemplatePermissionFeed) Name() string {
	return "template_permissions"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *TemplatePermissionFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *TemplatePermissionFeed) Model() interface{} {
	return TemplatePermission{}
}

// RowsModel returns the model of feed rows
func (f *TemplatePermissionFeed) RowsModel() interface{} {
	return &[]*TemplatePermission{}
}

// PrimaryKey returns the primary key(s)
func (f *TemplatePermissionFeed) PrimaryKey() []string {
	return []string{"permission_id"}
}

// Columns returns the columns of the row
func (f *TemplatePermissionFeed) Columns() []string {
	return []string{
		"template_id",
		"permission",
		"assignee_id",
		"assignee_type",
		"organisation_id",
	}
}

// Order returns the ordering when retrieving an export
func (f *TemplatePermissionFeed) Order() string {
	return "permission_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *TemplatePermissionFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*TemplatePermission{})
}

// Export exports the feed to the supplied exporter
func (f *TemplatePermissionFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Always truncate. This data must be refreshed in order to be accurate
		Truncate: true,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*TemplatePermission

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*TemplatePermission) error {
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
		InitialURL: "/feed/template_permissions",
		Params: GetFeedParams{
			ModifiedAfter: f.ModifiedAfter,
		},
	}
	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*TemplatePermission{})
}
