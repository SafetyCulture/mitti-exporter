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

// SheqsyDepartment represents a department in sheqsy
type SheqsyDepartment struct {
	DepartmentUID string    `json:"departmentUId" csv:"department_uid" gorm:"primarykey;column:department_uid;size:36"`
	DepartmentID  int       `json:"departmentId" csv:"department_id" gorm:"column:department_id"`
	Name          string    `json:"name" csv:"name" gorm:"column:name"`
	ExternalName  string    `json:"external_name" csv:"external_name" gorm:"column:external_name"`
	Number        string    `json:"number" csv:"number" gorm:"column:number"`
	Manager       string    `json:"manager" csv:"manager" gorm:"column:manager"`
	ExportedAt    time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// SheqsyDepartmentFeed is a representation of the departments feed
type SheqsyDepartmentFeed struct{}

// Name is the name of the feed
func (f *SheqsyDepartmentFeed) Name() string {
	return "sheqsy_departments"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *SheqsyDepartmentFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *SheqsyDepartmentFeed) Model() interface{} {
	return SheqsyDepartment{}
}

// RowsModel returns the model of feed rows
func (f *SheqsyDepartmentFeed) RowsModel() interface{} {
	return &[]*SheqsyDepartment{}
}

// PrimaryKey returns the primary key(s)
func (f *SheqsyDepartmentFeed) PrimaryKey() []string {
	return []string{"department_uid"}
}

// Columns returns the columns of the row
func (f *SheqsyDepartmentFeed) Columns() []string {
	return []string{
		"department_id",
		"name",
		"external_name",
		"number",
		"manager",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *SheqsyDepartmentFeed) Order() string {
	return "department_uid"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *SheqsyDepartmentFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*SheqsyDepartmentFeed{})
}

// Export exports the feed to the supplied exporter
func (f *SheqsyDepartmentFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, companyID string) error {
	log := logger.GetLogger().With("feed", f.Name(), "org_id", companyID)

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: !exporter.SupportsUpsert(),
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	var rows []*SheqsyDepartment

	resp, err := httpapi.ExecuteGet[json.RawMessage](ctx, apiClient, fmt.Sprintf("/SheqsyIntegrationApi/api/v3/companies/%s/departments", companyID), nil)
	if err != nil {
		return fmt.Errorf("fetch data: %w", err)
	}

	if err := json.Unmarshal(*resp, &rows); err != nil {
		return fmt.Errorf("map users data: %w", err)
	}

	if len(rows) != 0 {
		// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
		batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
		err := util.SplitSliceInBatch(batchSize, rows, func(batch []*SheqsyDepartment) error {
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

	return exporter.FinaliseExport(f, &[]*SheqsyDepartment{})
}
