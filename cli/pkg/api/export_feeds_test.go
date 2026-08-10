package api_test

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func TestExporterFeedClient_ExportFeeds_should_create_all_schemas_to_file(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	cfg := &feed.ExporterFeedCfg{AccessToken: "token-123", ExportSiteIncludeDeleted: true}
	exporterApp := feed.NewExporterApp(nil, nil, cfg)
	err = exporterApp.ExportSchemas(exporter)
	assert.NoError(t, err)

	filesEqualish(t, "mocks/set_1/schemas/inspections.csv", filepath.Join(exporter.ExportPath, "inspections.csv"))
	filesEqualish(t, "mocks/set_1/schemas/inspection_items.csv", filepath.Join(exporter.ExportPath, "inspection_items.csv"))
	filesEqualish(t, "mocks/set_1/schemas/templates.csv", filepath.Join(exporter.ExportPath, "templates.csv"))
	filesEqualish(t, "mocks/set_1/schemas/template_permissions.csv", filepath.Join(exporter.ExportPath, "template_permissions.csv"))

	filesEqualish(t, "mocks/set_1/schemas/sites.csv", filepath.Join(exporter.ExportPath, "sites.csv"))
	filesEqualish(t, "mocks/set_1/schemas/site_members.csv", filepath.Join(exporter.ExportPath, "site_members.csv"))

	filesEqualish(t, "mocks/set_1/schemas/users.csv", filepath.Join(exporter.ExportPath, "users.csv"))
	filesEqualish(t, "mocks/set_1/schemas/groups.csv", filepath.Join(exporter.ExportPath, "groups.csv"))
	filesEqualish(t, "mocks/set_1/schemas/group_users.csv", filepath.Join(exporter.ExportPath, "group_users.csv"))

	filesEqualish(t, "mocks/set_1/schemas/schedules.csv", filepath.Join(exporter.ExportPath, "schedules.csv"))
	filesEqualish(t, "mocks/set_1/schemas/schedule_assignees.csv", filepath.Join(exporter.ExportPath, "schedule_assignees.csv"))
	filesEqualish(t, "mocks/set_1/schemas/schedule_occurrences.csv", filepath.Join(exporter.ExportPath, "schedule_occurrences.csv"))

	filesEqualish(t, "mocks/set_1/schemas/training_course_progresses.csv", filepath.Join(exporter.ExportPath, "training_course_progresses.csv"))
	filesEqualish(t, "mocks/set_1/schemas/issue_assignees.csv", filepath.Join(exporter.ExportPath, "issue_assignees.csv"))

	filesEqualish(t, "mocks/set_1/schemas/issue_assignees.csv", filepath.Join(exporter.ExportPath, "issue_assignees.csv"))
	filesEqualish(t, "mocks/set_1/schemas/account_histories.csv", filepath.Join(exporter.ExportPath, "account_histories.csv"))
}

