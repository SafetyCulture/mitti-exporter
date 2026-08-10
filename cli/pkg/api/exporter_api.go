package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/exporter"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/inspections"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/templates"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/pkg/errors"
)

var ctx context.Context
var cancelFunc context.CancelFunc

// NewMittiExporter builds a MittiExporter with clients inferred from own configuration
func NewMittiExporter(cfg *ExporterConfiguration, version *AppVersion) (*MittiExporter, error) {
	apiClient, err := getAPIClient(cfg.ToApiConfig(), version)
	if err != nil {
		return nil, err
	}

	sheqsyApiClient, err := getSheqsyAPIClient(cfg.ToApiConfig(), version)
	if err != nil {
		return nil, err
	}

	return &MittiExporter{
		apiClient:       apiClient,
		sheqsyApiClient: sheqsyApiClient,
		cfg:             cfg,
		version:         version,
		exportStatus:    feed.GetExporterStatus(),
	}, nil
}

func getAPIClient(cfg *HttpApiCfg, version *AppVersion) (*httpapi.Client, error) {
	var apiOpts []httpapi.Opt

	if cfg.tlsSkipVerify {
		apiOpts = append(apiOpts, httpapi.OptSetInsecureTLS(true))
	}
	if cfg.tlsCert != "" {
		apiOpts = append(apiOpts, httpapi.OptAddTLSCert(cfg.tlsCert))
	}
	if cfg.proxyUrl != "" {
		proxyURL, err := url.Parse(cfg.proxyUrl)
		if err != nil {
			return nil, fmt.Errorf("unable to parse proxy URL")
		}
		apiOpts = append(apiOpts, httpapi.OptSetProxy(proxyURL))
	}

	config := httpapi.ClientCfg{
		Addr:                cfg.apiUrl,
		AuthorizationHeader: fmt.Sprintf("Bearer %s", cfg.accessToken),
		IntegrationID:       version.IntegrationID,
		IntegrationVersion:  version.IntegrationVersion,
	}

	return httpapi.NewClient(&config, apiOpts...), nil
}

func getSheqsyAPIClient(cfg *HttpApiCfg, version *AppVersion) (*httpapi.Client, error) {
	var apiOpts []httpapi.Opt
	if cfg.tlsSkipVerify {
		apiOpts = append(apiOpts, httpapi.OptSetInsecureTLS(true))
	}
	if cfg.tlsCert != "" {
		apiOpts = append(apiOpts, httpapi.OptAddTLSCert(cfg.tlsCert))
	}
	if cfg.proxyUrl != "" {
		proxyURL, err := url.Parse(cfg.proxyUrl)
		if err != nil {
			return nil, fmt.Errorf("unable to parse proxy URL")
		}
		apiOpts = append(apiOpts, httpapi.OptSetProxy(proxyURL))
	}

	token := base64.StdEncoding.EncodeToString(
		[]byte(
			fmt.Sprintf(
				"%s:%s",
				cfg.sheqsyUsername,
				cfg.sheqsyPassword,
			),
		),
	)

	config := httpapi.ClientCfg{
		Addr:                cfg.sheqsyApiUrl,
		AuthorizationHeader: fmt.Sprintf("Basic %s", token),
		IntegrationID:       version.IntegrationID,
		IntegrationVersion:  version.IntegrationVersion,
	}

	return httpapi.NewClient(&config, apiOpts...), nil
}

// NewReportExporter returns a new instance of ReportExporter
func NewReportExporter(exportPath string, reportCfg *ReportExporterCfg) (*feed.ReportExporter, error) {
	sqlExporter, err := feed.NewSQLExporter("sqlite", filepath.Join(exportPath, "reports.db"), true, "")
	if err != nil {
		return nil, err
	}
	if res := sqlExporter.DB.Exec("PRAGMA busy_timeout = 20000"); res.Error != nil {
		return nil, res.Error
	}

	return &feed.ReportExporter{
		SQLExporter:  sqlExporter,
		Logger:       sqlExporter.Logger,
		ExportPath:   exportPath,
		Format:       reportCfg.Format,
		PreferenceID: reportCfg.PreferenceID,
		Filename:     reportCfg.Filename,
		RetryTimeout: reportCfg.RetryTimeout,
	}, nil
}

type ReportExporterCfg struct {
	Format       []string
	PreferenceID string
	Filename     string
	RetryTimeout int
}

