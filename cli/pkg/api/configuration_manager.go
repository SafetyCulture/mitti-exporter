package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/inspections"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
)

// ExporterConfiguration is the equivalent struct of YAML
type ExporterConfiguration struct {
	AccessToken string `yaml:"access_token"`
	API         struct {
		ProxyURL       string `yaml:"proxy_url"`
		SheqsyURL      string `yaml:"sheqsy_url"`
		TLSCert        string `yaml:"tls_cert"`
		TLSSkipVerify  bool   `yaml:"tls_skip_verify"`
		URL            string `yaml:"url"`
		MaxConcurrency int    `yaml:"max_concurrency"`
	} `yaml:"api"`
	Csv struct {
		MaxRowsPerFile int `yaml:"max_rows_per_file"`
	} `yaml:"csv"`
	Db struct {
		ConnectionString    string `yaml:"connection_string"`
		Dialect             string `yaml:"dialect"`
		AutoMigrateDisabled bool   `yaml:"auto_migrate_disabled"`
	} `yaml:"db"`
	Export struct {
		Action struct {
			Limit int `yaml:"limit"`
		} `yaml:"action"`
		Asset struct {
			Limit int `yaml:"limit"`
		} `yaml:"asset"`
		Course struct {
			Progress struct {
				Limit            int    `yaml:"limit"`
				CompletionStatus string `yaml:"completion_status"`
			} `yaml:"progress"`
		} `yaml:"course"`
		Incremental bool `yaml:"incremental"`
		Inspection  struct {
			Archived              string   `yaml:"archived"`
			Completed             string   `yaml:"completed"`
			IncludedInactiveItems bool     `yaml:"included_inactive_items"`
			Limit                 int      `yaml:"limit"`
			SkipIds               []string `yaml:"skip_ids"`
			WebReportLink         string   `yaml:"web_report_link"`
			ModifiedBefore        mTime    `yaml:"modified_before"`
			BlockSize             string   `yaml:"block_size"`
		} `yaml:"inspection"`
		InspectionItems struct {
			SkipFields []string `yaml:"skip_fields"`
		} `yaml:"inspection_items"`
		Issue struct {
			Limit int `yaml:"limit"`
		} `yaml:"issue"`
		Media         bool   `yaml:"media"`
		MediaPath     string `yaml:"media_path"`
		ModifiedAfter mTime  `yaml:"modified_after"`
		TimeZone      string `yaml:"time_zone"`
		Path          string `yaml:"path"`
		SchemaOnly    bool   `yaml:"-"`
		Site          struct {
			IncludeDeleted       bool `yaml:"include_deleted"`
			IncludeFullHierarchy bool `yaml:"include_full_hierarchy"`
		} `yaml:"site"`
		Schedule struct {
			ResumeDownload bool `yaml:"resume_download"`
		} `yaml:"schedule"`
		Tables      []string `yaml:"tables"`
		TemplateIds []string `yaml:"template_ids"`
	} `yaml:"export"`
	Report struct {
		FilenameConvention string   `yaml:"filename_convention"`
		Format             []string `yaml:"format"`
		PreferenceID       string   `yaml:"preference_id"`
		RetryTimeout       int      `yaml:"retry_timeout"`
	} `yaml:"report"`
	SheqsyCompanyID string `yaml:"sheqsy_company_id"`
	SheqsyPassword  string `yaml:"sheqsy_password"`
	SheqsyUsername  string `yaml:"sheqsy_username"`
	Session         struct {
		ExportType string `yaml:"export_type"`
	} `yaml:"session"`
}

// AppVersion used to store the version and ID
type AppVersion struct {
	IntegrationID      string
	IntegrationVersion string
}

// mTime wrapper around time.Time in order to have a custom YAML marshaller/un-marshaller
type mTime struct {
	time.Time
}

// UnmarshalYAML custom un-marshaller for time.Time since empty strings throws an error
func (mt *mTime) UnmarshalYAML(value *yaml.Node) error {
	var timeString string
	err := value.Decode(&timeString)
	if err != nil {
		return err
	}
	timeString = strings.TrimSpace(timeString)

	var t time.Time
	switch timeString {
	case "":
		t = time.Time{}
	default:
		t, err = util.TimeFromString(timeString)
		if err != nil {
			return fmt.Errorf("failed to parse '%s' to time.Time: %v", timeString, err)
		}
	}

	mt.Time = t
	return nil
}