func TestExporterFeedClient_ExportFeeds_should_export_all_feeds_to_file(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_bda3042f16a44249915ddc088adef92b",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Get("/SheqsyIntegrationApi/api/v3/companies/ada3042f-16a4-4249-915d-dc088adef92a").
		Reply(200).
		BodyString(`{
			"companyId": 4834,
			"companyName": "SafetyCulture",
			"name": null,
			"companyUId": "ada3042f-16a4-4249-915d-dc088adef92a",
			"externalId": null,
			"billingContactUId": null,
			"integratedSystems": 0,
			"taskTypes": [],
			"companyPlan": null,
			"createdDateTimeLocal": "0001-01-01T00:00:00",
			"employees": null,
			"users": null,
			"departments": null,
			"status": null,
			"ssoSettings": null
		}`)

	cfg := &feed.ExporterFeedCfg{
		AccessToken:              "token-123",
		ExportSiteIncludeDeleted: true,
		SheqsyUsername:           "token-123",
		SheqsyCompanyID:          "ada3042f-16a4-4249-915d-dc088adef92a",
	}
	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	filesEqualish(t, "mocks/set_1/outputs/inspections.csv", filepath.Join(exporter.ExportPath, "inspections.csv"))
	filesEqualish(t, "mocks/set_1/outputs/inspection_items.csv", filepath.Join(exporter.ExportPath, "inspection_items.csv"))
	filesEqualish(t, "mocks/set_1/outputs/templates.csv", filepath.Join(exporter.ExportPath, "templates.csv"))
	filesEqualish(t, "mocks/set_1/outputs/template_permissions.csv", filepath.Join(exporter.ExportPath, "template_permissions.csv"))

	filesEqualish(t, "mocks/set_1/outputs/sites.csv", filepath.Join(exporter.ExportPath, "sites.csv"))
	filesEqualish(t, "mocks/set_1/outputs/site_members.csv", filepath.Join(exporter.ExportPath, "site_members.csv"))

	filesEqualish(t, "mocks/set_1/outputs/users.csv", filepath.Join(exporter.ExportPath, "users.csv"))
	filesEqualish(t, "mocks/set_1/outputs/groups.csv", filepath.Join(exporter.ExportPath, "groups.csv"))
	filesEqualish(t, "mocks/set_1/outputs/group_users.csv", filepath.Join(exporter.ExportPath, "group_users.csv"))

	filesEqualish(t, "mocks/set_1/outputs/schedules.csv", filepath.Join(exporter.ExportPath, "schedules.csv"))
	filesEqualish(t, "mocks/set_1/outputs/schedule_assignees.csv", filepath.Join(exporter.ExportPath, "schedule_assignees.csv"))
	filesEqualish(t, "mocks/set_1/outputs/schedule_occurrences.csv", filepath.Join(exporter.ExportPath, "schedule_occurrences.csv"))

	filesEqualish(t, "mocks/set_1/outputs/actions.csv", filepath.Join(exporter.ExportPath, "actions.csv"))
	filesEqualish(t, "mocks/set_1/outputs/action_assignees.csv", filepath.Join(exporter.ExportPath, "action_assignees.csv"))

	filesEqualish(t, "mocks/set_1/outputs/sheqsy_employees.csv", filepath.Join(exporter.ExportPath, "sheqsy_employees.csv"))
	filesEqualish(t, "mocks/set_1/outputs/sheqsy_department_employees.csv", filepath.Join(exporter.ExportPath, "sheqsy_department_employees.csv"))
	filesEqualish(t, "mocks/set_1/outputs/sheqsy_shifts.csv", filepath.Join(exporter.ExportPath, "sheqsy_shifts.csv"))
	filesEqualish(t, "mocks/set_1/outputs/sheqsy_activities.csv", filepath.Join(exporter.ExportPath, "sheqsy_activities.csv"))
	filesEqualish(t, "mocks/set_1/outputs/sheqsy_departments.csv", filepath.Join(exporter.ExportPath, "sheqsy_departments.csv"))

	filesEqualish(t, "mocks/set_1/outputs/training_course_progresses.csv", filepath.Join(exporter.ExportPath, "training_course_progresses.csv"))
	filesEqualish(t, "mocks/set_1/outputs/account_histories.csv", filepath.Join(exporter.ExportPath, "account_histories.csv"))
}

func TestExporterFeedClient_ExportFeeds_should_err_when_not_auth(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryCSVExporter()
	require.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(401).
		BodyString(`
		{
			"statusCode": 401,
			"error": "Unauthorized",
			"message": "Bad token or token expired"
		}
		`)

	cfg := &feed.ExporterFeedCfg{
		AccessToken: "token-123",
	}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.EqualError(t, err, "get details of the current user: api request: request error status: 401")
}

func TestExporterFeedClient_ExportFeeds_should_err_when_InitFeed_errors(t *testing.T) {
	defer gock.Off()

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_bda3042f16a44249915ddc088adef92b",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	exporter := getMockedExporter()
	exporter.
		On("InitFeed", mock.Anything, mock.Anything).
		Return(events.NewEventError(fmt.Errorf("unable to truncate table"), events.ErrorSeverityError, events.ErrorSubSystemDB, true))

	cfg := &feed.ExporterFeedCfg{
		AccessToken:  "token-123",
		ExportTables: []string{"users"},
	}

	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err := exporterApp.ExportFeeds(exporter, context.Background())
	ee, ok := err.(*events.EventError)
	require.True(t, ok)
	assert.True(t, ee.IsError())
	assert.True(t, ee.IsFatal())
	assert.EqualValues(t, "init feed: unable to truncate table", ee.Error())
}

func TestExporterFeedClient_ExportFeeds_should_err_when_cannot_unmarshal(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryCSVExporter()
	require.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_bda3042f16a44249915ddc088adef92b",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Get("/feed/inspections").
		Reply(200).
		File("mocks/set_1/feed_inspections_1.json")

	gock.New("http://localhost:9999").
		Get("/feed/inspections/2").
		Reply(200).
		File("mocks/set_1/feed_inspections_2.json")

	gock.New("http://localhost:9999").
		Get("/feed/users").
		Reply(200).
		File("mocks/misc/feed_users_bad_format.json")

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"0001-01-01T00:00:00Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_1", "inspections_deleted_single_page.json"))

	cfg := &feed.ExporterFeedCfg{
		AccessToken:  "token-123",
		ExportTables: []string{"inspections", "users"},
	}
	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.EqualError(t, err, `feed "users": map data: unexpected end of JSON input`)
}

