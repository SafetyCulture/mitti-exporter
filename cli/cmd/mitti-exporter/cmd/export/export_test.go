package export_test

import (
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/cmd/mitti-exporter/cmd/export"
	exporterAPI "github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintSchemaCmd(t *testing.T) {
	res := export.PrintSchemaCmd()
	require.NotNil(t, res)
	assert.EqualValues(t, "schema", res.Use)
	assert.EqualValues(t, "Print Mitti table schemas", res.Short)
	assert.EqualValues(t, "mitti-exporter schema", res.Example)
}

func TestReportCmd(t *testing.T) {
	res := export.ReportCmd()
	require.NotNil(t, res)
	assert.EqualValues(t, "report", res.Use)
	assert.EqualValues(t, "Export inspection report", res.Short)
}

func TestInspectionJSONCmd(t *testing.T) {
	res := export.InspectionJSONCmd()
	require.NotNil(t, res)
	assert.EqualValues(t, "inspection-json", res.Use)
	assert.EqualValues(t, "Export Mitti inspections to json files", res.Short)
}

func TestCSVCmd(t *testing.T) {
	res := export.CSVCmd()
	require.NotNil(t, res)
	assert.EqualValues(t, "csv", res.Use)
	assert.EqualValues(t, "Export Mitti data to CSV files", res.Short)
}

func TestSQLCmd(t *testing.T) {
	res := export.SQLCmd()
	require.NotNil(t, res)
	assert.EqualValues(t, "sql", res.Use)
	assert.EqualValues(t, "Export Mitti data to SQL database", res.Short)
}

func TestNewConfigurationManagerFromFile_should_apply_the_viper_defaults(t *testing.T) {
	cm := exporterAPI.NewConfigurationManager("", "new.yaml")
	require.NotNil(t, cm)

	viperConfig := viper.New()
	viperConfig.Set("access_token", "sc-api-123")
	viperConfig.Set("db.dialect", "mysql")
	viperConfig.Set("export.path", "./export/")
	viperConfig.Set("export.media_path", "./export/media/")
	viperConfig.Set("export.action.limit", 100)
	viperConfig.Set("export.issue.limit", 100)
	viperConfig.Set("export.incremental", true)
	viperConfig.Set("export.modified_after", "2022-01-20")
	viperConfig.Set("export.template_ids", "A B C")
	viperConfig.Set("export.tables", "A B C")
	viperConfig.Set("export.inspection.included_inactive_items", true)
	viperConfig.Set("export.inspection.archived", "both")
	viperConfig.Set("export.inspection.completed", "true")
	viperConfig.Set("export.inspection.skip_ids", "A B C")
	viperConfig.Set("export.inspection.limit", 10)
	viperConfig.Set("export.inspection.web_report_link", "web_link")
	viperConfig.Set("export.site.include_deleted", true)
	viperConfig.Set("export.site.include_full_hierarchy", true)
	viperConfig.Set("export.media", true)
	viperConfig.Set("csv.max_rows_per_file", "1000000")
	viperConfig.Set("report.retry_timeout", "15")
	viperConfig.Set("report.format", "PDF")
	viperConfig.Set("report.filename_convention", "INSPECTION_TITLE")

	export.MapViperConfigToExporterConfiguration(viperConfig, cm.Configuration)

	// GENERIC FIELDS
	assert.EqualValues(t, "sc-api-123", cm.Configuration.AccessToken)
	assert.True(t, cm.Configuration.Export.Incremental)
	modifiedAfter, err := time.Parse("2006-01-02", "2022-01-20")
	assert.Nil(t, err)
	assert.Equal(t, modifiedAfter, cm.Configuration.Export.ModifiedAfter.Time)
	assert.EqualValues(t, []string{"A", "B", "C"}, cm.Configuration.Export.TemplateIds)
	assert.EqualValues(t, []string{"A", "B", "C"}, cm.Configuration.Export.Tables)

	// INSPECTION FIELDS
	assert.True(t, cm.Configuration.Export.Inspection.IncludedInactiveItems)
	assert.EqualValues(t, "both", cm.Configuration.Export.Inspection.Archived)
	assert.EqualValues(t, "true", cm.Configuration.Export.Inspection.Completed)
	assert.EqualValues(t, []string{"A", "B", "C"}, cm.Configuration.Export.Inspection.SkipIds)
	assert.EqualValues(t, 10, cm.Configuration.Export.Inspection.Limit)
	assert.EqualValues(t, "web_link", cm.Configuration.Export.Inspection.WebReportLink)

	// SITE CONFIG FIELDS
	assert.True(t, cm.Configuration.Export.Site.IncludeDeleted)
	assert.True(t, cm.Configuration.Export.Site.IncludeFullHierarchy)

	// MEDIA CONFIG FIELDS
	assert.True(t, cm.Configuration.Export.Media)

	// ACTION CONFIG FIELDS
	assert.EqualValues(t, 100, cm.Configuration.Export.Action.Limit)

	// ISSUE CONFIG FIELDS
	assert.EqualValues(t, 100, cm.Configuration.Export.Issue.Limit)

	assert.Equal(t, "https://api.safetyculture.io", cm.Configuration.API.URL)
	assert.Equal(t, "https://app.sheqsy.com", cm.Configuration.API.SheqsyURL)
	assert.Equal(t, 1000000, cm.Configuration.Csv.MaxRowsPerFile)
	assert.Equal(t, "mysql", cm.Configuration.Db.Dialect)
	assert.Equal(t, 100, cm.Configuration.Export.Action.Limit)
	assert.True(t, cm.Configuration.Export.Incremental)
	assert.Equal(t, "both", cm.Configuration.Export.Inspection.Archived)
	assert.Equal(t, "true", cm.Configuration.Export.Inspection.Completed)
	assert.Equal(t, 10, cm.Configuration.Export.Inspection.Limit)
	assert.Equal(t, "web_link", cm.Configuration.Export.Inspection.WebReportLink)
	assert.Equal(t, "./export/media/", cm.Configuration.Export.MediaPath)
	assert.Equal(t, "./export/", cm.Configuration.Export.Path)
	assert.Equal(t, "INSPECTION_TITLE", cm.Configuration.Report.FilenameConvention)
	assert.Equal(t, []string{"PDF"}, cm.Configuration.Report.Format)
	assert.Equal(t, 15, cm.Configuration.Report.RetryTimeout)
}
