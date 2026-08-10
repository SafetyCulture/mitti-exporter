package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MickStanciu/go-fn/fn"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
	"time"
)

type IssueAssignee struct {
	ID             string    `json:"id" csv:"id" gorm:"primarykey;column:id;size:375"`
	IssueID        string    `json:"issue_id" csv:"issue_id" gorm:"index:idx_issue_asg_issue_id;column:issue_id;size:36"`
	AssigneeID     string    `json:"assignee_id" csv:"assignee_id" gorm:"size:256"`
	Name           string    `json:"name" csv:"name"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	ModifiedAt     time.Time `json:"modified_at" csv:"modified_at" gorm:"index:idx_issue_asg_modified_at,sort:desc;column:modified_at"`
	Type           string    `json:"type" csv:"type"`
}

type IssueAssigneeFeed struct {
	Incremental bool
	Limit       int
}

func (f *IssueAssigneeFeed) Name() string {
	return "issue_assignees"
}

func (f *IssueAssigneeFeed) writeRows(exporter Exporter, rows []*IssueAssignee) error {
	batchSize := exporter.ParameterLimit() / (len(f.Columns()) + 4)
	err := util.SplitSliceInBatch(batchSize, rows, func(batch []*IssueAssignee) error {
		issueIDs := fn.Map(batch, func(a *IssueAssignee) string { return a.IssueID })

		if err := exporter.DeleteRowsIfExist(f, "issue_id IN (?)", issueIDs); err != nil {
			return fmt.Errorf("delete rows: %w", err)
		}

		if err := exporter.WriteRows(f, batch); err != nil {
			return events.WrapEventError(err, "write rows")
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// Export exports the feed to the supplied exporter
func (f *IssueAssigneeFeed) Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error {
	l := logger.GetLogger().With("feed", f.Name(), "org_id", orgID)
	status := GetExporterStatus()

	if err := exporter.InitFeed(f, &InitFeedOptions{
		// Delete data if incremental refresh is disabled so there is no duplicates
		Truncate: !f.Incremental,
	}); err != nil {
		return events.WrapEventError(err, "init feed")
	}

	drainFn := func(resp *GetFeedResponse) error {
		var rows []*IssueAssignee

		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "map data")
		}

		numRows := len(rows)
		if numRows != 0 {
			if err := f.writeRows(exporter, rows); err != nil {
				return err
			}
		}

		status.IncrementStatus(f.Name(), int64(numRows), apiClient.Duration.Milliseconds())

		l.With(
			"downloaded", status.ReadCounter(f.Name()),
			"duration_ms", apiClient.Duration.Milliseconds(),
			"export_duration_ms", exporter.GetDuration().Milliseconds(),
		).Info("export batch complete")
		return nil
	}

	req := &GetFeedRequest{
		InitialURL: "/feed/issue_assignees",
		Params: GetFeedParams{
			Limit: f.Limit,
		},
	}
	if err := DrainFeed(ctx, apiClient, req, drainFn); err != nil {
		return events.WrapEventError(err, fmt.Sprintf("feed %q", f.Name()))

	}
	return exporter.FinaliseExport(f, &[]IssueAssignee{})
}

func (f *IssueAssigneeFeed) HasRemainingInformation() bool {
	return false
}

func (f *IssueAssigneeFeed) Model() interface{} {
	return IssueAssignee{}
}

func (f *IssueAssigneeFeed) RowsModel() interface{} {
	return &[]*IssueAssignee{}
}

func (f *IssueAssigneeFeed) PrimaryKey() []string {
	return []string{"id"}
}

func (f *IssueAssigneeFeed) Columns() []string {
	return []string{
		"issue_id",
		"assignee_id",
		"type",
		"name",
		"organisation_id",
		"modified_at",
	}
}

func (f *IssueAssigneeFeed) Order() string {
	return "issue_id, assignee_id"
}

func (f *IssueAssigneeFeed) CreateSchema(exporter Exporter) error {
	return exporter.CreateSchema(f, &[]*IssueAssignee{})
}
