package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/SafetyCulture/mitti-exporter/cmd/mitti-exporter/cmd/configure"
	"github.com/SafetyCulture/mitti-exporter/cmd/mitti-exporter/cmd/export"
	util "github.com/SafetyCulture/mitti-exporter/cmd/mitti-exporter/cmd/utils"
	"github.com/SafetyCulture/mitti-exporter/internal/app/version"
	"github.com/SafetyCulture/mitti-exporter/pkg/update"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"

	flag "github.com/spf13/pflag"
)

var cfgFile string
var connectionFlags, dbFlags, sqliteFlags, csvFlags, exportFlags, mediaFlags, inspectionFlags, actionFlags,
	templatesFlag, tablesFlag, schemasFlag, reportFlags, sitesFlags *flag.FlagSet

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Version: version.GetVersion(),
	Use:     "mitti-exporter",
	Short:   "A CLI tool for extracting your Mitti data",
	Long:    "A CLI tool for extracting your Mitti data",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fmt.Printf("Mitti Exporter CLI version %s\n", version.GetVersion())
	updateMsgChan := make(chan *update.ReleaseInfo)

	go func() {
		res := update.Check(version.GetVersion(), update.RepoExporter)
		updateMsgChan <- res
	}()

	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	if newRelease := <-updateMsgChan; newRelease != nil {
		yellow := color.FgYellow.Render
		cyan := color.FgCyan.Render
		fmt.Fprintf(os.Stderr, "\n\n%s %s → %s\n%s\n\n",
			yellow("A new version of mitti-exporter is available"),
			cyan(version.GetVersion()),
			cyan(newRelease.Version),
			yellow(newRelease.ChangelogURL),
		)
	}
}

func writeDocs(*cobra.Command, []string) error {
	return doc.GenMarkdownTree(RootCmd, "docs/")
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config-path", "./mitti-exporter.yaml", "config file")

	configFlags()
	bindFlags()

	// Add sub-commands
	addCmd(export.SQLCmd(), connectionFlags, exportFlags, dbFlags, inspectionFlags, actionFlags, templatesFlag, tablesFlag, schemasFlag, mediaFlags, sitesFlags)
	addCmd(export.CSVCmd(), connectionFlags, exportFlags, csvFlags, inspectionFlags, actionFlags, templatesFlag, tablesFlag, schemasFlag, mediaFlags, sitesFlags)
	addCmd(export.SQLiteCmd(), connectionFlags, exportFlags, sqliteFlags, inspectionFlags, actionFlags, templatesFlag, tablesFlag, schemasFlag, mediaFlags, sitesFlags)
	addCmd(export.InspectionJSONCmd(), exportFlags, connectionFlags, inspectionFlags, actionFlags, templatesFlag)
	addCmd(export.ReportCmd(), connectionFlags, exportFlags, inspectionFlags, actionFlags, templatesFlag, reportFlags)
	addCmd(export.PrintSchemaCmd())
	addCmd(configure.Cmd(), connectionFlags, dbFlags, exportFlags, inspectionFlags, actionFlags, templatesFlag, tablesFlag)
	RootCmd.AddCommand(&cobra.Command{
		Hidden: true,
		Use:    "docs",
		RunE:   writeDocs,
	})
}

