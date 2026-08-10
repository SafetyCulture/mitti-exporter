package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
)

func TestActionAssigneeFeedExport_should_export_rows_to_sql_db(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	actionAssigneeFeed := feed.ActionAssigneeFeed{
		ModifiedAfter: time.Now(),
		Incremental:   true,
	}

	err = actionAssigneeFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)

	var rows []feed.ActionAssignee
	resp := exporter.DB.Table("action_assignees").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 2, len(rows))
	assert.Equal(t, "email@example.com", rows[0].AssigneeID)
}
