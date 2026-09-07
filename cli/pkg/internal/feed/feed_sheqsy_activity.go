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

// SheqsyActivity represents an activity in sheqsy
type SheqsyActivity struct {
	ActivityUID             string    `json:"activityUId" csv:"activity_uid" gorm:"primarykey;column:activity_uid;size:36"`
	ActivityID              int       `json:"activityId" csv:"activity_id" gorm:"column:activity_id"`
	ExternalID              *string   `json:"externalId" csv:"external_id" gorm:"column:external_id"`
	Email                   string    `json:"email" csv:"email" gorm:"column:email"`
	PhoneNumber             string    `json:"phoneNumber" csv:"phone_number" gorm:"column:phone_number"`
	ActivityName            string    `json:"activityName" csv:"activity_name" gorm:"column:activity_name"`
	StartDateTimeUTC        time.Time `json:"startDateTimeUTC" csv:"start_date_time_utc" gorm:"column:start_date_time_utc"`
	FinishDateTimeUTC       time.Time `json:"finishDateTimeUTC" csv:"finish_date_time_utc" gorm:"column:finish_date_time_utc"`
	ActivityType            string    `json:"activityType" csv:"activity_type" gorm:"column:activity_type"`
	EmployeeName            string    `json:"employeeName" csv:"employee_name" gorm:"column:employee_name"`
	EmployeeSurname         string    `json:"employeeSurname" csv:"employee_surname" gorm:"column:employee_surname"`
	StartLatitude           float64   `json:"startLatitude" csv:"start_latitude" gorm:"column:start_latitude"`
	StartLongitude          float64   `json:"startLongitude" csv:"start_longitude" gorm:"column:start_longitude"`
	StartAddress            string    `json:"startAddress" csv:"start_address" gorm:"column:start_address"`
	FinishLatitude          float64   `json:"finishLatitude" csv:"finish_latitude" gorm:"column:finish_latitude"`
	FinishLongitude         float64   `json:"finishLongitude" csv:"finish_longitude" gorm:"column:finish_longitude"`
	FinishAddress           string    `json:"finishAddress" csv:"finish_address" gorm:"column:finish_address"`
	TimeSpentSec            int       `json:"timeSpentSec" csv:"time_spent_sec" gorm:"column:time_spent_sec"`
	Version                 int       `json:"version" csv:"version" gorm:"column:version"`
	TimeEnrouteSec          int       `json:"timeEnrouteSec" csv:"time_enroute_sec" gorm:"column:time_enroute_sec"`
	DistanceTravelledMeters int       `json:"distanceTravelledMeters" csv:"distance_travelled_meters" gorm:"column:distance_travelled_meters"`
	ShiftID                 *int      `json:"shiftId" csv:"shift_id" gorm:"column:shift_id"`
	Departments             string    `json:"departments" csv:"departments" gorm:"type:string;column:departments"`
	ExportedAt              time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// SheqsyActivityFeed is a representation of the activities feed
type SheqsyActivityFeed struct{}

// Name is the name of the feed
func (f *SheqsyActivityFeed) Name() string {
	return "sheqsy_activities"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *SheqsyActivityFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *SheqsyActivityFeed) Model() interface{} {
	return SheqsyActivity{}
}

// RowsModel returns the model of feed rows
func (f *SheqsyActivityFeed) RowsModel() interface{} {
	return &[]*SheqsyActivity{}
}

// PrimaryKey returns the primary key(s)
func (f *SheqsyActivityFeed) PrimaryKey() []string {
	return []string{"activity_uid"}
}

// Columns returns the columns of the row
func (f *SheqsyActivityFeed) Columns() []string {
	return []string{
		"activity_id",
		"external_id",
		"email",
		"phone_number",
		"activity_name",
		"start_date_time_utc",
		"finish_date_time_utc",
		"activity_type",
		"employee_name",
		"employee_surname",
		"start_latitude",
		"start_longitude",
		"start_address",
		"finish_latitude",
		"finish_longitude",
		"finish_address",
		"time_spent_sec",
		"version",
		"time_enroute_sec",
		"distance_travelled_meters",
		"shift_id",
		"departments",
	}
}

// Order returns the ordering when retrieving an export
func (f *SheqsyActivityFeed) Order() string {
	return "activity_uid"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *SheqsyActivityFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*SheqsyActivityFeed{})
}

// Export exports the feed to the supplied exporter
func (f *SheqsyActivityFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, companyID string) error {
	log := logger.GetLogger().With("feed", f.Name(), "org_id", companyID)

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: !exporter.SupportsUpsert(),
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	type apiResp struct {
		Data         []*SheqsyActivity `json:"data"`
		LastVersion  int               `json:"lastVersion"`
		HasMoreItems bool              `json:"hasMoreItems"`
		ItemsLeft    int               `json:"itemsLeft"`
	}

	data := apiResp{}

	version := 1
	for version != 0 {
		resp, err := httpapi.ExecuteGet[json.RawMessage](ctx, apiClient, fmt.Sprintf("/SheqsyIntegrationApi/api/v3/companies/%s/activities/history?ver=%d", companyID, version), nil)
		if err != nil {
			return fmt.Errorf("fetch data: %w", err)
		}

		var respBytes []byte
		respBytes = *resp

		gjson.GetBytes(respBytes, "data").ForEach(func(key, value gjson.Result) bool {
			// These timestamps aren't parsable as RFC 3339 strings, so we have to munge them to that format
			respBytes, err = sjson.SetBytes(
				respBytes,
				fmt.Sprintf("data.%d.startDateTimeUTC", key.Int()),
				value.Get("startDateTimeUTC").String()+"Z",
			)
			if err != nil {
				log.Errorf("fix timestamp: %v", err)
				return false
			}

			respBytes, err = sjson.SetBytes(
				respBytes,
				fmt.Sprintf("data.%d.finishDateTimeUTC", key.Int()),
				value.Get("finishDateTimeUTC").String()+"Z",
			)
			if err != nil {
				log.Errorf("fix timestamp: %v", err)
				return false
			}

			// Departments needs to be a flat string
			var departments []string
			value.Get("departments").ForEach(func(key, value gjson.Result) bool {
				departments = append(departments, value.String())
				return true
			})

			respBytes, err = sjson.SetBytes(
				respBytes,
				fmt.Sprintf("data.%d.departments", key.Int()),
				strings.Join(departments, ","),
			)
			if err != nil {
				log.Errorf("join departments: %v", err)
				return false
			}
			return true
		})

		if err := json.Unmarshal(respBytes, &data); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(data.Data) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, data.Data, func(batch []*SheqsyActivity) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		version = data.LastVersion
		if !data.HasMoreItems {
			version = 0
		}

		log.With(
			"estimated_remaining", 0,
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
	}

	return exporter.FinaliseExport(f, &[]*SheqsyActivity{})
}
