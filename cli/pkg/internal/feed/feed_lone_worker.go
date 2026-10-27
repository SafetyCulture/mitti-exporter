package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
)

// LoneWorkerJob represents a row from the Lone Worker jobs feed. The API returns one record per job that
// ended, either finished or canceled.
type LoneWorkerJob struct {
	ID                          string    `json:"id" csv:"id" gorm:"primarykey;column:id;size:36"`
	Code                        string    `json:"code" csv:"code"`
	Name                        string    `json:"name" csv:"name"`
	Status                      string    `json:"status" csv:"status" gorm:"size:20"`
	JobTypeID                   string    `json:"job_type_id" csv:"job_type_id" gorm:"size:36"`
	JobTypeName                 string    `json:"job_type_name" csv:"job_type_name"`
	GroupID                     string    `json:"group_id" csv:"group_id" gorm:"size:36"`
	UserID                      string    `json:"user_id" csv:"user_id" gorm:"size:36"`
	TravelStartedAt             string    `json:"travel_started_at" csv:"travel_started_at"`
	StartedAt                   string    `json:"started_at" csv:"started_at"`
	FinishedAt                  string    `json:"finished_at" csv:"finished_at"`
	CanceledAt                  string    `json:"canceled_at" csv:"canceled_at"`
	PlannedDurationMinutes      int32     `json:"planned_duration_minutes" csv:"planned_duration_minutes"`
	ActualDurationSeconds       *int32    `json:"actual_duration_seconds" csv:"actual_duration_seconds"`
	PanicReason                 string    `json:"panic_reason" csv:"panic_reason" gorm:"size:32"`
	StartAddress                string    `json:"start_address" csv:"start_address"`
	StartLatitude               *float64  `json:"start_latitude" csv:"start_latitude"`
	StartLongitude              *float64  `json:"start_longitude" csv:"start_longitude"`
	StartLocationNotes          string    `json:"start_location_notes" csv:"start_location_notes"`
	FinishAddress               string    `json:"finish_address" csv:"finish_address"`
	FinishLatitude              *float64  `json:"finish_latitude" csv:"finish_latitude"`
	FinishLongitude             *float64  `json:"finish_longitude" csv:"finish_longitude"`
	FinishLocatedAt             string    `json:"finish_located_at" csv:"finish_located_at"`
	SatelliteDeviceID           string    `json:"satellite_device_id" csv:"satellite_device_id" gorm:"size:36"`
	SatelliteDeviceSerialNumber string    `json:"satellite_device_serial_number" csv:"satellite_device_serial_number"`
	ModifiedAt                  time.Time `json:"modified_at" csv:"modified_at" gorm:"index:idx_lwj_modified_at,sort:desc"`
	ExportedAt                  time.Time `json:"exported_at" csv:"exported_at" gorm:"index:idx_lwj_modified_at;autoUpdateTime"`
	// OrganisationID is not part of the API record: the endpoint is already scoped to the credential's
	// organisation, and the exporter fills this column from the organisation it is exporting for.
	OrganisationID string `json:"-" csv:"organisation_id" gorm:"index:idx_lwj_modified_at;size:37"`
}

// TableName returns the table the feed writes to.
func (LoneWorkerJob) TableName() string {
	return "lone_worker_jobs"
}

// LoneWorkerJobFeed is a representation of the Lone Worker jobs feed.
type LoneWorkerJobFeed struct {
	ModifiedAfter time.Time
	Incremental   bool
	Limit         int
}

// Name is the name of the feed
func (f *LoneWorkerJobFeed) Name() string {
	return "lone_worker_jobs"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *LoneWorkerJobFeed) HasRemainingInformation() bool {
	return false
}

// Model returns the model of the feed row
func (f *LoneWorkerJobFeed) Model() interface{} {
	return LoneWorkerJob{}
}

// RowsModel returns the model of feed rows
func (f *LoneWorkerJobFeed) RowsModel() interface{} {
	return &[]*LoneWorkerJob{}
}

// PrimaryKey returns the primary key(s)
func (f *LoneWorkerJobFeed) PrimaryKey() []string {
	return []string{"id"}
}

// Columns returns the columns of the row
func (f *LoneWorkerJobFeed) Columns() []string {
	return []string{
		"code",
		"name",
		"status",
		"job_type_id",
		"job_type_name",
		"group_id",
		"user_id",
		"travel_started_at",
		"started_at",
		"finished_at",
		"canceled_at",
		"planned_duration_minutes",
		"actual_duration_seconds",
		"panic_reason",
		"start_address",
		"start_latitude",
		"start_longitude",
		"start_location_notes",
		"finish_address",
		"finish_latitude",
		"finish_longitude",
		"finish_located_at",
		"satellite_device_id",
		"satellite_device_serial_number",
		"modified_at",
		"exported_at",
		"organisation_id",
	}
}

// Order returns the ordering when retrieving an export
func (f *LoneWorkerJobFeed) Order() string {
	return "id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *LoneWorkerJobFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*LoneWorkerJob{})
}

// Export exports the feed to the supplied exporter
func (f *LoneWorkerJobFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	var err error
	f.ModifiedAfter, err = exporter.LastModifiedAt(f, f.ModifiedAfter, orgID)
	if err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDB, false, "unable to load modified after")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*LoneWorkerJob

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		for _, row := range rows {
			row.OrganisationID = orgID
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*LoneWorkerJob) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
				return err
			}
		}

		l.With(
			"batch_size", len(rows),
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{
		InitialURL: "/loneworker/v1/feed/jobs",
		Params: GetFeedParams{
			ModifiedAfter: f.ModifiedAfter,
			Limit:         f.Limit,
		},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}

	return exporter.FinaliseExport(f, &[]*LoneWorkerJob{})
}

// NewLoneWorkerJobFeed creates a new Lone Worker jobs feed.
func NewLoneWorkerJobFeed(limit int) Feed {
	return &LoneWorkerJobFeed{
		Limit: limit,
	}
}
