package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

// Action represents a row from the actions feed
type Action struct {
	ID              string     `json:"id" csv:"action_id" gorm:"primarykey;column:action_id;size:36"`
	Title           string     `json:"title" csv:"title"`
	Description     string     `json:"description" csv:"description"`
	SiteID          string     `json:"site_id" csv:"site_id" gorm:"size:41"`
	Priority        string     `json:"priority" csv:"priority" gorm:"size:20"`
	Status          string     `json:"status" csv:"status" gorm:"size:20"`
	DueDate         time.Time  `json:"due_date" csv:"due_date"`
	CreatedAt       time.Time  `json:"created_at" csv:"created_at"`
	ModifiedAt      time.Time  `json:"modified_at" csv:"modified_at" gorm:"index:idx_act_modified_at,sort:desc"`
	ExportedAt      time.Time  `json:"exported_at" csv:"exported_at" gorm:"index:idx_act_modified_at;autoUpdateTime"`
	CreatorUserID   string     `json:"creator_user_id" csv:"creator_user_id" gorm:"size:37"`
	CreatorUserName string     `json:"creator_user_name" csv:"creator_user_name"`
	TemplateID      string     `json:"template_id" csv:"template_id" gorm:"size:100"`
	AuditID         string     `json:"audit_id" csv:"audit_id" gorm:"size:100"`
	AuditTitle      string     `json:"audit_title" csv:"audit_title"`
	AuditItemID     string     `json:"audit_item_id" csv:"audit_item_id" gorm:"size:100"`
	AuditItemLabel  string     `json:"audit_item_label" csv:"audit_item_label"`
	OrganisationID  string     `json:"organisation_id" csv:"organisation_id" gorm:"index:idx_act_modified_at;size:37"`
	CompletedAt     *time.Time `json:"completed_at" csv:"completed_at"`
	ActionLabel     string     `json:"action_label" csv:"action_label"`
	Deleted         bool       `json:"deleted" csv:"deleted"`
	AssetID         string     `json:"asset_id" csv:"asset_id" gorm:"size:36"`
	UniqueID        string     `json:"unique_id" csv:"unique_id"`
}

// ActionFeed is a representation of the actions feed
type ActionFeed struct {
	ModifiedAfter time.Time
	Incremental   bool
	Limit         int
}

// Name is the name of the feed
func (f *ActionFeed) Name() string {
	return "actions"
}

// HasRemainingInformation returns true if the feed returns remaining items information
func (f *ActionFeed) HasRemainingInformation() bool {
	return true
}

// Model returns the model of the feed row
func (f *ActionFeed) Model() interface{} {
	return Action{}
}

// RowsModel returns the model of feed rows
func (f *ActionFeed) RowsModel() interface{} {
	return &[]*Action{}
}

// PrimaryKey returns the primary key(s)
func (f *ActionFeed) PrimaryKey() []string {
	return []string{"action_id"}
}

// Columns returns the columns of the row
func (f *ActionFeed) Columns() []string {
	return []string{
		"title",
		"description",
		"site_id",
		"priority",
		"status",
		"due_date",
		"created_at",
		"modified_at",
		"exported_at",
		"creator_user_id",
		"creator_user_name",
		"template_id",
		"audit_id",
		"audit_title",
		"audit_item_id",
		"audit_item_label",
		"organisation_id",
		"completed_at",
		"action_label",
		"unique_id",
	}
}

// Order returns the ordering when retrieving an export
func (f *ActionFeed) Order() string {
	return "action_id"
}

// CreateSchema creates the schema of the feed for the supplied exporter
func (f *ActionFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*Action{})
}

// Export exports the feed to the supplied exporter
func (f *ActionFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
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

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*Action

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		if len(rows) != 0 {
			// Calculate the size of the batch we can insert into the DB at once. Column count + buffer to account for primary keys
			batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
			err := util.SplitSliceInBatch(batchSize, rows, func(batch []*Action) error {
				if err := exporter.WriteRows(f, batch); err != nil {
					return events.WrapEventError(err, "write rows")
				}
				return nil
			})

			if err != nil {
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
		InitialURL: "/feed/actions",
		Params: GetFeedParams{
			ModifiedAfter: f.ModifiedAfter,
			Limit:         f.Limit,
		},
	}

	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))
	}

	// Process Deleted Actions
	err = f.processDeletedActions(ctx, apiClient, exporter)
	if err != nil && events.IsBlockingError(err) {
		return events.WrapEventError(err, "process deleted actions")
	}

	return exporter.FinaliseExport(f, &[]*Action{})
}

func (f *ActionFeed) processDeletedActions(ctx context.Context, apiClient *httpapi.Client, exporter Exporter) error {
	lg := logger.GetLogger()
	dreq := httpapi.NewGetAccountsActivityLogRequest(f.Limit, f.ModifiedAfter, []string{"action.actions_deleted"})
	delFn := func(resp *httpapi.GetAccountsActivityLogResponse) error {
		var pkeys = make([]string, 0, len(resp.Activities))
		for _, act := range resp.Activities {
			for key, value := range act.Metadata {
				if strings.HasPrefix(key, "action_id") {
					pkeys = append(pkeys, value)
				}
			}
		}
		if len(pkeys) > 0 {
			rowsUpdated, err := exporter.UpdateRows(f, pkeys, map[string]interface{}{"deleted": true})
			if err != nil {
				return events.NewEventErrorWithMessage(err,
					events.ErrorSeverityWarning, events.ErrorSubSystemDB, false,
					"unable to update database records")
			}
			lg.Infof("there were %d rows marked as deleted", rowsUpdated)
		}

		return nil
	}
	return DrainAccountActivityHistoryLog(ctx, apiClient, dreq, delFn)
}
