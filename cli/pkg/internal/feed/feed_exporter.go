package feed

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MickStanciu/go-fn/fn"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
)

/*
*
NOTE: these functions were migrated from various feed methods and adapted not to use viper
They are called directly by the CMD from export cmd.package
*/
const maxConcurrentGoRoutines = 10

// MittiFeedExporter defines the basic action in regard to the exporter
type MittiFeedExporter interface {
	// ExportSchemas will generate the schema for SafetyCulture feeds, without downloading data
	ExportSchemas(exporter Exporter) error
	// ExportFeeds will export SafetyCulture feeds and Sheqsy feeds
	ExportFeeds(exporter Exporter) error
	// PrintSchemas is used to print the schema of each feed to console output
	PrintSchemas(exporter *SchemaExporter) error
	// ExportInspectionReports download all the reports for inspections and stores them on disk
	ExportInspectionReports(exporter *ReportExporter) error
}

type ExporterFeedClient struct {
	configuration   *ExporterFeedCfg
	apiClient       *httpapi.Client
	sheqsyApiClient *httpapi.Client
	errMu           sync.Mutex
	errs            []error
}

type ExporterFeedCfg struct {
	AccessToken                           string
	ExportTables                          []string
	SheqsyUsername                        string
	SheqsyCompanyID                       string
	ExportInspectionSkipIds               []string
	ExportModifiedAfterTime               time.Time
	ExportModifiedBeforeTime              time.Time
	ExportBlockSize                       string
	ExportTemplateIds                     []string
	ExportInspectionArchived              string
	ExportInspectionCompleted             string
	ExportInspectionIncludedInactiveItems bool
	ExportInspectionWebReportLink         string
	ExportInspectionItemsSkipFields       []string
	ExportIncremental                     bool
	ExportInspectionLimit                 int
	ExportMedia                           bool
	ExportSiteIncludeDeleted              bool
	ExportActionLimit                     int
	ExportSiteIncludeFullHierarchy        bool
	ExportIssueLimit                      int
	ExportAssetLimit                      int
	ExportCourseProgressLimit             int
	ExportScheduleResumeDownload          bool
	MaxConcurrentGoRoutines               int
}

func NewExporterApp(scApiClient *httpapi.Client, sheqsyApiClient *httpapi.Client, cfg *ExporterFeedCfg) *ExporterFeedClient {
	return &ExporterFeedClient{
		configuration:   cfg,
		apiClient:       scApiClient,
		sheqsyApiClient: sheqsyApiClient,
	}
}

// ExportSchemas generates schemas for the data feeds without fetching any data
func (e *ExporterFeedClient) ExportSchemas(exporter Exporter) error {
	var lastErr error = nil
	feeds := e.GetFeeds()
	for _, feed := range feeds {
		lastErr = feed.CreateSchema(exporter)
	}

	return lastErr
}

func (e *ExporterFeedClient) addError(err error) {
	e.errMu.Lock()
	e.errs = append(e.errs, err)
	e.errMu.Unlock()
}

