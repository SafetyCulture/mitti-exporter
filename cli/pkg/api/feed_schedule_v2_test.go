package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestScheduleV2Feed_should_follow_pagination(t *testing.T) {
	defer gock.Off()
	// drop the mocks that other tests left behind
	gock.Flush()

	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())
	gock.New("http://localhost:9999").Get("/scheduling/v1/feed/schedules").
		MatchParam("limit", "1000").
		Reply(200).
		BodyString(`{"metadata":{"next_page":"/scheduling/v1/feed/schedules/page2","next_page_token":"t"},"data":[{"id":"scheduleitem_1","title":"One","modified_at":"2026-01-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z","site_ids":["site_1","site_2"],"assignees":["user_1"],"ending_rule_end_after_repeats":5}]}`)
	gock.New("http://localhost:9999").Get("/scheduling/v1/feed/schedules/page2").
		Reply(200).
		BodyString(`{"metadata":{},"data":[{"id":"scheduleitem_2","title":"Two","modified_at":"2026-01-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z"}]}`)

	assert.NoError(t, feed.NewScheduleV2Feed(nil, nil).Export(context.Background(), apiClient, exporter, ""))

	var rows []feed.ScheduleV2
	assert.NoError(t, exporter.DB.Table("new_schedules").Order("schedule_id").Scan(&rows).Error)
	assert.Len(t, rows, 2)
	assert.Equal(t, []string{"One", "Two"}, []string{rows[0].Description, rows[1].Description})
}

func TestSchedulingV2Feeds_should_send_the_status_and_date_filters(t *testing.T) {
	defer gock.Off()
	// drop the mocks that other tests left behind
	gock.Flush()

	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())
	for _, path := range []string{"/scheduling/v1/feed/schedules", "/scheduling/v1/feed/schedule_assignees"} {
		gock.New("http://localhost:9999").Get(path).
			MatchParam("limit", "1000").
			MatchParam("show_active", "true").
			MatchParam("show_paused", "false").
			MatchParam("show_finished", "true").
			Reply(200).
			BodyString(`{"metadata":{},"data":[]}`)
	}
	gock.New("http://localhost:9999").Get("/scheduling/v1/feed/schedule_occurrences").
		MatchParam("include_entity_names", "true").
		MatchParam("start_date", "2026-01-01").
		MatchParam("end_date", "2026-03-31").
		Reply(200).
		BodyString(`{"metadata":{},"data":[]}`)

	statuses := []string{"active", "FINISHED"}
	start, end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	assert.NoError(t, feed.NewScheduleV2Feed(nil, statuses).Export(context.Background(), apiClient, exporter, ""))
	assert.NoError(t, feed.NewScheduleAssigneeV2Feed(nil, statuses).Export(context.Background(), apiClient, exporter, ""))
	assert.NoError(t, feed.NewScheduleOccurrenceV2Feed(nil, start, end).Export(context.Background(), apiClient, exporter, ""))
	assert.True(t, gock.IsDone())
}