func TestExporterFeedClient_ExportFeeds_should_err_when_cannot_write_rows(t *testing.T) {
	defer gock.Off()

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_bda3042f16a44249915ddc088adef92b",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Get("/feed/users").
		Reply(200).
		File("mocks/set_1/feed_users_1.json")

	cfg := &feed.ExporterFeedCfg{
		AccessToken:  "token-123",
		ExportTables: []string{"users"},
	}
	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)

	exporter := getMockedExporter()
	exporter.On("InitFeed", mock.Anything, mock.Anything).Return(nil)
	exporter.On("WriteRows", mock.Anything, mock.Anything).Return(fmt.Errorf("cannot write rows"))

	err := exporterApp.ExportFeeds(exporter, context.Background())
	assert.EqualError(t, err, `feed "users": write rows: cannot write rows`)
}

// The expectation of this test is that group_users and schedule_assignees are truncated and refreshed
// and that other tables are incrementally updated
func TestExporterFeedClient_ExportFeeds_should_perform_incremental_update_on_second_run(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_bda3042f16a44249915ddc088adef92b",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"0001-01-01T00:00:00Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_2", "inspections_deleted_single_page.json"))

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"2014-03-17T00:35:40Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_2", "inspections_deleted_single_page.json"))

	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	cfg := &feed.ExporterFeedCfg{
		AccessToken:              "token-123",
		ExportSiteIncludeDeleted: true,
	}

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	initMockFeedsSet2(apiClient.HTTPClient())
	exporterApp = feed.NewExporterApp(apiClient, nil, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	filesEqualish(t, "mocks/set_2/outputs/inspections.csv", filepath.Join(exporter.ExportPath, "inspections.csv"))
	filesEqualish(t, "mocks/set_2/outputs/inspection_items.csv", filepath.Join(exporter.ExportPath, "inspection_items.csv"))
	filesEqualish(t, "mocks/set_2/outputs/templates.csv", filepath.Join(exporter.ExportPath, "templates.csv"))
	filesEqualish(t, "mocks/set_2/outputs/template_permissions.csv", filepath.Join(exporter.ExportPath, "template_permissions.csv"))

	filesEqualish(t, "mocks/set_2/outputs/sites.csv", filepath.Join(exporter.ExportPath, "sites.csv"))
	filesEqualish(t, "mocks/set_2/outputs/site_members.csv", filepath.Join(exporter.ExportPath, "site_members.csv"))

	filesEqualish(t, "mocks/set_2/outputs/users.csv", filepath.Join(exporter.ExportPath, "users.csv"))
	filesEqualish(t, "mocks/set_2/outputs/groups.csv", filepath.Join(exporter.ExportPath, "groups.csv"))

	filesEqualish(t, "mocks/set_2/outputs/group_users.csv", filepath.Join(exporter.ExportPath, "group_users.csv"))

	filesEqualish(t, "mocks/set_2/outputs/schedules.csv", filepath.Join(exporter.ExportPath, "schedules.csv"))
	filesEqualish(t, "mocks/set_2/outputs/schedule_assignees.csv", filepath.Join(exporter.ExportPath, "schedule_assignees.csv"))
	filesEqualish(t, "mocks/set_2/outputs/schedule_occurrences.csv", filepath.Join(exporter.ExportPath, "schedule_occurrences.csv"))

	filesEqualish(t, "mocks/set_2/outputs/actions.csv", filepath.Join(exporter.ExportPath, "actions.csv"))
	filesEqualish(t, "mocks/set_2/outputs/action_assignees.csv", filepath.Join(exporter.ExportPath, "action_assignees.csv"))
}

func TestExporterFeedClient_ExportFeeds_should_handle_lots_of_rows_ok(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet3(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"0001-01-01T00:00:00Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_3", "inspections_deleted_single_page.json"))

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_bda3042f16a44249915ddc088adef92b",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	cfg := &feed.ExporterFeedCfg{
		AccessToken:              "token-123",
		ExportSiteIncludeDeleted: true,
	}

	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	inspectionsLines, err := countFileLines(filepath.Join(exporter.ExportPath, "inspections.csv"))
	assert.NoError(t, err)
	assert.Equal(t, 97, inspectionsLines)

	inspectionItemsLines, err := countFileLines(filepath.Join(exporter.ExportPath, "inspection_items.csv"))
	assert.NoError(t, err)
	assert.Equal(t, 230, inspectionItemsLines)
}

func getMockedExporter() *mocks.Exporter {
	exporter := &mocks.Exporter{}
	exporter.On("SupportsUpsert").Return(true)
	exporter.On("ParameterLimit").Return(100)
	return exporter
}
