package api_test

import (
	"context"
	"net/http"
	"path"
	"testing"
	"time"

	"gopkg.in/h2non/gock.v1"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
)

func TestInspectionFeedExport_should_export_rows_to_sql_db(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())
	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		Reply(http.StatusOK).
		File(path.Join("mocks", "set_1", "inspections_deleted_single_page.json"))
	inspectionsFeed := feed.InspectionFeed{
		SkipIDs:       []string{},
		ModifiedAfter: time.Now(),
		TemplateIDs:   []string{},
		Archived:      "both",
		Completed:     "both",
		Incremental:   true,
	}

	err = inspectionsFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)

	var rows []feed.Inspection
	resp := exporter.DB.Table("inspections").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 3, len(rows))
	assert.Equal(t, "audit_47ac0dce16f94d73b5178372368af162", rows[0].ID)
	assert.True(t, rows[0].Deleted)

	assert.Equal(t, "audit_4e28ab2cce8c44a781d376d0ac47dc92", rows[1].ID)
	assert.False(t, rows[1].Deleted)

	assert.Equal(t, "audit_4d95cb4be1e7488bba5893fecd2379d2", rows[2].ID)
	assert.False(t, rows[2].Deleted)
}
