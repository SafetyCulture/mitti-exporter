package feed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
)

// TrainingCourseProgress represents a row for the feed
type TrainingCourseProgress struct {
	OpenedAt         string  `json:"opened_at" csv:"opened_at"`
	CompletedAt      string  `json:"completed_at" csv:"completed_at"`
	TotalLessons     int32   `json:"total_lessons" csv:"total_lessons"`
	CompletedLessons int32   `json:"completed_lessons" csv:"completed_lessons"`
	CourseID         string  `json:"course_id" csv:"course_id" gorm:"primarykey;column:course_id;size:64"`
	CourseExternalID string  `json:"course_external_id" csv:"course_external_id" gorm:"size:256"`
	CourseTitle      string  `json:"course_title" csv:"course_title"`
	UserEmail        string  `json:"user_email" csv:"user_email" gorm:"size:256"`
	UserFirstName    string  `json:"user_first_name" csv:"user_first_name"`
	UserLastName     string  `json:"user_last_name" csv:"user_last_name"`
	UserID           string  `json:"user_id" csv:"user_id" gorm:"primarykey;column:user_id;size:37"`
	UserExternalID   string  `json:"user_external_id" csv:"user_external_id"`
	ProgressPercent  float32 `json:"progress_percent" csv:"progress_percent"`
	Score            int32   `json:"score" csv:"score"`
	DueAt            string  `json:"due_at" csv:"due_at"`
}

// TrainingCourseProgressFeed is a representation of the feed
type TrainingCourseProgressFeed struct {
	Incremental      bool
	Limit            int
	CompletionStatus string
}

// Name is the name of the feed
func (f *TrainingCourseProgressFeed) Name() string {
	return "training_course_progresses"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *TrainingCourseProgressFeed) HasRemainingInformation() bool {
	return false
}

// Model returns the model of the feed row
func (f *TrainingCourseProgressFeed) Model() interface{} {
	return TrainingCourseProgress{}
}

// RowsModel returns the model of feed rows
func (f *TrainingCourseProgressFeed) RowsModel() interface{} {
	return &[]*TrainingCourseProgress{}
}

// PrimaryKey returns the primary key(s)
func (f *TrainingCourseProgressFeed) PrimaryKey() []string {
	return []string{"course_id", "user_id"}
}

func (f *TrainingCourseProgressFeed) Columns() []string {
	return []string{
		"opened_at",
		"completed_at",
		"total_lessons",
		"completed_lessons",
		"course_id",
		"course_external_id",
		"course_title",
		"user_email",
		"user_first_name",
		"user_last_name",
		"user_id",
		"user_external_id",
		"progress_percent",
		"score",
		"due_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *TrainingCourseProgressFeed) Order() string {
	return "opened_at"
}

func (f *TrainingCourseProgressFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*TrainingCourseProgress{})
}

func (f *TrainingCourseProgressFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*TrainingCourseProgress

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		numRows := len(rows)
		if numRows != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*TrainingCourseProgress) error {
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

	req := &GetFeedRequest{
		InitialURL: "/training/v1/feed/training-course-progress",
		Params: GetFeedParams{
			Limit:            f.Limit,
			CompletionStatus: f.CompletionStatus,
		},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	return exporter.FinaliseExport(f, &[]*TrainingCourseProgress{})
}