type HttpApiCfg struct {
	tlsSkipVerify  bool
	tlsCert        string
	proxyUrl       string
	apiUrl         string
	accessToken    string
	sheqsyApiUrl   string
	sheqsyUsername string
	sheqsyPassword string
}

type MittiExporter struct {
	apiClient       *httpapi.Client
	sheqsyApiClient *httpapi.Client
	cfg             *ExporterConfiguration
	version         *AppVersion
	exportStatus    *feed.ExportStatus
}

func (s *MittiExporter) SetApiClient(apiClient *httpapi.Client) {
	s.apiClient = apiClient
}

func (s *MittiExporter) SetSheqsyApiClient(apiClient *httpapi.Client) {
	s.sheqsyApiClient = apiClient
}

func (s *MittiExporter) RunInspectionJSON() error {
	exportPath := fmt.Sprintf("%s/json/", s.cfg.Export.Path)
	err := os.MkdirAll(exportPath, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Failed to create directory %s", exportPath)
	}

	e := exporter.NewJSONExporter(exportPath)
	cfg := inspections.InspectionClientCfg{
		SkipIDs:       s.cfg.Export.Inspection.SkipIds,
		ModifiedAfter: s.cfg.Export.ModifiedAfter.Time,
		TemplateIDs:   s.cfg.Export.TemplateIds,
		Archived:      s.cfg.Export.Inspection.Archived,
		Completed:     s.cfg.Export.Inspection.Completed,
		Incremental:   s.cfg.Export.Incremental,
	}
	inspectionsClient := inspections.NewInspectionClient(&cfg, s.apiClient, e)

	err = inspectionsClient.Export(context.Background())
	if err != nil {
		return errors.Wrap(err, "error while exporting JSON")
	}
	return nil
}

func (s *MittiExporter) CheckDBConnection() error {
	_, err := feed.GetDatabase(s.cfg.Db.Dialect, s.cfg.Db.ConnectionString)
	if err != nil {
		return errors.Wrap(err, "create sql exporter")
	}
	return nil
}

func (s *MittiExporter) RunSQL() error {
	ctx, cancelFunc = context.WithCancel(context.Background())
	if s.cfg.Export.Media {
		err := os.MkdirAll(s.cfg.Export.MediaPath, os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "Failed to create directory %s", s.cfg.Export.MediaPath)
		}
	}

	e, err := feed.NewSQLExporter(s.cfg.Db.Dialect, s.cfg.Db.ConnectionString, !s.cfg.Db.AutoMigrateDisabled, s.cfg.Export.MediaPath)
	if err != nil {
		return errors.Wrap(err, "create sql exporter")
	}

	exporterApp := feed.NewExporterApp(s.apiClient, s.sheqsyApiClient, s.cfg.ToExporterConfig())
	if s.cfg.Export.SchemaOnly {
		return exporterApp.ExportSchemas(e)
	}

	if len(s.cfg.AccessToken) != 0 || len(s.cfg.SheqsyUsername) != 0 {
		err = exporterApp.ExportFeeds(e, ctx)
		if err != nil {
			return errors.Wrap(err, "exporting feeds")
		}
	}

	return nil
}

// RunSQLite - runs the export and will save into a local sqlite db file
func (s *MittiExporter) RunSQLite() error {
	ctx, cancelFunc = context.WithCancel(context.Background())
	exportPath := s.cfg.Export.Path

	err := os.MkdirAll(exportPath, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Failed to create directory %s", exportPath)
	}

	if s.cfg.Export.Media {
		err := os.MkdirAll(s.cfg.Export.MediaPath, os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "Failed to create directory %s", s.cfg.Export.MediaPath)
		}
	}

	sqlExporter, err := feed.NewSQLiteExporter(exportPath, s.cfg.Export.MediaPath)
	if err != nil {
		return errors.Wrap(err, "unable to create sqlite exporter")
	}

	exporterApp := feed.NewExporterApp(s.apiClient, s.sheqsyApiClient, s.cfg.ToExporterConfig())
	if s.cfg.Export.SchemaOnly {
		return exporterApp.ExportSchemas(sqlExporter)
	}

	if len(s.cfg.AccessToken) != 0 || len(s.cfg.SheqsyUsername) != 0 {
		err = exporterApp.ExportFeeds(sqlExporter, ctx)
		if err != nil {
			return errors.Wrap(err, "exporting feeds")
		}
	}

	return nil
}

