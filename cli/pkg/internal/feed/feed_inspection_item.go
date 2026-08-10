package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

const maxGoRoutines = 10

const IgnoreMediaHypertextReference = "media_hypertext_reference"

// InspectionItem represents a row from the inspection_items feed
type InspectionItem struct {
	ID                      string    `json:"id" csv:"id" gorm:"primarykey;size:150"`
	ItemID                  string    `json:"item_id" csv:"item_id" gorm:"size:100"`
	AuditID                 string    `json:"audit_id" csv:"audit_id" gorm:"size:100"`
	ItemIndex               int64     `json:"item_index" csv:"item_index"`
	TemplateID              string    `json:"template_id" csv:"template_id" gorm:"size:100"`
	ParentID                string    `json:"parent_id" csv:"parent_id" gorm:"size:100"`
	CreatedAt               time.Time `json:"created_at" csv:"created_at"`
	ModifiedAt              time.Time `json:"modified_at" csv:"modified_at" gorm:"index:idx_ins_itm_modified_at,sort:desc"`
	ExportedAt              time.Time `json:"exported_at" csv:"exported_at" gorm:"index:idx_ins_itm_modified_at;autoUpdateTime"`
	Type                    string    `json:"type" csv:"type" gorm:"size:20"`
	Category                string    `json:"category" csv:"category"`
	CategoryID              string    `json:"category_id" csv:"category_id" gorm:"size:100"`
	OrganisationID          string    `json:"organisation_id" csv:"organisation_id" gorm:"index:idx_ins_itm_modified_at;size:37"`
	ParentIDs               string    `json:"parent_ids" csv:"parent_ids"`
	Label                   string    `json:"label" csv:"label"`
	Response                string    `json:"response" csv:"response"`
	ResponseID              string    `json:"response_id" csv:"response_id" gorm:"size:100"`
	ResponseSetID           string    `json:"response_set_id" csv:"response_set_id" gorm:"size:100"`
	IsFailedResponse        bool      `json:"is_failed_response" csv:"is_failed_response"`
	Comment                 string    `json:"comment" csv:"comment"`
	MediaFiles              string    `json:"media_files" csv:"media_files"`
	MediaIDs                string    `json:"media_ids" csv:"media_ids"`
	MediaHypertextReference string    `json:"media_hypertext_reference" csv:"media_hypertext_reference"`
	Score                   float32   `json:"score" csv:"score"`
	MaxScore                float32   `json:"max_score" csv:"max_score"`
	ScorePercentage         float32   `json:"score_percentage" csv:"score_percentage"`
	CombinedScore           float32   `json:"combined_score" csv:"combined_score"`
	CombinedMaxScore        float32   `json:"combined_max_score" csv:"combined_max_score"`
	CombinedScorePercentage float32   `json:"combined_score_percentage" csv:"combined_score_percentage"`
	Mandatory               bool      `json:"mandatory" csv:"mandatory"`
	Inactive                bool      `json:"inactive" csv:"inactive"`
	LocationLatitude        *float32  `json:"location_latitude" csv:"location_latitude"`
	LocationLongitude       *float32  `json:"location_longitude" csv:"location_longitude"`
	PrimeelementID          string    `json:"primeelement_id" csv:"primeelement_id" gorm:"size:100"`
	PrimeelementIndex       *int64    `json:"primeelement_index" csv:"primeelement_index"`
}

// InspectionItemFeed is a representation of the inspection_items feed
type InspectionItemFeed struct {
	SkipIDs         []string
	SkipFields      []string
	ModifiedAfter   time.Time
	ModifiedBefore  time.Time
	BlockSize       string
	TemplateIDs     []string
	Archived        string
	Completed       string
	IncludeInactive bool
	Incremental     bool
	ExportMedia     bool
	Limit           int
}

// Name is the name of the feed
func (f *InspectionItemFeed) Name() string {
	return "inspection_items"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *InspectionItemFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *InspectionItemFeed) Model() interface{} {
	return InspectionItem{}
}

// RowsModel returns the model of feed rows
func (f *InspectionItemFeed) RowsModel() interface{} {
	return &[]*InspectionItem{}
}

// PrimaryKey returns the primary key(s)
func (f *InspectionItemFeed) PrimaryKey() []string {
	return []string{"id"}
}

