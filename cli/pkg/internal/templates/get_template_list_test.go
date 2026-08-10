package templates_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/templates"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestClient_GetTemplateList_ShouldNotPaginate(t *testing.T) {
	defer gock.Off()
	//gock.Observe(gock.DumpRequest)

	apiClient := getTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "1000").
		Reply(200).
		File("fixtures/single_page_success.json")

	templatesClient := templates.NewTemplatesClient(apiClient)
	res := templatesClient.GetTemplateList(context.Background(), 1000)
	assert.EqualValues(t, 7, len(res))
}

func TestClient_GetTemplateList_ShouldPaginate(t *testing.T) {
	defer gock.Off()
	//gock.Observe(gock.DumpRequest)

	apiClient := getTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "3").
		Reply(200).
		File("fixtures/page_1_of_3.json")

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "3").
		MatchParam("page_token", "Mw==").
		Reply(200).
		File("fixtures/page_2_of_3.json")

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "3").
		MatchParam("page_token", "Ng==").
		Reply(200).
		File("fixtures/page_3_of_3.json")

	templatesClient := templates.NewTemplatesClient(apiClient)
	res := templatesClient.GetTemplateList(context.Background(), 3)
	assert.EqualValues(t, 7, len(res))
}

func TestClient_GetTemplateList_WhenApiError(t *testing.T) {
	defer gock.Off()
	//gock.Observe(gock.DumpRequest)

	apiClient := getTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "3").
		Reply(200).
		File("fixtures/page_1_of_3.json")

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "3").
		MatchParam("page_token", "Mw==").
		ReplyError(fmt.Errorf("test error"))

	templatesClient := templates.NewTemplatesClient(apiClient)
	res := templatesClient.GetTemplateList(context.Background(), 3)
	assert.EqualValues(t, 3, len(res))
}

// getTestClient creates a new test apiClient
func getTestClient(opts ...httpapi.Opt) *httpapi.Client {
	cfg := httpapi.ClientCfg{
		Addr:                "http://localhost:9999",
		AuthorizationHeader: "abc123",
		IntegrationID:       "test",
		IntegrationVersion:  "dev",
	}
	apiClient := httpapi.NewClient(&cfg, opts...)
	apiClient.RetryWaitMin = 10 * time.Millisecond
	apiClient.RetryWaitMax = 10 * time.Millisecond
	apiClient.CheckForRetry = httpapi.DefaultRetryPolicy
	apiClient.RetryMax = 1
	return apiClient
}
