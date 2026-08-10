package feed

import (
	"context"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
)

// DrainAccountActivityHistoryLog cycle through GetAccountsActivityLogResponse and adapts the filter while there is a next page
func DrainAccountActivityHistoryLog(ctx context.Context, apiClient *httpapi.Client, req *httpapi.GetAccountsActivityLogRequestParams, feedFn func(*httpapi.GetAccountsActivityLogResponse) error) error {
	for {
		res, err := httpapi.ListOrganisationActivityLog(ctx, apiClient, req)
		if err != nil {
			return events.NewEventErrorWithMessage(err,
				events.ErrorSeverityWarning, events.ErrorSubSystemAPI, false,
				"unable to access Accounts Activity Logs")
		}

		err = feedFn(res)
		if err != nil {
			return err
		}

		if res.NextPageToken != "" {
			req.PageToken = res.NextPageToken
		} else {
			break
		}
	}
	return nil
}
