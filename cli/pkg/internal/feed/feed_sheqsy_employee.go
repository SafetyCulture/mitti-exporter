package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SheqsyEmployee represents a user in sheqsy
type SheqsyEmployee struct {
	EmployeeUID             string    `json:"employeeUId" csv:"employee_uid" gorm:"primarykey;column:employee_uid;size:36"`
	EmployeeID              int       `json:"employeeId" csv:"employee_id" gorm:"column:employee_id"`
	ExternalID              *string   `json:"externalId" csv:"external_id" gorm:"column:external_id"`
	FirstName               string    `json:"firstName" csv:"first_name" gorm:"column:first_name"`
	LastName                string    `json:"lastName" csv:"last_name" gorm:"column:last_name"`
	Email                   string    `json:"email" csv:"email" gorm:"column:email"`
	AcceptedActivitiesCount int       `json:"acceptedActivitiesCount" csv:"accepted_activities_count" gorm:"column:accepted_activities_count"`
	PendingActivitiesCount  int       `json:"pendingActivitiesCount" csv:"pending_activities_count" gorm:"column:pending_activities_count"`
	IsInPanic               bool      `json:"isInPanic" csv:"is_in_panic" gorm:"column:is_in_panic"`
	Status                  string    `json:"status" csv:"status" gorm:"column:status"`
	LastActivityDateTimeUTC string    `json:"lastActivityDateTimeUTC" csv:"last_activity_date_time_utc" gorm:"column:last_activity_date_time_utc"`
	Departments             string    `json:"departments" csv:"departments" gorm:"column:departments"`
	ExportedAt              time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// SheqsyEmployeeFeed is a representation of the users feed
type SheqsyEmployeeFeed struct{}

// Name is the name of the feed
func (f *SheqsyEmployeeFeed) Name() string {
	return "sheqsy_employees"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *SheqsyEmployeeFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *SheqsyEmployeeFeed) Model() interface{} {
	return SheqsyEmployee{}
}

// RowsModel returns the model of feed rows
func (f *SheqsyEmployeeFeed) RowsModel() interface{} {
	return &[]*SheqsyEmployee{}
}

// PrimaryKey returns the primary key(s)
func (f *SheqsyEmployeeFeed) PrimaryKey() []string {
	return []string{"employee_uid"}
}

// Columns returns the columns of the row
func (f *SheqsyEmployeeFeed) Columns() []string {
	return []string{
		"employee_id",
		"external_id",
		"first_name",
		"last_name",
		"email",
		"accepted_activities_count",
		"pending_activities_count",
		"is_in_panic",
		"status",
		"last_activity_date_time_utc",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *SheqsyEmployeeFeed) Order() string {
	return "employee_uid"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *SheqsyEmployeeFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*SheqsyEmployee{})
}

// Export exports the feed to the supplied exporter
func (f *SheqsyEmployeeFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, companyID string) error {
	log := logger.GetLogger().With("feed", f.Name(), "org_id", companyID)

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: !exporter.SupportsUpsert(),
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	var rows []*SheqsyEmployee

	resp, err := httpapi.ExecuteGet[json.RawMessage](ctx, apiClient, fmt.Sprintf("/SheqsyIntegrationApi/api/v3/companies/%s/employees", companyID), nil)
	if err != nil {
		return fmt.Errorf("fetch data: %w", err)
	}

	var respBytes []byte
	respBytes = *resp

	gjson.ParseBytes(respBytes).ForEach(func(key, value gjson.Result) bool {
		// These timestamps aren't parsable as RFC 3339 strings, so we have to munge them to that format
		lastSeen := value.Get("lastActivityDateTimeUTC").String()
		if len(lastSeen) != 0 {
			respBytes, err = sjson.SetBytes(
				respBytes,
				fmt.Sprintf("%d.lastActivityDateTimeUTC", key.Int()),
				lastSeen+"Z",
			)
			if err != nil {
				log.Errorf("failed to update lastActivityDateTimeUTC: %v", err)
				return false
			}
		}

		var departments []string
		value.Get("departments.#.name").ForEach(func(key, value gjson.Result) bool {
			departments = append(departments, value.String())
			return true
		})
		respBytes, err = sjson.SetBytes(
			respBytes,
			fmt.Sprintf("%d.departments", key.Int()),
			strings.Join(departments, ","),
		)
		if err != nil {
			log.Errorf("failed to set departments: %v", err)
			return false
		}
		return true
	})

	if err := json.Unmarshal(respBytes, &rows); err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
	}

	if len(rows) != 0 {
		// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
		batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
		err := util.SplitSliceInBatch(batchSize, rows, func(batch []*SheqsyEmployee) error {
			if err := exporter.WriteRows(f, batch); err != nil {
				return events.WrapEventError(err, "write rows")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	log.With(
		"estimated_remaining", 0,
		"duration_ms", apiClient.Duration.Milliseconds(),
		"export_duration_ms", exporter.GetDuration().Milliseconds(),
	).Info("export batch complete")

	return exporter.FinaliseExport(f, &[]*SheqsyEmployee{})
}
