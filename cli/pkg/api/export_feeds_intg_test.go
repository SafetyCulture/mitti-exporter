//go:build sql
// +build sql

package api_test

import (
	"context"
	"net/http"
	"path"
	"path/filepath"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/stretchr/testify/assert"
)

/*
	For these tests we use the CSV Exporter, but instead of using a SQLite DB as an intermediary layer
	we write the data to a real DB. So we get to test the SQL exporting logic, and compare the results easily.
*/

func TestIntegrationDbCreateSchema_should_create_all_schemas(t *testing.T) {
	sqlExporter, err := getTestingSQLExporter()
	require.NoError(t, err)
	exporter, err := getTemporaryCSVExporterWithRealSQLExporter(sqlExporter)
	require.NoError(t, err)

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	cfg := &feed.ExporterFeedCfg{
		AccessToken: "token-123",
	}

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

	filesEqualish(t, "mocks/set_1/schemas/issues.csv", filepath.Join(exporter.ExportPath, "issues.csv"))
}

func TestIntegrationDbExportFeeds_should_export_all_feeds_to_file(t *testing.T) {
	sqlExporter, err := getTestingSQLExporter()
	require.NoError(t, err)
	exporter, err := getTemporaryCSVExporterWithRealSQLExporter(sqlExporter)
	require.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	cfg := &feed.ExporterFeedCfg{
		AccessToken: "token-123",
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

	filesEqualish(t, "mocks/set_1/outputs/issues.csv", filepath.Join(exporter.ExportPath, "issues.csv"))
}

// Expectation of this test is that group_users and schedule_assignees are truncated and refreshed
// and that other tables are incrementally updated
func TestIntegrationDbExportFeeds_should_perform_incremental_update_on_second_run(t *testing.T) {
	sqlExporter, err := getTestingSQLExporter()
	require.NoError(t, err)
	exporter, err := getTemporaryCSVExporterWithRealSQLExporter(sqlExporter)
	require.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_2", "inspections_deleted_single_page.json"))

	cfg := &feed.ExporterFeedCfg{
		AccessToken:       "token-123",
		ExportIncremental: true,
	}

	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	initMockFeedsSet2(apiClient.HTTPClient())

	exporterApp = feed.NewExporterApp(apiClient, apiClient, cfg)
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

	filesEqualish(t, "mocks/set_2/outputs/issues.csv", filepath.Join(exporter.ExportPath, "issues.csv"))
}

func TestIntegrationDbExportFeeds_should_handle_lots_of_rows_ok(t *testing.T) {
	sqlExporter, err := getTestingSQLExporter()
	require.NoError(t, err)
	exporter, err := getTemporaryCSVExporterWithRealSQLExporter(sqlExporter)
	require.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet3(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"0001-01-01T00:00:00Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		BodyString(`{"activites": []}`)

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	cfg := &feed.ExporterFeedCfg{
		AccessToken:       "token-123",
		ExportIncremental: true,
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

func TestIntegrationDbExportFeeds_should_update_action_assignees_on_second_run(t *testing.T) {
	sqlExporter, err := getTestingSQLExporter()
	require.NoError(t, err)
	exporter, err := getTemporaryCSVExporterWithRealSQLExporter(sqlExporter)
	require.NoError(t, err)

	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())
	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"2014-03-17T11:35:40+11:00"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_1", "inspections_deleted_single_page.json"))

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"2014-03-17T00:35:40Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_1", "inspections_deleted_single_page.json"))

	cfg := &feed.ExporterFeedCfg{
		AccessToken:       "token-123",
		ExportIncremental: true,
	}

	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	filesEqualish(t, "mocks/set_1/outputs/action_assignees.csv", filepath.Join(exporter.ExportPath, "action_assignees.csv"))

	initMockFeedsSet2(apiClient.HTTPClient())

	exporterApp = feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)
	filesEqualish(t, "mocks/set_2/outputs/action_assignees.csv", filepath.Join(exporter.ExportPath, "action_assignees.csv"))
}

func TestGroupUserFeed_Export_should_filter_duplicates(t *testing.T) {
	sqlExporter, err := getTestingSQLExporter()
	require.Nil(t, err)
	require.NotNil(t, sqlExporter)

	exporter, err := getTemporaryCSVExporterWithRealSQLExporter(sqlExporter)
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())
	gock.New("http://localhost:9999").
		Get("/accounts/user/v1/user:WhoAmI").
		Times(2).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_ada3042f16a44249915ddc088adef92a",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)
	gock.New("http://localhost:9999").
		Get("/feed/group_users").
		Times(1).
		Reply(200).
		File("mocks/set_5/feed_group_users_1.json")

	cfg := &feed.ExporterFeedCfg{
		AccessToken:       "token-123",
		ExportIncremental: true,
		ExportTables:      []string{"group_users"},
	}

	exporterApp := feed.NewExporterApp(apiClient, apiClient, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)

	lines, err := countFileLines(filepath.Join(exporter.ExportPath, "group_users.csv"))
	assert.NoError(t, err)
	assert.Equal(t, 5, lines)
}