// MarshalYAML custom marshaller for time, when is ZERO, marshal as empty string
// note: doesn't work with pointer receiver
func (mt mTime) MarshalYAML() (interface{}, error) {
	if mt.IsZero() {
		return "", nil
	}

	return mt.Format(time.RFC3339), nil
}

// ConfigurationManager wrapper for configuration and fileName
type ConfigurationManager struct {
	path          string
	fileName      string
	Configuration *ExporterConfiguration
}

// loadConfiguration will load the specified YAML file if exists and map it
func (c *ConfigurationManager) loadConfiguration() error {
	if len(strings.TrimSpace(c.fileName)) == 0 || !strings.HasSuffix(c.fileName, ".yaml") {
		return fmt.Errorf("invalid file name provided")
	}

	yamlContents, err := os.ReadFile(filepath.Join(c.path, c.fileName))
	if err != nil {
		return fmt.Errorf("cannot read the configuration file: %w", err)
	}
	if err := yaml.Unmarshal(yamlContents, c.Configuration); err != nil {
		return fmt.Errorf("configuration file is corrupt: %w", err)
	}

	return nil
}

// ApplySafetyGuards will adjust certain values to acceptable maximum values
func (c *ConfigurationManager) ApplySafetyGuards() {
	defaultCfg := BuildConfigurationWithDefaults()

	// caps action batch limit to 100
	if c.Configuration.Export.Action.Limit > 100 || c.Configuration.Export.Action.Limit == 0 {
		c.Configuration.Export.Action.Limit = defaultCfg.Export.Action.Limit
	}

	// caps issue batch limit to 100
	if c.Configuration.Export.Issue.Limit > 100 || c.Configuration.Export.Issue.Limit == 0 {
		c.Configuration.Export.Issue.Limit = defaultCfg.Export.Issue.Limit
	}

	// caps course progress batch limit to 1000
	if c.Configuration.Export.Course.Progress.Limit > 1000 || c.Configuration.Export.Course.Progress.Limit == 0 {
		c.Configuration.Export.Course.Progress.Limit = defaultCfg.Export.Course.Progress.Limit
	}

	if c.Configuration.Export.Course.Progress.CompletionStatus == "" {
		c.Configuration.Export.Course.Progress.CompletionStatus = defaultCfg.Export.Course.Progress.CompletionStatus
	}

	if c.Configuration.Export.Inspection.Limit == 0 {
		c.Configuration.Export.Inspection.Limit = defaultCfg.Export.Inspection.Limit
	}

	if c.Configuration.Export.Asset.Limit == 0 {
		c.Configuration.Export.Asset.Limit = defaultCfg.Export.Asset.Limit
	}

	if c.Configuration.API.URL == "" {
		c.Configuration.API.URL = defaultCfg.API.URL
	}

	if c.Configuration.Csv.MaxRowsPerFile == 0 {
		c.Configuration.Csv.MaxRowsPerFile = defaultCfg.Csv.MaxRowsPerFile
	}

	if c.Configuration.Export.Tables == nil {
		c.Configuration.Export.Tables = defaultCfg.Export.Tables
	}

	if c.Configuration.Export.TemplateIds == nil {
		c.Configuration.Export.TemplateIds = defaultCfg.Export.TemplateIds
	}

	if c.Configuration.Export.Inspection.SkipIds == nil {
		c.Configuration.Export.Inspection.SkipIds = defaultCfg.Export.Inspection.SkipIds
	}

	if len(c.Configuration.Report.Format) == 0 {
		c.Configuration.Report.Format = defaultCfg.Report.Format
	}

	if c.Configuration.Report.FilenameConvention == "" {
		c.Configuration.Report.FilenameConvention = defaultCfg.Report.FilenameConvention
	}

	if c.Configuration.Export.Inspection.Completed == "" {
		c.Configuration.Export.Inspection.Completed = defaultCfg.Export.Inspection.Completed
	}

	if c.Configuration.Export.Inspection.Archived == "" {
		c.Configuration.Export.Inspection.Archived = defaultCfg.Export.Inspection.Archived
	}

	if c.Configuration.Export.TimeZone == "" {
		c.Configuration.Export.TimeZone = defaultCfg.Export.TimeZone
	}

	if c.Configuration.Session.ExportType == "" {
		c.Configuration.Session.ExportType = defaultCfg.Session.ExportType
	}

	if c.Configuration.Db.Dialect == "" {
		c.Configuration.Db.Dialect = defaultCfg.Db.Dialect
	}

	if c.Configuration.Export.Path == "" {
		c.Configuration.Export.Path = defaultCfg.Export.Path
	}

	if c.Configuration.Export.MediaPath == "" {
		c.Configuration.Export.MediaPath = defaultCfg.Export.MediaPath
	}

	if c.Configuration.Export.Inspection.BlockSize != "" {
		if _, err := util.ParseDuration(c.Configuration.Export.Inspection.BlockSize); err != nil {
			c.Configuration.Export.Inspection.BlockSize = defaultCfg.Export.Inspection.BlockSize
		}
	}

	if !c.Configuration.Export.Inspection.ModifiedBefore.IsZero() && !c.Configuration.Export.ModifiedAfter.IsZero() {
		if c.Configuration.Export.Inspection.ModifiedBefore.Before(c.Configuration.Export.ModifiedAfter.Time) ||
			c.Configuration.Export.Inspection.ModifiedBefore.Equal(c.Configuration.Export.ModifiedAfter.Time) {
			c.Configuration.Export.Inspection.ModifiedBefore = defaultCfg.Export.Inspection.ModifiedBefore
		}
	}
}

