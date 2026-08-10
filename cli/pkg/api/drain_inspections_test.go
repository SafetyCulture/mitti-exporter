package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/inspections"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gopkg.in/h2non/gock.v1"
)

func TestAPIClientDrainInspections_should_return_for_as_long_next_page_set(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/search").
		Reply(200).
		BodyString(`{
			"count": 2,
			"total": 2,
			"audits": [
				{
					"audit_id": "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
				},
				{
					"audit_id": "audit_1743ae1aaa8741e6a23db83300e56efe"
				}
			]
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	var auditIDs []string
	err := inspections.DrainInspections(
		context.Background(),
		apiClient,
		&httpapi.ListInspectionsParams{},
		func(data *httpapi.ListInspectionsResponse, log *zap.SugaredLogger) error {
			for _, inspection := range data.Inspections {
				auditIDs = append(auditIDs, inspection.ID)
			}
			return nil
		})
	assert.NoError(t, err)

	assert.Equal(t, []string{
		"audit_8E2B1F3CB9C94D8792957F9F99E2E4BD",
		"audit_1743ae1aaa8741e6a23db83300e56efe",
	}, auditIDs)
}

func TestDrainInspectionsWithAPIError(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/search").
		ReplyError(fmt.Errorf("test error"))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	err := inspections.DrainInspections(
		context.Background(),
		apiClient,
		&httpapi.ListInspectionsParams{},
		func(data *httpapi.ListInspectionsResponse, log *zap.SugaredLogger) error {
			return nil
		})
	assert.NotNil(t, err)
}

func TestDrainInspectionsWithCallbackError(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/search").
		Reply(200).
		BodyString(`{
			"count": 2,
			"total": 2,
			"audits": [
				{
					"audit_id": "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
				},
				{
					"audit_id": "audit_1743ae1aaa8741e6a23db83300e56efe"
				}
			]
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	err := inspections.DrainInspections(
		context.Background(),
		apiClient,
		&httpapi.ListInspectionsParams{},
		func(data *httpapi.ListInspectionsResponse, log *zap.SugaredLogger) error {
			return fmt.Errorf("test error")
		})
	assert.NotNil(t, err)
}
