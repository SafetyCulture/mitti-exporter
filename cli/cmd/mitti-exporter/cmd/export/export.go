package export

import (
	util "github.com/SafetyCulture/mitti-exporter/cmd/mitti-exporter/cmd/utils"
	"github.com/SafetyCulture/mitti-exporter/internal/app/version"
	exporterAPI "github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SQLCmd is used to export data in sql format
func SQLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sql",
		Short: "Export Mitti data to SQL database",
		Example: `// Limit inspections and schedules to these templates
mitti-exporter sql --template-ids template_F492E54D87F2419E9398F7BDCA0FA5D9,template_d54e06808d2f11e2893e83a731dba0ca

// Customise export location
mitti-exporter sql --export-path /path/to/export/to`,
		RunE: runSQL,
	}
}

// CSVCmd is used to export data in csv format
func CSVCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "csv",
		Short: "Export Mitti data to CSV files",
		Example: `// Limit inspections and schedules to these templates
mitti-exporter csv --template-ids template_F492E54D87F2419E9398F7BDCA0FA5D9,template_d54e06808d2f11e2893e83a731dba0ca

// Customise export location
mitti-exporter csv --export-path /path/to/export/to`,
		RunE: runCSV,
	}
}

// SQLiteCmd is used to export data into SQLite db
func SQLiteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sqlite",
		Short: "Export Mitti data to a local SQLite file",
		Example: `// Limit inspections and schedules to these templates
mitti-exporter sqlite --template-ids template_F492E54D87F2419E9398F7BDCA0FA5D9,template_d54e06808d2f11e2893e83a731dba0ca

// Customise export location
mitti-exporter sqlite --export-path /path/to/export/to`,
		RunE: runSQLite,
	}
}

// InspectionJSONCmd is used to export inspections to json files
func InspectionJSONCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspection-json",
		Short: "Export Mitti inspections to json files",
		Example: `// Limit inspections to these templates
mitti-exporter inspection-json --template-ids template_F492E54D87F2419E9398F7BDCA0FA5D9,template_d54e06808d2f11e2893e83a731dba0ca

// Customise export location
mitti-exporter inspection-json --export-path /path/to/export/to`,
		RunE: runInspectionJSON,
	}
}

// PrintSchemaCmd is used to print table schemas
func PrintSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "schema",
		Short:   "Print Mitti table schemas",
		Example: `mitti-exporter schema`,
		RunE:    printSchema,
	}
}

// ReportCmd is used to download inspection reports
func ReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Export inspection report",
		Example: `// Export PDF and Word inspection reports
		mitti-exporter report --export-path /path/to/export/to --format PDF,WORD
		// Export PDF inspection reports with a custom report layout
		mitti-exporter report --export-path /path/to/export/to --format PDF --preference-id abc`,
		RunE: runInspectionReports,
	}
}

func runSQL(*cobra.Command, []string) error {
	exp := NewMittiExporter(viper.GetViper())
	err := exp.RunSQL()
	util.Check(err, "error while exporting SQL")
	return nil
}

func runInspectionJSON(*cobra.Command, []string) error {
	exp := NewMittiExporter(viper.GetViper())
	err := exp.RunInspectionJSON()
	util.Check(err, "error while exporting JSON")
	return nil
}

func runCSV(*cobra.Command, []string) error {
	exp := NewMittiExporter(viper.GetViper())
	err := exp.RunCSV()
	util.Check(err, "error while exporting CSV")
	return nil
}

func runSQLite(*cobra.Command, []string) error {
	exp := NewMittiExporter(viper.GetViper())
	err := exp.RunSQLite()
	util.Check(err, "error while exporting SQLITE")
	return nil
}

func printSchema(*cobra.Command, []string) error {
	exp := NewMittiExporter(viper.GetViper())
	err := exp.RunPrintSchema()
	util.Check(err, "error while printing schema")
	return nil
}

func runInspectionReports(*cobra.Command, []string) error {
	exp := NewMittiExporter(viper.GetViper())
	err := exp.RunInspectionReports()
	util.Check(err, "failed to generate reports")
	return nil
}

// NewMittiExporter create a new MittiExporter with configuration from Viper
func NewMittiExporter(v *viper.Viper) *exporterAPI.MittiExporter {
	cm, err := exporterAPI.NewConfigurationManagerFromFile("", v.ConfigFileUsed())
	if err != nil {
		util.Check(err, "failed to initialize the exporter")
	}
	MapViperConfigToExporterConfiguration(v, cm.Configuration)
	cm.ApplySafetyGuards()
	util.Check(err, "while loading config file")

	ver := exporterAPI.AppVersion{
		IntegrationID:      version.GetIntegrationID(),
		IntegrationVersion: version.GetVersion(),
	}
	exporter, err := exporterAPI.NewMittiExporter(cm.Configuration, &ver)
	util.Check(err, "failed to initialize the exporter")
	return exporter
}

// MapViperConfigToExporterConfiguration maps Viper config to ExporterConfiguration structure
func MapViperConfigToExporterConfiguration(v *viper.Viper, cfg *exporterAPI.ExporterConfiguration) {
	cfg.AccessToken = v.GetString("access_token")
	cfg.API.MaxConcurrency = v.GetInt("api.max_concurrency")
	cfg.SheqsyUsername = v.GetString("sheqsy_username")
	cfg.SheqsyCompanyID = v.GetString("sheqsy_company_id")
	cfg.Db.Dialect = v.GetString("db.dialect")
	cfg.Db.ConnectionString = v.GetString("db.connection_string")
	cfg.Db.AutoMigrateDisabled = v.GetBool("db.auto_migrate_disabled")
	cfg.Csv.MaxRowsPerFile = v.GetInt("csv.max_rows_per_file")
	cfg.Export.Path = v.GetString("export.path")
	cfg.Export.Incremental = v.GetBool("export.incremental")
	cfg.Export.ModifiedAfter.Time = v.GetTime("export.modified_after")
	cfg.Export.TemplateIds = v.GetStringSlice("export.template_ids")
	cfg.Export.Tables = v.GetStringSlice("export.tables")
	cfg.Export.SchemaOnly = v.GetBool("export.schema_only")
	cfg.Export.Inspection.IncludedInactiveItems = v.GetBool("export.inspection.included_inactive_items")
	cfg.Export.Inspection.Archived = v.GetString("export.inspection.archived")
	cfg.Export.Inspection.Completed = v.GetString("export.inspection.completed")
	cfg.Export.Inspection.SkipIds = v.GetStringSlice("export.inspection.skip_ids")
	cfg.Export.Inspection.Limit = v.GetInt("export.inspection.limit")
	cfg.Export.Inspection.WebReportLink = v.GetString("export.inspection.web_report_link")
	cfg.Export.Site.IncludeDeleted = v.GetBool("export.site.include_deleted")
	cfg.Export.Site.IncludeFullHierarchy = v.GetBool("export.site.include_full_hierarchy")
	cfg.Export.Media = v.GetBool("export.media")
	cfg.Export.MediaPath = v.GetString("export.media_path")
	cfg.Export.Action.Limit = v.GetInt("export.action.limit")
	cfg.Export.Issue.Limit = v.GetInt("export.issue.limit")
	cfg.Report.Format = v.GetStringSlice("report.format")
	cfg.Report.PreferenceID = v.GetString("report.preference_id")
	cfg.Report.FilenameConvention = v.GetString("report.filename_convention")
	cfg.Report.RetryTimeout = v.GetInt("report.retry_timeout")
}
