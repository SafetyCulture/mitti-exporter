package api_test

import (
	"context"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

// newFeedFixtures describes a single-row response for each of the feeds added from the data feeds reference
var newFeedFixtures = []struct {
	table string
	path  string
	body  string
}{
	{"new_schedules", "/scheduling/v1/feed/schedules", `{"metadata":{},"data":[{"id":"scheduleitem_1","title":"Weekly","modified_at":"2026-01-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z","site_ids":["site_1"],"asset_ids":null}]}`},
	{"new_schedule_assignees", "/scheduling/v1/feed/schedule_assignees", `{"metadata":{},"data":[{"id":"scheduleitem_1_user_1","schedule_id":"scheduleitem_1","assignee_id":"user_1","type":"user"}]}`},
	{"new_schedule_occurrences", "/scheduling/v1/feed/schedule_occurrences", `{"metadata":{},"data":[{"id":"scheduleitem_1_occurrence_1_user_1","schedule_id":"scheduleitem_1","occurrence_id":"occurrence_1","occurrence_status":"WONT_DO","wont_do_note":"closed"}]}`},
	{"action_fields", "/tasks/v1/feed/action_fields", `{"metadata":{"remaining_records":0},"data":[{"action_id":"a1","field_id":"f1","field_type":"text","field_value":"v"}]}`},
	{"issue_relations", "/feed/issue_relations", `{"metadata":{},"data":[{"id":"r1","from_id":"i1","rel_type":"ISSUE-INSPECTION","to_id":"audit_1"}]}`},
	{"investigations", "/incidents/v1/feed/investigations", `{"metadata":{},"data":[{"investigation_id":"inv1","title":"t","status_badge_color":3,"identifier_sequence":7}]}`},
	{"investigation_fields", "/incidents/v1/feed/investigation_fields", `{"metadata":{},"data":[{"investigation_id":"inv1","field_id":"f1","field_instance_id":"fi1","content":"md"}]}`},
	{"investigation_detail_fields", "/incidents/v1/feed/investigation_detail_fields", `{"metadata":{},"data":[{"investigation_id":"inv1","detail_field_id":"d1","detail_field_instance_id":"di1","multi_select_content":"[\"a\"]"}]}`},
	{"investigation_relationships", "/incidents/v1/feed/investigation_relationships", `{"metadata":{},"data":[{"investigation_id":"inv1","resource_type":"issue","resource_id":"i1"}]}`},
	{"contractor_companies", "/companies/v1/feed/companies", `{"metadata":{},"data":[{"company_id":"c1","company_name":"Acme"}]}`},
	{"contractor_company_documents", "/companies/v1/feed/companies/documents", `{"metadata":{},"data":[{"company_id":"c1","document_id":"d1","document_version":"1"}]}`},
	{"contractor_company_members", "/companies/v1/feed/companies/members", `{"metadata":{},"data":[{"company_id":"c1","user_id":"u1","user_role":"admin"}]}`},
	{"maintenance_programs", "/assets/v1/feed/maintenance_programs", `{"metadata":{},"data":[{"id":"p1","name":"n","assets_count":2,"plans_count":1}]}`},
	{"maintenance_plans", "/assets/v1/feed/maintenance_plans", `{"metadata":{},"data":[{"id":"pl1","program_id":"p1"}]}`},
	{"assets_maintenance", "/assets/v1/feed/assets_maintenance", `{"metadata":{},"data":[{"asset_id":"as1","plan_id":"pl1","service_status":"due"}]}`},
	{"credential_types", "/credentials/v1/feed/credential-types", `{"metadata":{},"data":[{"document_type_id":"t1","document_type_name":"Licence"}]}`},
	{"credentials", "/credentials/v1/feed/credentials", `{"metadata":{},"data":[{"document_id":"doc1","subject_user_id":"u1","expiry_status":"EXPIRY_STATUS_VALID","deleted":false}]}`},
}

func TestNewFeeds_should_export_rows_to_sql_db(t *testing.T) {
	defer gock.Off()

	cfg := &feed.ExporterFeedCfg{}
	byName := map[string]feed.Feed{}
	for _, f := range feed.NewExporterApp(nil, nil, cfg).GetFeeds() {
		byName[f.Name()] = f
	}

	for _, fixture := range newFeedFixtures {
		t.Run(fixture.table, func(t *testing.T) {
			exporter, err := getInmemorySQLExporter("")
			assert.NoError(t, err)

			apiClient := GetTestClient()
			gock.InterceptClient(apiClient.HTTPClient())
			gock.New("http://localhost:9999").Get(fixture.path).Reply(200).BodyString(fixture.body)

			f, ok := byName[fixture.table]
			assert.True(t, ok, "feed should be registered")

			assert.NoError(t, f.Export(context.Background(), apiClient, exporter, ""))

			var rows []map[string]interface{}
			assert.NoError(t, exporter.DB.Table(fixture.table).Scan(&rows).Error)
			assert.Len(t, rows, 1)
		})
	}
}
