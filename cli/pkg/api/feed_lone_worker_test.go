package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

const loneWorkerJobsPage = `{
	"metadata": {},
	"data": [
		{
			"id": "job_1",
			"code": "JOB-1",
			"name": "Morning round",
			"status": "FINISHED",
			"job_type_id": "jt_1",
			"job_type_name": "Site visit",
			"group_id": "grp_1",
			"user_id": "usr_1",
			"travel_started_at": "2026-10-01T08:00:00Z",
			"started_at": "2026-10-01T08:30:00Z",
			"finished_at": "2026-10-01T10:00:00Z",
			"planned_duration_minutes": 90,
			"actual_duration_seconds": 5400,
			"panic_reason": "PANIC_REASON_NO_PANIC",
			"start_address": "1 Test St",
			"start_latitude": -33.86,
			"start_longitude": 151.2,
			"satellite_device_id": "sat_1",
			"satellite_device_serial_number": "SN-1",
			"modified_at": "2026-10-01T10:00:05Z"
		},
		{
			"id": "job_2",
			"code": "JOB-2",
			"status": "CANCELED",
			"job_type_id": "jt_1",
			"job_type_name": "Site visit",
			"group_id": "grp_1",
			"user_id": "usr_2",
			"travel_started_at": "2026-10-02T08:00:00Z",
			"canceled_at": "2026-10-02T08:15:00Z",
			"planned_duration_minutes": 60,
			"panic_reason": "PANIC_REASON_NO_PANIC",
			"modified_at": "2026-10-02T08:15:05Z"
		}
	]
}`

func TestLoneWorkerJobFeed_should_export_rows_to_sql_db(t *testing.T) {
	defer gock.Off()

	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())
	gock.New("http://localhost:9999").
		Get("/loneworker/v1/feed/jobs").
		Reply(200).
		BodyString(loneWorkerJobsPage)

	loneWorkerFeed := feed.LoneWorkerJobFeed{
		Incremental: true,
		Limit:       100,
	}

	err = loneWorkerFeed.Export(context.Background(), apiClient, exporter, "role_ada3042f16a44249915ddc088adef92a")
	assert.NoError(t, err)

	var rows []feed.LoneWorkerJob
	resp := exporter.DB.Table("lone_worker_jobs").Scan(&rows)
	assert.Nil(t, resp.Error)
	assert.Len(t, rows, 2)

	assert.Equal(t, "job_1", rows[0].ID)
	assert.Equal(t, "FINISHED", rows[0].Status)
	assert.Equal(t, "Morning round", rows[0].Name)
	assert.Equal(t, int32(90), rows[0].PlannedDurationMinutes)
	assert.Equal(t, int32(5400), *rows[0].ActualDurationSeconds)
	assert.Equal(t, -33.86, *rows[0].StartLatitude)
	assert.Equal(t, "SN-1", rows[0].SatelliteDeviceSerialNumber)
	// The endpoint does not return the organisation, so the exporter fills the column from the export's org.
	assert.Equal(t, "role_ada3042f16a44249915ddc088adef92a", rows[0].OrganisationID)
	assert.Equal(t, time.Date(2026, 10, 1, 10, 0, 5, 0, time.UTC), rows[0].ModifiedAt)

	assert.Equal(t, "job_2", rows[1].ID)
	assert.Equal(t, "CANCELED", rows[1].Status)
	assert.Equal(t, "2026-10-02T08:15:00Z", rows[1].CanceledAt)
	// A canceled journey has no start or finish, so these stay empty rather than becoming 0 or the year 0001.
	assert.Empty(t, rows[1].StartedAt)
	assert.Empty(t, rows[1].FinishedAt)
	assert.Nil(t, rows[1].ActualDurationSeconds)
	assert.Nil(t, rows[1].StartLatitude)
	assert.Nil(t, rows[1].StartLongitude)
	assert.Empty(t, rows[1].SatelliteDeviceID)
}

// The second run continues from the newest modified_at in the table, and it does not duplicate rows.
func TestLoneWorkerJobFeed_should_export_incrementally_on_second_run(t *testing.T) {
	defer gock.Off()

	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/loneworker/v1/feed/jobs").
		Reply(200).
		BodyString(loneWorkerJobsPage)

	loneWorkerFeed := feed.LoneWorkerJobFeed{Incremental: true, Limit: 100}
	err = loneWorkerFeed.Export(context.Background(), apiClient, exporter, "role_1")
	assert.NoError(t, err)

	var count int64
	assert.Nil(t, exporter.DB.Table("lone_worker_jobs").Count(&count).Error)
	assert.Equal(t, int64(2), count)

	// The next run asks only for jobs modified after the newest row it holds.
	gock.New("http://localhost:9999").
		Get("/loneworker/v1/feed/jobs").
		MatchParam("modified_after", "2026-10-02T08:15:05Z").
		Reply(200).
		BodyString(`{"metadata":{},"data":[]}`)

	secondFeed := feed.LoneWorkerJobFeed{Incremental: true, Limit: 100}
	err = secondFeed.Export(context.Background(), apiClient, exporter, "role_1")
	assert.NoError(t, err)

	assert.Nil(t, exporter.DB.Table("lone_worker_jobs").Count(&count).Error)
	assert.Equal(t, int64(2), count)
	assert.True(t, gock.IsDone(), "the incremental request was not sent")
}