// Columns returns the columns of the row
func (f *InspectionItemFeed) Columns() []string {
	return []string{
		"item_index",
		"template_id",
		"parent_id",
		"created_at",
		"modified_at",
		"type",
		"category",
		"category_id",
		"organisation_id",
		"parent_ids",
		"label",
		"response",
		"response_id",
		"response_set_id",
		"is_failed_response",
		"comment",
		"media_files",
		"media_ids",
		"media_hypertext_reference",
		"score",
		"max_score",
		"score_percentage",
		"combined_score",
		"combined_max_score",
		"combined_score_percentage",
		"mandatory",
		"inactive",
		"location_latitude",
		"location_longitude",
		"primeelement_id",
		"primeelement_index",
	}
}

// Order returns the ordering when retrieving an export
func (f *InspectionItemFeed) Order() string {
	return "modified_at ASC, id"
}

func fetchAndWriteMedia(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, auditID, mediaURL string) error {
	status := GetExporterStatus()
	resp, err := GetMedia(
		ctx,
		apiClient,
		&GetMediaRequest{
			URL:     mediaURL,
			AuditID: auditID,
		},
	)
	if err != nil {
		return err
	}

	// If the response is empty, then ignore this media object
	if resp == nil {
		return nil
	}

	err = exporter.WriteMedia(auditID, resp.MediaID, resp.ContentType, resp.Body)
	if err != nil {
		return err
	}

	status.IncrementStatus("media", 1, 0)
	return nil
}

func (f *InspectionItemFeed) writeRows(ctx context.Context, exporter Exporter, rows []*InspectionItem, skipFields []string, apiClient *httpapi.Client) error {
	l := logger.GetLogger()
	skipIDs := map[string]bool{}
	for _, id := range f.SkipIDs {
		skipIDs[id] = true
	}

	// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
	batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 5)
	err := util.SplitSliceInBatch(batchSize, rows, func(batch []*InspectionItem) error {
		// you can specify level of concurrency by increasing channel size
		buffers := make(chan bool, maxGoRoutines)
		var wg sync.WaitGroup

		// Some audits in production have the same item ID multiple times
		// We can't insert them simultaneously. This means we are dropping data, which sucks.
		var rowsToInsert []*InspectionItem
		idSeen := map[string]bool{}
		for _, row := range batch {
			skip := skipIDs[row.AuditID]
			seen := idSeen[row.ID]
			if seen || skip {
				continue
			}
			idSeen[row.ID] = true
			rowsToInsert = append(rowsToInsert, processSkipFields(skipFields, row))

			if !f.ExportMedia || len(row.MediaHypertextReference) == 0 {
				continue
			}

			mediaURLList := strings.Split(row.MediaHypertextReference, "\n")
			if len(mediaURLList) > 0 {
				l.Infof(" downloading media for inspection item %s", row.ItemID)
			}

			for _, mediaURL := range mediaURLList {
				wg.Add(1)

				go func(mediaURL string, c context.Context) error {
					defer wg.Done()
					select {
					case <-c.Done():
						l.Infof(" ... canceling media downloads ")
						return nil
					default:
						buffers <- true

						if err := fetchAndWriteMedia(c, apiClient, exporter, row.AuditID, mediaURL); err != nil {
							return events.WrapEventError(err, "write media")
						}

						<-buffers
						return nil
					}

				}(mediaURL, ctx)
			}
			wg.Wait()
		}

		if err := exporter.WriteRows(f, rowsToInsert); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// processSkipFields - will remove the content if the field is in the list.
func processSkipFields(fields []string, row *InspectionItem) *InspectionItem {
	if len(fields) == 0 {
		return row
	}

	var rowClone = *row
	for _, field := range fields {
		switch field {
		case IgnoreMediaHypertextReference:
			rowClone.MediaHypertextReference = ""
		}
	}
	return &rowClone
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *InspectionItemFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*InspectionItem{})
}

// Export exports the feed to the supplied exporter
func (f *InspectionItemFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	var err error
	f.ModifiedAfter, err = exporter.LastModifiedAt(f, f.ModifiedAfter, orgID)
	if err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDB, false, "unable to load modified after")
	}

	// Process inspection items
	if err := f.processInspectionItems(ctx, apiClient, exporter, orgID, status); err != nil {
		return events.WrapEventError(err, "export")
	}

	err = exporter.FinaliseExport(f, &[]*InspectionItem{})
	if err != nil {
		return events.WrapEventError(err, "finalise export")
	}

	return nil
}

