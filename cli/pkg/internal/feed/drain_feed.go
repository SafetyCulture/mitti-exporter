package feed

import (
	"context"
	"encoding/json"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

// DrainFeed fetches the data in batches and triggers the callback for each batch.
func DrainFeed(ctx context.Context, apiClient *httpapi.Client, request *GetFeedRequest, feedFn func(*GetFeedResponse) error) error {
	var nextURL string
	// Used to both ensure the fetchFn is called at least once
	first := true
	for nextURL != "" || first {
		first = false
		execURL := request.InitialURL
		execParams := &request.Params

		if nextURL != "" {
			execURL = nextURL
			execParams = nil
		}

		resp, httpErr := httpapi.ExecuteGet[GetFeedResponse](ctx, apiClient, execURL, execParams)
		if httpErr != nil {
			return events.NewEventError(httpErr, events.ErrorSeverityError, events.ErrorSubSystemAPI, false)
		}
		nextURL = resp.Metadata.NextPage

		err := feedFn(resp)
		if err != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemAPI, false)
		}
	}

	return nil
}

// GetFeedRequest has all the data needed to make a request to get a feed
type GetFeedRequest struct {
	URL        string
	InitialURL string
	Params     GetFeedParams
}

// GetFeedResponse is a representation of the data returned when fetching a feed
type GetFeedResponse struct {
	Metadata PageFeedMetadata `json:"metadata"`

	Data json.RawMessage `json:"data"`
}

// PageFeedMetadata is a representation of the metadata returned when fetching a feed
type PageFeedMetadata struct {
	NextPage         string `json:"next_page"`
	RemainingRecords int64  `json:"remaining_records"`
}

// GetFeedParams is a list of all parameters we can set when fetching a feed
type GetFeedParams struct {
	ModifiedAfter   time.Time `url:"modified_after,omitempty"`
	ModifiedBefore  time.Time `url:"modified_before,omitempty"`
	TemplateIDs     []string  `url:"template,omitempty"`
	Archived        string    `url:"archived,omitempty"`
	Completed       string    `url:"completed,omitempty"`
	IncludeInactive bool      `url:"include_inactive,omitempty"`
	Limit           int       `url:"limit,omitempty"`
	WebReportLink   string    `url:"web_report_link,omitempty"`
	StartDate       time.Time `url:"start_date,omitempty"`
	EndDate         time.Time `url:"end_date,omitempty"`

	// Applicable only for sites
	IncludeDeleted    bool  `url:"include_deleted,omitempty"`
	ShowOnlyLeafNodes *bool `url:"show_only_leaf_nodes,omitempty"`

	// Applicable only for course progress
	Offset           int    `url:"offset,omitempty"`
	CompletionStatus string `url:"completion_status,omitempty"`

	// Applicable only for account history
	CreatedAfter time.Time `url:"created_after,omitempty"`
}
