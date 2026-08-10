package report

import (
	"context"
	"fmt"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
)

// InitiateInspectionReportExport export the report of the given auditID.
func InitiateInspectionReportExport(ctx context.Context, apiClient *httpapi.Client, auditID string, format string, preferenceID string) (string, error) {
	url := fmt.Sprintf("audits/%s/report", auditID)
	body := &initiateInspectionReportExportRequest{
		Format:       format,
		PreferenceID: preferenceID,
	}

	result, err := httpapi.ExecutePost[initiateInspectionReportExportResponse](ctx, apiClient, url, body)
	if err != nil {
		return "", err
	}

	return result.MessageID, nil
}

type initiateInspectionReportExportRequest struct {
	Format       string `json:"format"`
	PreferenceID string `json:"preference_id,omitempty"`
}

type initiateInspectionReportExportResponse struct {
	MessageID string `json:"messageId"`
}