func TestSchedulingV2Feeds_should_export_every_column_and_a_row_per_target(t *testing.T) {
	defer gock.Off()
	// drop the mocks that other tests left behind
	gock.Flush()

	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())
	gock.New("http://localhost:9999").Get("/scheduling/v1/feed/schedules").
		Reply(200).
		BodyString(`{"metadata":{},"data":[{"id":"scheduleitem_1","title":"Daily check","recurrence":"DTSTART;TZID=Australia/Sydney:20260928T093000\nRRULE:FREQ=DAILY","duration":"PT8H","modified_at":"2026-09-28T05:59:08Z","organisation_id":"role_org","created_at":"2026-09-28T05:59:08Z","completion_rule":"all","status":"ACTIVE","timezone":"Australia/Sydney","timezone_type":"CUSTOM_TIMEZONE","is_late_submit_allowed":true,"site_ids":["location_75fc77d9a3b14c5e9f0e1a2b3c4d5e6f"],"template_id":"template_1","created_by":"user_1","grace_period_duration":"until_next","site_based_assignment_enabled":true,"ending_rule_type":"end_date","ending_rule_end_date":"2026-12-31T00:00:00Z","assignees":["role_1"],"intersections":[],"schedule_type":"SCHEDULE_TYPE_SITES","asset_ids":[]}]}`)
	gock.New("http://localhost:9999").Get("/scheduling/v1/feed/schedule_assignees").
		Reply(200).
		BodyString(`{"metadata":{},"data":[
			{"id":"scheduleitem_1_role_1","schedule_id":"scheduleitem_1","assignee_id":"role_1","type":"role","name":"Cleaners","organisation_id":"role_org","completion_rule":"all","target_id":"site_1","target_type":"SITE"},
			{"id":"scheduleitem_1_role_1","schedule_id":"scheduleitem_1","assignee_id":"role_1","type":"role","name":"Cleaners","organisation_id":"role_org","completion_rule":"all","target_id":"site_2","target_type":"SITE"}
		]}`)
	gock.New("http://localhost:9999").Get("/scheduling/v1/feed/schedule_occurrences").
		MatchParam("include_entity_names", "true").
		Reply(200).
		BodyString(`{"metadata":{},"data":[
			{"id":"scheduleitem_1_occurrence_1_user_1","schedule_id":"scheduleitem_1","occurrence_id":"1","template_id":"template_1","start_time":"2026-09-28T09:30:00Z","due_time":"2026-09-28T17:30:00Z","miss_time":"2026-09-28T13:30:00.123Z","occurrence_status":"COMPLETED","audit_id":"audit_1","completed_at":"2026-09-28T10:00:00Z","completion_rule":"all","organisation_id":"role_org","assignee_id":"user_1","assignee_from":"location_1+role_1","target_id":"site_1","target_type":"SITE","assignee_status":"COMPLETED","completed_by_user_id":"user_2","assignee_name":"Ada Lovelace","assignee_from_names":["Sydney","Cleaners"],"completed_by_user_name":"Alan Turing","target_name":"Sydney"},
			{"id":"scheduleitem_1_occurrence_1_user_1","schedule_id":"scheduleitem_1","occurrence_id":"1","template_id":"template_1","start_time":"2026-09-28T09:30:00Z","due_time":"2026-09-28T17:30:00Z","miss_time":null,"occurrence_status":"WONT_DO","completion_rule":"all","organisation_id":"role_org","wont_do_note":"closed","assignee_id":"user_1","assignee_from":"role_1","target_id":"site_2","target_type":"SITE","assignee_status":"WONT_DO","assignee_name":"Ada Lovelace","assignee_from_names":["Cleaners"],"completed_by_user_name":null,"target_name":"Melbourne"}
		]}`)

	assert.NoError(t, feed.NewScheduleV2Feed(nil, nil).Export(context.Background(), apiClient, exporter, ""))
	assert.NoError(t, feed.NewScheduleAssigneeV2Feed(nil, nil).Export(context.Background(), apiClient, exporter, ""))
	assert.NoError(t, feed.NewScheduleOccurrenceV2Feed(nil, time.Time{}, time.Time{}).Export(context.Background(), apiClient, exporter, ""))

	var schedules []feed.ScheduleV2
	assert.NoError(t, exporter.DB.Table("new_schedules").Scan(&schedules).Error)
	assert.Len(t, schedules, 1)
	schedule := schedules[0]
	sydney, err := time.LoadLocation("Australia/Sydney")
	assert.NoError(t, err)
	assert.True(t, time.Date(2026, 9, 28, 9, 30, 0, 0, sydney).Equal(*schedule.FromDate))
	_, fromDateOffset := schedule.FromDate.Zone()
	assert.Equal(t, 0, fromDateOffset, "from_date should be in UTC")
	assert.True(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC).Equal(*schedule.ToDate))
	assert.False(t, schedule.ExportedAt.IsZero())
	schedule.FromDate, schedule.ToDate, schedule.ExportedAt = nil, nil, time.Time{}
	schedule.ModifiedAt = schedule.ModifiedAt.UTC()
	assert.Equal(t, feed.ScheduleV2{
		ID:              "scheduleitem_1",
		Description:     "Daily check",
		Recurrence:      "DTSTART;TZID=Australia/Sydney:20260928T093000\nRRULE:FREQ=DAILY",
		Duration:        "PT8H",
		ModifiedAt:      time.Date(2026, 9, 28, 5, 59, 8, 0, time.UTC),
		StartTimeHour:   9,
		StartTimeMinute: 30,
		AllMustComplete: true,
		Status:          "ACTIVE",
		OrganisationID:  "role_org",
		Timezone:        "Australia/Sydney",
		TimezoneType:    "CUSTOM_TIMEZONE",
		CanLateSubmit:   true,
		SiteID:          "location_75fc77d9a3b14c5e9f0e1a2b3c4d5e6f",
		TemplateID:      "template_1",
		CreatorUserID:   "user_1",
		TargetType:      "SITE",
		TargetIDs:       `["75fc77d9-a3b1-4c5e-9f0e-1a2b3c4d5e6f"]`,
	}, schedule)

	var assignees []feed.ScheduleAssigneeV2
	assert.NoError(t, exporter.DB.Table("new_schedule_assignees").Order("target_id").Scan(&assignees).Error)
	for i := range assignees {
		assert.False(t, assignees[i].ExportedAt.IsZero())
		assignees[i].ExportedAt = time.Time{}
	}
	assigneeOf := func(targetID string) feed.ScheduleAssigneeV2 {
		return feed.ScheduleAssigneeV2{
			ID: "scheduleitem_1_role_1", ScheduleID: "scheduleitem_1", AssigneeID: "role_1", OrganisationID: "role_org",
			Type: "role", Name: "Cleaners", TargetID: targetID, TargetType: "SITE",
		}
	}
	assert.Equal(t, []feed.ScheduleAssigneeV2{assigneeOf("site_1"), assigneeOf("site_2")}, assignees)

	var occurrences []feed.ScheduleOccurrenceV2
	assert.NoError(t, exporter.DB.Table("new_schedule_occurrences").Order("target_id").Scan(&occurrences).Error)
	for i := range occurrences {
		assert.False(t, occurrences[i].ExportedAt.IsZero())
		occurrences[i].ExportedAt = time.Time{}
		for _, t := range []**time.Time{&occurrences[i].StartTime, &occurrences[i].DueTime, &occurrences[i].MissTime, &occurrences[i].CompletedAt} {
			if *t != nil {
				utc := (*t).UTC()
				*t = &utc
			}
		}
	}
	at := func(hour, minute, ms int) *time.Time {
		t := time.Date(2026, 9, 28, hour, minute, 0, ms*int(time.Millisecond), time.UTC)
		return &t
	}
	assert.Equal(t, []feed.ScheduleOccurrenceV2{
		{
			ID: "scheduleitem_1_occurrence_1_user_1", ScheduleID: "scheduleitem_1", OccurrenceID: "1", TemplateID: "template_1",
			StartTime: at(9, 30, 0), DueTime: at(17, 30, 0), MissTime: at(13, 30, 123), OccurrenceStatus: "COMPLETED",
			AuditID: "audit_1", CompletedAt: at(10, 0, 0), AssigneeType: "site", AssigneeSiteID: "location_1", AssigneeSiteName: "Sydney",
			AssigneeGroupID: "role_1", AssigneeGroupName: "Cleaners", CompletedByUserID: "user_2",
			CompletedByUserName: "Alan Turing", TargetID: "site_1", TargetType: "SITE", TargetName: "Sydney",
			OrganisationID: "role_org", UserID: "user_1", AssigneeStatus: "COMPLETED",
		},
		{
			ID: "scheduleitem_1_occurrence_1_user_1", ScheduleID: "scheduleitem_1", OccurrenceID: "1", TemplateID: "template_1",
			StartTime: at(9, 30, 0), DueTime: at(17, 30, 0), OccurrenceStatus: "WONT_DO",
			AssigneeType: "group", AssigneeGroupID: "role_1", AssigneeGroupName: "Cleaners", TargetID: "site_2",
			TargetType: "SITE", TargetName: "Melbourne", OrganisationID: "role_org", WontDoNote: "closed", UserID: "user_1",
			AssigneeStatus: "WONT_DO", Note: "closed",
		},
	}, occurrences)
}