func (s *MittiExporter) RunCSV() error {
	ctx, cancelFunc = context.WithCancel(context.Background())
	exportPath := s.cfg.Export.Path

	err := os.MkdirAll(exportPath, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Failed to create directory %s", exportPath)
	}

	if s.cfg.Export.Media {
		err := os.MkdirAll(s.cfg.Export.MediaPath, os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "Failed to create directory %s", s.cfg.Export.MediaPath)
		}
	}

	e, err := feed.NewCSVExporter(exportPath, s.cfg.Export.MediaPath, s.cfg.Csv.MaxRowsPerFile)
	if err != nil {
		return errors.Wrap(err, "unable to create csv exporter")
	}

	exporterApp := feed.NewExporterApp(s.apiClient, s.sheqsyApiClient, s.cfg.ToExporterConfig())
	if s.cfg.Export.SchemaOnly {
		return exporterApp.ExportSchemas(e)
	}

	if len(s.cfg.AccessToken) != 0 || len(s.cfg.SheqsyUsername) != 0 {
		err = exporterApp.ExportFeeds(e, ctx)
		if err != nil {
			return errors.Wrap(err, "exporting feeds")
		}
	}

	return nil
}

func (s *MittiExporter) RunInspectionReports() error {
	ctx, cancelFunc = context.WithCancel(context.Background())
	err := os.MkdirAll(s.cfg.Export.Path, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Failed to create directory %s", s.cfg.Export.Path)
	}

	e, err := NewReportExporter(s.cfg.Export.Path, s.cfg.ToReporterConfig())
	if err != nil {
		return errors.Wrap(err, "unable to create report exporter")
	}

	// Create a dedicated client with a longer timeout for report downloads,
	// as Word report generation and download can exceed the default 120s timeout.
	reportClient, err := getAPIClient(s.cfg.ToApiConfig(), s.version)
	if err != nil {
		return errors.Wrap(err, "unable to create report download client")
	}
	reportClient.HTTPClient().Timeout = 10 * time.Minute
	e.ReportClient = reportClient

	exporterApp := feed.NewExporterApp(s.apiClient, s.sheqsyApiClient, s.cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(e, ctx)
	if err != nil {
		return errors.Wrap(err, "generate reports")
	}

	return nil
}

func (s *MittiExporter) RunPrintSchema() error {
	e, err := feed.NewSchemaExporter(os.Stdout)
	if err != nil {
		return errors.Wrap(err, "unable to create exporter")
	}

	exporterApp := feed.NewExporterApp(s.apiClient, s.sheqsyApiClient, s.cfg.ToExporterConfig())
	err = exporterApp.PrintSchemas(e)
	if err != nil {
		return errors.Wrap(err, "error while printing schema")
	}

	return nil
}

func (s *MittiExporter) GetTemplateList() []TemplateResponseItem {
	client := templates.NewTemplatesClient(s.apiClient)
	res := client.GetTemplateList(context.Background(), 1000)

	transformer := func(data templates.TemplateResponseItem) TemplateResponseItem {
		return TemplateResponseItem{
			ID:         data.ID,
			Name:       data.Name,
			ModifiedAt: data.ModifiedAt,
		}
	}

	return util.GenericCollectionMapper(res, transformer)
}

// GetExportStatus called by UI
func (s *MittiExporter) GetExportStatus() *ExportStatusResponse {
	data := s.exportStatus.ReadStatus()
	var res []ExportStatusResponseItem

	for _, v := range data {
		res = append(res, ExportStatusResponseItem{
			FeedName:           v.Name,
			Started:            v.Started,
			Finished:           v.Finished,
			HasError:           v.HasError,
			DurationMs:         v.DurationMs,
			Counter:            v.Counter,
			CounterDecremental: v.CounterDecremental,
			StatusMessage:      v.StatusMessage,
			Stage:              string(v.Stage),
		})
	}

	s.exportStatus.PurgeFinished()

	return &ExportStatusResponse{
		ExportStarted:   s.exportStatus.GetExportStarted(),
		ExportCompleted: s.exportStatus.GetExportCompleted(),
		Feeds:           res,
	}
}

// SetConfiguration will replace the configuration. Used by the UI to pass in the newly saved configuration
func (s *MittiExporter) SetConfiguration(cfg *ExporterConfiguration) {
	s.cfg = cfg
}

// CleanExportStatus will clean the status items. Used by the UI
func (s *MittiExporter) CleanExportStatus() {
	s.exportStatus.Reset()
}

func (s *MittiExporter) CancelExport() {
	cancelFunc()
}
