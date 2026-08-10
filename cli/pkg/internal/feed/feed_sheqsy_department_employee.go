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

// SheqsyDepartmentEmployee represents a user in sheqsy
type SheqsyDepartmentEmployee struct {
	EmployeeUID   string    `json:"employeeUId" csv:"employee_uid" gorm:"primaryKey;column:employee_uid;size:36"`
	DepartmentUID string    `json:"departmentUId" csv:"department_uid" gorm:"primaryKey;column:department_uid;size:36"`
	EmployeeID    int       `json:"employeeId" csv:"employee_id" gorm:"column:employee_id"`
	DepartmentID  int       `json:"departmentId" csv:"department_id" gorm:"column:department_id"`
	ExportedAt    time.Time `json:"exported_at" csv:"exported_at" gorm:"autoUpdateTime"`
}

// SheqsyDepartmentEmployeeFeed is a representation of the users feed
type SheqsyDepartmentEmployeeFeed struct{}

// Name is the name of the feed
func (f *SheqsyDepartmentEmployeeFeed) Name() string {
	return "sheqsy_department_employees"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *SheqsyDepartmentEmployeeFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *SheqsyDepartmentEmployeeFeed) Model() interface{} {
	return SheqsyDepartmentEmployee{}
}

// RowsModel returns the model of feed rows
func (f *SheqsyDepartmentEmployeeFeed) RowsModel() interface{} {
	return &[]*SheqsyDepartmentEmployee{}
}

// PrimaryKey returns the primary key(s)
func (f *SheqsyDepartmentEmployeeFeed) PrimaryKey() []string {
	return []string{"employee_uid", "department_uid"}
}

// Columns returns the columns of the row
func (f *SheqsyDepartmentEmployeeFeed) Columns() []string {
	return []string{
		"employee_id",
		"department_id",
		"exported_at",
	}
}

// Order returns the ordering when retrieving an export
func (f *SheqsyDepartmentEmployeeFeed) Order() string {
	return "employee_uid DESC, department_uid DESC"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *SheqsyDepartmentEmployeeFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*SheqsyDepartmentEmployee{})
}

type sheqsyEmployeeRaw struct {
	EmployeeUID string `json:"employeeUId"`
	EmployeeID  int    `json:"employeeId"`
	Departments []struct {
		DepartmentUID string `json:"departmentUId"`
		DepartmentID  int    `json:"departmentId"`
	}
}

// Export exports the feed to the supplied exporter
func (f *SheqsyDepartmentEmployeeFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, companyID string) error {
	log := logger.GetLogger().With("feed", f.Name(), "org_id", companyID)

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Truncate files if upserts aren't supported.
		// This ensures that the export does not contain duplicate rows
		Truncate: !exporter.SupportsUpsert(),
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	var rows []*SheqsyDepartmentEmployee

	resp, err := httpapi.ExecuteGet[json.RawMessage](ctx, apiClient, fmt.Sprintf("/SheqsyIntegrationApi/api/v3/companies/%s/employees", companyID), nil)
	if err != nil {
		return fmt.Errorf("fetch data: %w", err)
	}

	var rawData []sheqsyEmployeeRaw
	if err := json.Unmarshal(*resp, &rawData); err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
	}

	for _, row := range rawData {
		for _, department := range row.Departments {
			rows = append(rows, &SheqsyDepartmentEmployee{
				EmployeeUID:   row.EmployeeUID,
				EmployeeID:    row.EmployeeID,
				DepartmentUID: department.DepartmentUID,
				DepartmentID:  department.DepartmentID,
			})
		}
	}

	if len(rows) != 0 {
		// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
		batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
		err := util.SplitSliceInBatch(batchSize, rows, func(batch []*SheqsyDepartmentEmployee) error {
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

	return exporter.FinaliseExport(f, &[]*SheqsyDepartmentEmployee{})
}