// SaveConfiguration will save the configuration to the file
func (c *ConfigurationManager) SaveConfiguration() error {
	if len(strings.TrimSpace(c.fileName)) == 0 || !strings.HasSuffix(c.fileName, ".yaml") {
		return fmt.Errorf("invalid file name provided")
	}

	data, err := yaml.Marshal(c.Configuration)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(filepath.Join(c.path, c.fileName), data, 0666); err != nil {
		return fmt.Errorf("writing file %s: %w", c.fileName, err)
	}
	return nil
}

// BuildConfigurationWithDefaults will set up an initial configuration with default values
func BuildConfigurationWithDefaults() *ExporterConfiguration {
	exportLocation := getExportLocation()
	mediaPathLocation := filepath.Join(exportLocation, "media")

	cfg := &ExporterConfiguration{}
	cfg.API.SheqsyURL = "https://app.sheqsy.com"
	cfg.API.URL = "https://api.safetyculture.io"
	cfg.API.MaxConcurrency = 10
	cfg.Csv.MaxRowsPerFile = 1000000
	cfg.Db.Dialect = "mysql"
	cfg.Export.Tables = []string{}
	cfg.Export.TemplateIds = []string{}
	cfg.Export.Action.Limit = 100
	cfg.Export.Asset.Limit = 100
	cfg.Export.Course.Progress.Limit = 1000
	cfg.Export.Course.Progress.CompletionStatus = "COMPLETION_STATUS_COMPLETED"
	cfg.Export.Incremental = true
	cfg.Export.Inspection.Archived = "false"
	cfg.Export.Inspection.Completed = "true"
	cfg.Export.Inspection.ModifiedBefore = mTime{}
	cfg.Export.Inspection.BlockSize = ""
	cfg.Export.Inspection.Limit = 100
	cfg.Export.Inspection.SkipIds = []string{}
	cfg.Export.Inspection.WebReportLink = "private"
	cfg.Export.InspectionItems.SkipFields = []string{}
	cfg.Export.Issue.Limit = 100
	cfg.Export.Path = exportLocation
	cfg.Export.MediaPath = mediaPathLocation
	cfg.Export.TimeZone = "UTC"
	cfg.Export.ModifiedAfter = mTime{}
	cfg.Export.Site.IncludeFullHierarchy = true
	cfg.Report.FilenameConvention = "INSPECTION_TITLE"
	cfg.Report.Format = []string{"PDF"}
	cfg.Report.RetryTimeout = 15
	cfg.Session.ExportType = "csv"

	err := os.MkdirAll(cfg.Export.Path, os.ModePerm)
	if err != nil {
		cfg.Export.Path = filepath.Join("export")
		cfg.Export.MediaPath = filepath.Join("export", "media")
	}

	return cfg
}

func getExportLocation() string {
	exportLocation, err := filepath.Rel("./", "export")
	if err != nil {
		exportLocation = "export"
	}
	return exportLocation
}

