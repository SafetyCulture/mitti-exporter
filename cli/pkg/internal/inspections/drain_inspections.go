package inspections

import (
	"context"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
	"go.uber.org/zap"
)

// DrainInspections fetches the inspections in batches and triggers the callback for each batch.
func DrainInspections(ctx context.Context, apiClient *httpapi.Client, params *httpapi.ListInspectionsParams, callback func(*httpapi.ListInspectionsResponse, *zap.SugaredLogger) error) error {
	l := logger.GetLogger().With("type", "inspection-json")
	modifiedAfter := params.ModifiedAfter

	for {
		resp, err := httpapi.ListInspections(
			ctx,
			apiClient,
			&httpapi.ListInspectionsParams{
				ModifiedAfter: modifiedAfter,
				TemplateIDs:   params.TemplateIDs,
				Archived:      params.Archived,
				Completed:     params.Completed,
				Limit:         100,
			},
		)
		if err != nil {
			return err
		}

		if err := callback(resp, l); err != nil {
			return err
		}

		if (resp.Total - resp.Count) == 0 {
			break
		}
		modifiedAfter = resp.Inspections[resp.Count-1].ModifiedAt
	}

	return nil
}