// ExportFeeds fetches all the feeds data from server and stores them in the format provided
func (e *ExporterFeedClient) ExportFeeds(exporter Exporter, ctx context.Context) error {
	log := logger.GetLogger()

	status := GetExporterStatus()
	status.Reset()

	tables := e.configuration.ExportTables
	tablesMap := map[string]bool{}
	for _, table := range tables {
		tablesMap[table] = true
	}

	maxConcurrentRoutines := fn.GetOrElse(e.configuration.MaxConcurrentGoRoutines, maxConcurrentGoRoutines, func(i int) bool {
		return i > 0
	})

	var wg sync.WaitGroup
	semaphore := make(chan int, maxConcurrentRoutines)

	atLeastOneRun := false

	// Run export for SafetyCulture data
	if len(e.configuration.AccessToken) != 0 {
		status.started = true
		atLeastOneRun = true
		log.Info("exporting SafetyCulture data")

		var feeds []Feed
		for _, feed := range e.GetFeeds() {
			if tablesMap[feed.Name()] || len(tables) == 0 {
				feeds = append(feeds, feed)
			}
		}

		resp, err := httpapi.WhoAmI(ctx, e.apiClient)
		if err != nil {
			return fmt.Errorf("get details of the current user: %w", err)
		}

		log = log.With(
			"user.id", resp.UserID,
			"user.org_id", resp.OrganisationID,
			"user.name", fmt.Sprintf("%s %s", resp.Firstname, resp.Lastname),
		)

		log.Infof("exporting data for user")

		if len(feeds) == 0 {
			return errors.New("no tables selected")
		}

		for _, feed := range feeds {
			semaphore <- 1
			wg.Add(1)

			go func(f Feed, c context.Context) {
				defer wg.Done()
				select {
				case <-c.Done():
					log.Infof(" ... canceling export")
					return
				default:
					log.Infof(" ... queueing %s\n", f.Name())
					status.StartFeedExport(f.Name(), f.HasRemainingInformation())
					exportErr := f.Export(c, e.apiClient, exporter, resp.OrganisationID)
					var curatedErr error
					if exportErr != nil {
						e.addError(exportErr)
						if events.IsBlockingError(exportErr) {
							log.Errorf("exporting feeds: %v", exportErr)
							curatedErr = exportErr
						}
					}
					status.FinishFeedExport(f.Name(), curatedErr)
					<-semaphore
				}
			}(feed, ctx)
		}

	}

	// Run export for SHEQSY data
	if len(e.configuration.SheqsyUsername) != 0 {
		atLeastOneRun = true
		log.Info("exporting SHEQSY data")

		var feeds []Feed
		for _, feed := range e.GetSheqsyFeeds() {
			if tablesMap[feed.Name()] || len(tables) == 0 {
				feeds = append(feeds, feed)
			}
		}

		resp, err := httpapi.GetSheqsyCompany(ctx, e.sheqsyApiClient, e.configuration.SheqsyCompanyID)
		if err != nil {
			return fmt.Errorf("get details of the current user: %w", err)
		}

		log.Infof("Exporting data for SHEQSY company: %s %s", resp.Name, resp.CompanyUID)

		if len(feeds) == 0 {
			return errors.New("no tables selected")
		}

		for _, feed := range feeds {
			semaphore <- 1
			wg.Add(1)

			go func(f Feed) {
				log.Infof(" ... queueing %s\n", f.Name())
				defer wg.Done()
				err := f.Export(ctx, e.sheqsyApiClient, exporter, resp.CompanyUID)
				if err != nil {
					e.addError(err)
				}
				<-semaphore
			}(feed)
		}
	}

	wg.Wait()

	if !atLeastOneRun {
		return errors.New("no API tokens provided")
	}

	log.Info("Export finished")
	status.MarkExportCompleted()

	if len(e.errs) != 0 {
		log.Warn("These were errors during the export:")
		for _, ee := range e.errs {
			switch theError := ee.(type) {
			case *events.EventError:
				theError.Log(log)
			default:
				log.Infof(" > %s", theError.Error())
			}
		}

		return e.errs[0]
	}

	return nil
}

// GetFeeds returns list of available SafetyCulture feeds
func (e *ExporterFeedClient) GetFeeds() []Feed {
	return []Feed{
		e.getInspectionFeed(),
		&UserFeed{},
		&TemplateFeed{
			Incremental: e.configuration.ExportIncremental,
		},
		&TemplatePermissionFeed{
			Incremental: e.configuration.ExportIncremental,
		},
		&SiteFeed{
			IncludeDeleted:       e.configuration.ExportSiteIncludeDeleted,
			IncludeFullHierarchy: e.configuration.ExportSiteIncludeFullHierarchy,
		},
		&SiteMemberFeed{},
		&GroupFeed{},
		&GroupUserFeed{},
		&ScheduleFeed{
			TemplateIDs: e.configuration.ExportTemplateIds,
		},
		&ScheduleAssigneeFeed{
			TemplateIDs: e.configuration.ExportTemplateIds,
		},
		&ScheduleOccurrenceFeed{
			TemplateIDs:    e.configuration.ExportTemplateIds,
			StartDate:      e.configuration.ExportModifiedAfterTime,
			ResumeDownload: e.configuration.ExportScheduleResumeDownload,
		},
		&ActionFeed{
			ModifiedAfter: e.configuration.ExportModifiedAfterTime,
			Incremental:   e.configuration.ExportIncremental,
			Limit:         e.configuration.ExportActionLimit,
		},
		&ActionAssigneeFeed{
			ModifiedAfter: e.configuration.ExportModifiedAfterTime,
			Incremental:   e.configuration.ExportIncremental,
			Limit:         e.configuration.ExportActionLimit,
		},
		&ActionTimelineItemFeed{
			ModifiedAfter: e.configuration.ExportModifiedAfterTime,
			Incremental:   e.configuration.ExportIncremental,
			Limit:         e.configuration.ExportActionLimit,
		},
		&InspectionItemFeed{
			SkipIDs:         e.configuration.ExportInspectionSkipIds,
			SkipFields:      e.configuration.ExportInspectionItemsSkipFields,
			ModifiedAfter:   e.configuration.ExportModifiedAfterTime,
			ModifiedBefore:  e.configuration.ExportModifiedBeforeTime,
			BlockSize:       e.configuration.ExportBlockSize,
			TemplateIDs:     e.configuration.ExportTemplateIds,
			Archived:        e.configuration.ExportInspectionArchived,
			Completed:       e.configuration.ExportInspectionCompleted,
			IncludeInactive: e.configuration.ExportInspectionIncludedInactiveItems,
			Incremental:     e.configuration.ExportIncremental,
			Limit:           e.configuration.ExportInspectionLimit,
			ExportMedia:     e.configuration.ExportMedia,
		},
		&IssueFeed{
			Incremental: false, // this was disabled on request. Issues API doesn't support modified After filters
			Limit:       e.configuration.ExportIssueLimit,
		},
		&IssueTimelineItemFeed{
			Incremental: false, // Issues API doesn't support modified after filters
			Limit:       e.configuration.ExportIssueLimit,
		},
		&AssetFeed{
			Incremental: false, // Assets API doesn't support modified after filters
			Limit:       e.configuration.ExportAssetLimit,
		},
		&TrainingCourseProgressFeed{
			Incremental:      false, // CourseProgress doesn't support modified after filters,
			Limit:            e.configuration.ExportCourseProgressLimit,
			CompletionStatus: "COMPLETION_STATUS_COMPLETED",
		},
		&IssueAssigneeFeed{
			Incremental: false, // IssueAssignee doesn't support modified after filters
			Limit:       e.configuration.ExportIssueLimit,
		},
		&AccountHistoryFeed{
			Incremental: true,
			Limit:       250,
		},
	}
}