// NewConfigurationManagerFromFile will create a ConfigurationManager with data from the specified file
func NewConfigurationManagerFromFile(path string, fileName string) (*ConfigurationManager, error) {
	cm := &ConfigurationManager{
		path:          path,
		fileName:      fileName,
		Configuration: &ExporterConfiguration{},
	}

	err := cm.loadConfiguration()
	if err != nil {
		return nil, err
	}

	cm.ApplySafetyGuards()
	return cm, nil
}

// NewConfigurationManager will create a ConfigurationManager with default data,
func NewConfigurationManager(path string, fileName string) *ConfigurationManager {
	cm := &ConfigurationManager{
		path:          path,
		fileName:      fileName,
		Configuration: BuildConfigurationWithDefaults(),
	}

	cm.ApplySafetyGuards()
	return cm
}

func (ec *ExporterConfiguration) ToExporterConfig() *feed.ExporterFeedCfg {
	return &feed.ExporterFeedCfg{
		AccessToken:                           ec.AccessToken,
		ExportTables:                          ec.Export.Tables,
		SheqsyUsername:                        ec.SheqsyUsername,
		SheqsyCompanyID:                       ec.SheqsyCompanyID,
		ExportInspectionSkipIds:               ec.Export.Inspection.SkipIds,
		ExportModifiedAfterTime:               ec.Export.ModifiedAfter.Time,
		ExportModifiedBeforeTime:              ec.Export.Inspection.ModifiedBefore.Time,
		ExportBlockSize:                       ec.Export.Inspection.BlockSize,
		ExportTemplateIds:                     ec.Export.TemplateIds,
		ExportInspectionArchived:              ec.Export.Inspection.Archived,
		ExportInspectionCompleted:             ec.Export.Inspection.Completed,
		ExportInspectionIncludedInactiveItems: ec.Export.Inspection.IncludedInactiveItems,
		ExportInspectionWebReportLink:         ec.Export.Inspection.WebReportLink,
		ExportInspectionItemsSkipFields:       ec.Export.InspectionItems.SkipFields,
		ExportScheduleResumeDownload:          ec.Export.Schedule.ResumeDownload,
		ExportIncremental:                     ec.Export.Incremental,
		ExportInspectionLimit:                 ec.Export.Inspection.Limit,
		ExportMedia:                           ec.Export.Media,
		ExportSiteIncludeDeleted:              ec.Export.Site.IncludeDeleted,
		ExportActionLimit:                     ec.Export.Action.Limit,
		ExportSiteIncludeFullHierarchy:        ec.Export.Site.IncludeFullHierarchy,
		ExportIssueLimit:                      ec.Export.Issue.Limit,
		ExportAssetLimit:                      ec.Export.Asset.Limit,
		ExportCourseProgressLimit:             ec.Export.Course.Progress.Limit,
		MaxConcurrentGoRoutines:               ec.API.MaxConcurrency,
	}
}

func (ec *ExporterConfiguration) ToReporterConfig() *ReportExporterCfg {
	return &ReportExporterCfg{
		Format:       ec.Report.Format,
		PreferenceID: ec.Report.PreferenceID,
		Filename:     ec.Report.FilenameConvention,
		RetryTimeout: ec.Report.RetryTimeout,
	}
}

func (ec *ExporterConfiguration) ToInspectionConfig() *inspections.InspectionClientCfg {
	return &inspections.InspectionClientCfg{
		SkipIDs:       ec.Export.Inspection.SkipIds,
		ModifiedAfter: ec.Export.ModifiedAfter.Time,
		TemplateIDs:   ec.Export.TemplateIds,
		Archived:      ec.Export.Inspection.Archived,
		Completed:     ec.Export.Inspection.Completed,
		Incremental:   ec.Export.Incremental,
	}
}

func (ec *ExporterConfiguration) ToApiConfig() *HttpApiCfg {
	return &HttpApiCfg{
		tlsSkipVerify:  ec.API.TLSSkipVerify,
		tlsCert:        ec.API.TLSCert,
		proxyUrl:       ec.API.ProxyURL,
		apiUrl:         ec.API.URL,
		accessToken:    ec.AccessToken,
		sheqsyApiUrl:   ec.API.SheqsyURL,
		sheqsyUsername: ec.SheqsyUsername,
		sheqsyPassword: ec.SheqsyPassword,
	}
}
