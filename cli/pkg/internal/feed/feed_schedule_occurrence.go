package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MickStanciu/go-fn/fn"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

// ScheduleOccurrence represents a row from the schedule_occurrences feed
type ScheduleOccurrence struct {
	ID               string     `json:"id" csv:"id" gorm:"primarykey;size:128"`
	ScheduleID       string     `json:"schedule_id" csv:"schedule_id" gorm:"size:45"`
	OccurrenceID     string     `json:"occurrence_id" csv:"occurrence_id" gorm:"size:30"`
	TemplateID       string     `json:"template_id" csv:"template_id" gorm:"size:100"`
	OrganisationID   string     `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	StartTime        *time.Time `json:"start_time" csv:"start_time"`
	DueTime          *time.Time `json:"due_time" csv:"due_time"`
	MissTime         *time.Time `json:"miss_time" csv:"miss_time"`
	OccurrenceStatus string     `json:"occurrence_status" csv:"occurrence_status" gorm:"size:20"`
	AuditID          *string    `json:"audit_id" csv:"audit_id" gorm:"size:100"`
	CompletedAt      *time.Time `json:"completed_at" csv:"completed_at"`
	ExportedAt       time.Time  `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
	UserID           string     `json:"user_id" csv:"user_id" gorm:"size:37"`
	AssigneeStatus   string     `json:"assignee_status" csv:"assignee_status" gorm:"size:20"`
	Note             string     `json:"note" csv:"note" gorm:"size:255"`
}

// ScheduleOccurrenceFeed is a representation of the schedule_occurrences feed
type ScheduleOccurrenceFeed struct {
	TemplateIDs    []string
	StartDate      time.Time
	ResumeDownload bool // EXPERIMENTAL: we don't have modified_at from the backend. We will use start_time
}

// Name is the name of the feed
func (f *ScheduleOccurrenceFeed) Name() string {
	return "schedule_occurrences"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *ScheduleOccurrenceFeed) HasRemainingInformation() bool {
	return true
}

// RowsModel returns the model of feed rows
func (f *ScheduleOccurrenceFeed) RowsModel() interface{} {
	return &[]*ScheduleOccurrence{}
}

// Model returns the model of the feed row
func (f *ScheduleOccurrenceFeed) Model() interface{} {
	return ScheduleOccurrence{}
}

// PrimaryKey returns the primary key(s)
func (f *ScheduleOccurrenceFeed) PrimaryKey() []string {
	return []string{"id"}
}

// Columns returns the columns of the row
func (f *ScheduleOccurrenceFeed) Columns() []string {
	return []string{
		"schedule_id",
		"occurrence_id",
		"template_id",
		"organisation_id",
		"start_time",
		"due_time",
		"miss_time",
		"occurrence_status",
		"audit_id",
		"completed_at",
		"user_id",
		"assignee_status",
	}
}

// Order returns the ordering when retrieving an export
func (f *ScheduleOccurrenceFeed) Order() string {
	return "occurrence_id ASC, schedule_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *ScheduleOccurrenceFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*ScheduleOccurrence{})
}

// Export exports the feed to the supplied exporter
func (f *ScheduleOccurrenceFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.ResumeDownload,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	if f.ResumeDownload {
		f.StartDate = exporter.LastRecord(f, f.StartDate, orgID, "start_time")
		l.Info("resuming schedule occurrences from ", f.StartDate.String())
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*ScheduleOccurrence

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		// deduplicate rows (hotfix) because the feed might return duplicates in the same page and this creates PK violations issues
		deDupedRows := fn.DeduplicateOrderedList(rows, func(row *ScheduleOccurrence) string {
			return fmt.Sprintf("pk__%s", row.ID)
		})

		if len(deDupedRows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, deDupedRows, func(batch []*ScheduleOccurrence) error {
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
		InitialURL: "/feed/schedule_occurrences",
		Params: GetFeedParams{
			TemplateIDs: f.TemplateIDs,
		},
	}

	if !f.StartDate.IsZero() {
		req.Params.StartDate = f.StartDate
		req.Params.EndDate = f.StartDate.Add(31 * day) // hard limit to 31 days at a time
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*ScheduleOccurrence{})
}

var day = 24 * 60 * 60 * time.Second