func configFlags() {
	connectionFlags = flag.NewFlagSet("connection", flag.ContinueOnError)
	// TODO - Can we validate these tokens and throw error if they are wrong?
	connectionFlags.StringP("access-token", "t", "", "API Access Token")
	connectionFlags.String("api-url", "https://api.safetyculture.io", "API URL")
	connectionFlags.StringP("sheqsy-username", "", "", "SHEQSY API Username")
	connectionFlags.StringP("sheqsy-password", "", "", "SHEQSY API Password")
	connectionFlags.StringP("sheqsy-company-id", "", "", "SHEQSY Company ID")
	connectionFlags.String("sheqsy-api-url", "https://app.sheqsy.com", "API URL")
	connectionFlags.Bool("tls-skip-verify", false, "Skip verification of API TLS certificates")
	connectionFlags.String("tls-cert", "", "Custom root CA certificate to use when making API requests")
	connectionFlags.String("proxy-url", "", "Proxy URL for making API requests through")
	connectionFlags.Int("max-concurrency", 10, "Maximum number of concurrent API requests (defaults to max 10)")

	dbFlags = flag.NewFlagSet("db", flag.ContinueOnError)
	dbFlags.String("db-dialect", "mysql", "Database dialect. mysql, postgres and sqlserver are the only valid options.")
	dbFlags.String("db-connection-string", "", "Database connection string")
	dbFlags.Bool("db-auto-migrate-disabled", false, "Disable database auto migrations")

	sqliteFlags = flag.NewFlagSet("sqlite", flag.ContinueOnError)

	csvFlags = flag.NewFlagSet("csv", flag.ContinueOnError)
	csvFlags.Int("max-rows-per-file", 1000000, "Maximum number of rows in a csv file. New files will be created when reaching this limit.")

	exportFlags = flag.NewFlagSet("export", flag.ContinueOnError)
	exportFlags.String("export-path", "./export/", "File Export Path")
	exportFlags.Bool("incremental", true, "Update inspections, inspection_items and templates tables incrementally")
	exportFlags.String("modified-after", "", "Return inspections modified after this date (see readme for supported formats)")
	exportFlags.String("modified-before", "", "Return inspections modified before this date (see readme for supported formats)")
	exportFlags.String("block-size", "", "Split export into time blocks (e.g., \"1d\", \"1w\", \"1m\")")

	mediaFlags = flag.NewFlagSet("media", flag.ContinueOnError)
	mediaFlags.Bool("export-media", false, "Export media")
	mediaFlags.String("export-media-path", "./export/media/", "Media Export Path")

	inspectionFlags = flag.NewFlagSet("inspection", flag.ContinueOnError)
	inspectionFlags.StringSlice("inspection-skip-ids", []string{}, "Skip storing these inspection IDs")
	inspectionFlags.Bool("inspection-include-inactive-items", false, "Include inactive items in the inspection_items table (default false)")
	inspectionFlags.String("inspection-archived", "false", "Return archived inspections, false, true or both")
	inspectionFlags.String("inspection-completed", "true", "Return completed inspections, false, true or both")
	inspectionFlags.Int("inspection-limit", 100, "Number of inspections fetched at once. Lower this number if the exporter fails to load the data")
	inspectionFlags.String("inspection-web-report-link", "private", "Web report link format. Can be public or private")

	actionFlags = flag.NewFlagSet("action", flag.ContinueOnError)
	actionFlags.Int("action-limit", 100, "Number of actions fetched at once. Lower this number if the exporter fails to load the data")

	templatesFlag = flag.NewFlagSet("templates", flag.ContinueOnError)
	templatesFlag.StringSlice("template-ids", []string{}, "Template IDs to filter inspections and schedules by (default all)")

	tablesFlag = flag.NewFlagSet("tables", flag.ContinueOnError)
	tablesFlag.StringSlice("tables", []string{}, "ExportTables to export (default all)")

	schemasFlag = flag.NewFlagSet("schemas", flag.ContinueOnError)
	schemasFlag.Bool("create-schema-only", false, "Create schema only (default false)")

	reportFlags = flag.NewFlagSet("report", flag.ContinueOnError)
	reportFlags.StringSlice("format", []string{"PDF"}, "Export format (PDF,WORD)")
	reportFlags.String("filename-convention", "INSPECTION_TITLE", "The name of the report exported, either INSPECTION_TITLE or INSPECTION_ID")
	reportFlags.String("preference-id", "", "The report layout to apply to the document")
	reportFlags.Int("retry-timeout", 15, "Specify the time in seconds spent retrieving each report. Values greater than 60 seconds will be treated as 60 seconds.")

	sitesFlags = flag.NewFlagSet("sites", flag.ContinueOnError)
	sitesFlags.Bool("site-include-deleted", false, "Include deleted sites in the sites table (default false)")
	sitesFlags.Bool("site-include-full-hierarchy", true, "Include full sites hierarchy in table e.g. areas, regions, etc (default true)")
}