func (f *InspectionItemFeed) processInspectionItems(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string, status *ExportStatus) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)

	// Split date range into smaller blocks if blocks and process one block at a time if enabled
	if f.BlockSize != "" {
		return f.processInspectionItemsInBlocks(ctx, apiClient, exporter, orgID, status)
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*InspectionItem

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			if err := f.writeRows(ctx, exporter, rows, f.SkipFields, apiClient); err != nil {
				return err
			}
		}

		status.UpdateStatus(f.Name(), resp.Metadata.RemainingRecords, exporter.GetDuration().Milliseconds())

		l.With(
			"estimated_remaining", resp.Metadata.RemainingRecords,
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{
		InitialURL: "/feed/inspection_items",
		Params: GetFeedParams{
			ModifiedAfter:   f.ModifiedAfter,
			ModifiedBefore:  f.ModifiedBefore,
			TemplateIDs:     f.TemplateIDs,
			Archived:        f.Archived,
			Completed:       f.Completed,
			IncludeInactive: f.IncludeInactive,
			Limit:           f.Limit,
		},
	}

	if f.ExportMedia {
		status.StartFeedExport("media", false)
	}
	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		if f.ExportMedia {
			status.FinishFeedExport("media", err)
		}
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}
	if f.ExportMedia {
		status.FinishFeedExport("media", nil)
	}
	return nil
}

func (f *InspectionItemFeed) processInspectionItemsInBlocks(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string, status *ExportStatus) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)

	// Set end date to now if not specified
	endDate := f.ModifiedBefore
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// Generate time blocks
	blocks, err := util.GenerateTimeBlocksFromString(f.ModifiedAfter, endDate, f.BlockSize)
	if err != nil {
		return fmt.Errorf("failed to generate time blocks: %w", err)
	}

	if len(blocks) == 0 {
		l.Info("no time blocks to process")
		return nil
	}

	l.With("total_blocks", len(blocks), "block_size", f.BlockSize).Info("starting block-based export")

	// Process each block
	for i, block := range blocks {
		l.With("block", i+1, "total_blocks", len(blocks), "start", block.Start.Format(time.RFC3339), "end", block.End.Format(time.RFC3339)).Info("processing time block")

		req := GetFeedRequest{
			InitialURL: "/feed/inspection_items",
			Params: GetFeedParams{
				ModifiedAfter:   block.Start,
				ModifiedBefore:  block.End,
				TemplateIDs:     f.TemplateIDs,
				Archived:        f.Archived,
				Completed:       f.Completed,
				IncludeInactive: f.IncludeInactive,
				Limit:           f.Limit,
			},
		}

		drainFn := func(resp *GetFeedResponse) error {
			var rows []*InspectionItem

			if err := json.Unmarshal(resp.Data, &rows); err != nil {
				return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
			}

			if len(rows) != 0 {
				if err := f.writeRows(ctx, exporter, rows, f.SkipFields, apiClient); err != nil {
					return err
				}
			}

			status.UpdateStatus(f.Name(), resp.Metadata.RemainingRecords, exporter.GetDuration().Milliseconds())

			l.With(
				"block", i+1,
				"estimated_remaining", resp.Metadata.RemainingRecords,
				"duration_ms", apiClient.Duration.Milliseconds(),
				"export_duration_ms", exporter.GetDuration().Milliseconds(),
			).Info("export batch complete")

			return nil
		}

		if f.ExportMedia && i == 0 { // Start media export only once
			status.StartFeedExport("media", false)
		}

		if err := DrainFeed(ctx, apiClient, &req, drainFn); err != nil {
			l.With("block", i+1, "error", err).Error("failed to process block")
			if f.ExportMedia {
				status.FinishFeedExport("media", err)
			}
			return fmt.Errorf("failed to process block %d/%d: %w", i+1, len(blocks), err)
		}

		l.With("block", i+1, "total_blocks", len(blocks)).Info("completed time block")
	}

	// Finish media export if enabled
	if f.ExportMedia {
		status.FinishFeedExport("media", nil)
	}

	l.With("total_blocks", len(blocks)).Info("completed block-based export")
	return nil
}