func (e *ExporterFeedClient) getInspectionFeed() *InspectionFeed {
	return &InspectionFeed{
		SkipIDs:        e.configuration.ExportInspectionSkipIds,
		ModifiedAfter:  e.configuration.ExportModifiedAfterTime,
		ModifiedBefore: e.configuration.ExportModifiedBeforeTime,
		BlockSize:      e.configuration.ExportBlockSize,
		TemplateIDs:    e.configuration.ExportTemplateIds,
		Archived:       e.configuration.ExportInspectionArchived,
		Completed:      e.configuration.ExportInspectionCompleted,
		Incremental:    e.configuration.ExportIncremental,
		Limit:          e.configuration.ExportInspectionLimit,
		WebReportLink:  e.configuration.ExportInspectionWebReportLink,
	}
}

// GetSheqsyFeeds returns list of all available data feeds for sheqsy
func (e *ExporterFeedClient) GetSheqsyFeeds() []Feed {
	return []Feed{
		&SheqsyEmployeeFeed{},
		&SheqsyDepartmentEmployeeFeed{},
		&SheqsyDepartmentFeed{},
		&SheqsyActivityFeed{},
		&SheqsyShiftFeed{},
	}
}

// PrintSchemas is used to print the schema of each feed to console output
func (e *ExporterFeedClient) PrintSchemas(exporter *SchemaExporter) error {
	for _, feed := range append(e.GetFeeds(), e.GetSheqsyFeeds()...) {
		if err := exporter.CreateSchema(feed, feed.RowsModel()); err != nil {
			return fmt.Errorf("create schema: %w", err)
		}

		if err := exporter.WriteSchema(feed); err != nil {
			return fmt.Errorf("write schema: %w", err)
		}
	}
	return nil
}

func (e *ExporterFeedClient) ExportInspectionReports(exporter *ReportExporter, ctx context.Context) error {
	log := logger.GetLogger()
	status := GetExporterStatus()
	status.Reset()
	status.started = true

	resp, err := httpapi.WhoAmI(ctx, e.apiClient)
	if err != nil {
		return fmt.Errorf("get details of the current user: %w", err)
	}

	log.Infof("Exporting inspection reports by user: %s %s", resp.Firstname, resp.Lastname)

	feed := e.getInspectionFeed()
	status.StartFeedExport(feed.Name(), feed.HasRemainingInformation())
	if err := feed.Export(ctx, e.apiClient, exporter, resp.OrganisationID); err != nil {
		status.FinishFeedExport(feed.Name(), err)
		status.MarkExportCompleted()
		return fmt.Errorf("export inspection feed: %w", err)
	}

	err = exporter.SaveReports(ctx, e.apiClient, feed)
	if err != nil {
		return fmt.Errorf("save reports: %w", err)
	}

	return err
}
