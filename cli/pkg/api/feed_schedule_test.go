package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
)

func TestScheduleFeedExport_should_export_rows_to_sql_db(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	startDate, _ := time.Parse(time.RFC3339, "2021-08-12T12:30:00.000Z")
	scheduleOccurrenceFeed := feed.ScheduleOccurrenceFeed{
		TemplateIDs: []string{"template_2"},
		StartDate:   startDate,
	}

	err = scheduleOccurrenceFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)

	var rows []feed.ScheduleOccurrence
	resp := exporter.DB.Table("schedule_occurrences").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 3, len(rows))
	assert.Equal(t, "scheduleitem_1_occurrence_1641027600000_user_1", rows[0].ID)
}

func TestScheduleFeedExport_should_export_rows_to_sql_db_when_ResumeDownload(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	startDate, _ := time.Parse(time.RFC3339, "2021-08-12T12:30:00.000Z")
	scheduleOccurrenceFeed := feed.ScheduleOccurrenceFeed{
		TemplateIDs:    []string{"template_2"},
		StartDate:      startDate,
		ResumeDownload: true,
	}

	err = scheduleOccurrenceFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)

	var rows []feed.ScheduleOccurrence
	resp := exporter.DB.Table("schedule_occurrences").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 3, len(rows))
	assert.Equal(t, "scheduleitem_1_occurrence_1641027600000_user_1", rows[0].ID)
}