func bindFlags() {
	util.Check(viper.BindPFlag("access_token", connectionFlags.Lookup("access-token")), "while binding flag")
	util.Check(viper.BindPFlag("sheqsy_username", connectionFlags.Lookup("sheqsy-username")), "while binding flag")
	util.Check(viper.BindPFlag("sheqsy_password", connectionFlags.Lookup("sheqsy-password")), "while binding flag")
	util.Check(viper.BindPFlag("sheqsy_company_id", connectionFlags.Lookup("sheqsy-company-id")), "while binding flag")

	util.Check(viper.BindPFlag("api.url", connectionFlags.Lookup("api-url")), "while binding flag")
	util.Check(viper.BindPFlag("api.sheqsy_url", connectionFlags.Lookup("sheqsy-api-url")), "while binding flag")
	util.Check(viper.BindPFlag("api.tls_skip_verify", connectionFlags.Lookup("tls-skip-verify")), "while binding flag")
	util.Check(viper.BindPFlag("api.tls_cert", connectionFlags.Lookup("tls-cert")), "while binding flag")
	util.Check(viper.BindPFlag("api.proxy_url", connectionFlags.Lookup("proxy-url")), "while binding flag")
	util.Check(viper.BindPFlag("api.max_concurrency", connectionFlags.Lookup("max-concurrency")), "while binding flag")

	util.Check(viper.BindPFlag("db.dialect", dbFlags.Lookup("db-dialect")), "while binding flag")
	util.Check(viper.BindPFlag("db.connection_string", dbFlags.Lookup("db-connection-string")), "while binding flag")
	util.Check(viper.BindPFlag("db.auto_migrate_disabled", dbFlags.Lookup("db-auto-migrate-disabled")), "while binding flag")

	util.Check(viper.BindPFlag("csv.max_rows_per_file", csvFlags.Lookup("max-rows-per-file")), "while binding flag")

	util.Check(viper.BindPFlag("export.path", exportFlags.Lookup("export-path")), "while binding flag")
	util.Check(viper.BindPFlag("export.incremental", exportFlags.Lookup("incremental")), "while binding flag")
	util.Check(viper.BindPFlag("export.modified_after", exportFlags.Lookup("modified-after")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.modified_before", exportFlags.Lookup("modified-before")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.block_size", exportFlags.Lookup("block-size")), "while binding flag")

	util.Check(viper.BindPFlag("export.media", mediaFlags.Lookup("export-media")), "while binding flag")
	util.Check(viper.BindPFlag("export.media_path", mediaFlags.Lookup("export-media-path")), "while binding flag")
	util.Check(viper.BindPFlag("export.template_ids", templatesFlag.Lookup("template-ids")), "while binding flag")
	util.Check(viper.BindPFlag("export.tables", tablesFlag.Lookup("tables")), "while binding flag")

	util.Check(viper.BindPFlag("export.inspection.included_inactive_items", inspectionFlags.Lookup("inspection-include-inactive-items")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.archived", inspectionFlags.Lookup("inspection-archived")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.completed", inspectionFlags.Lookup("inspection-completed")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.skip_ids", inspectionFlags.Lookup("inspection-skip-ids")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.limit", inspectionFlags.Lookup("inspection-limit")), "while binding flag")
	util.Check(viper.BindPFlag("export.inspection.web_report_link", inspectionFlags.Lookup("inspection-web-report-link")), "while binding flag")

	util.Check(viper.BindPFlag("export.action.limit", actionFlags.Lookup("action-limit")), "while binding flag")

	util.Check(viper.BindPFlag("export.site.include_deleted", sitesFlags.Lookup("site-include-deleted")), "while binding flag")
	util.Check(viper.BindPFlag("export.site.include_full_hierarchy", sitesFlags.Lookup("site-include-full-hierarchy")), "while binding flag")

	util.Check(viper.BindPFlag("report.format", reportFlags.Lookup("format")), "while binding flag")
	util.Check(viper.BindPFlag("report.filename_convention", reportFlags.Lookup("filename-convention")), "while binding flag")
	util.Check(viper.BindPFlag("report.preference_id", reportFlags.Lookup("preference-id")), "while binding flag")
	util.Check(viper.BindPFlag("report.retry_timeout", reportFlags.Lookup("retry-timeout")), "while binding flag")
}

func addCmd(cmd *cobra.Command, flags ...*flag.FlagSet) {
	for _, f := range flags {
		cmd.PersistentFlags().AddFlagSet(f)
	}

	RootCmd.AddCommand(cmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigFile(resolveConfigFile(cfgFile))

	viper.SetEnvPrefix("IAUD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	_ = viper.ReadInConfig()
}

// resolveConfigFile falls back to the pre-rebrand "safetyculture-exporter.yaml"
// name if the default config path doesn't exist but the legacy one does
// alongside it, so existing configs keep working without a manual rename.
// An explicit, custom --config-path is always used as-is.
func resolveConfigFile(cfgFile string) string {
	if cfgFile == "" {
		cfgFile = "mitti-exporter.yaml"
	}

	if _, err := os.Stat(cfgFile); err == nil {
		return cfgFile
	}

	if legacy := strings.Replace(cfgFile, "mitti-exporter.yaml", "safetyculture-exporter.yaml", 1); legacy != cfgFile {
		if _, err := os.Stat(legacy); err == nil {
			return legacy
		}
	}

	return cfgFile
}
