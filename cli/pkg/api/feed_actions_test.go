package api_test

import (
	"context"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestActionFeedExport_should_export_rows_to_sql_db(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	actionsFeed := feed.ActionFeed{
		Limit: 100,
	}

	err = actionsFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)

	var rows []feed.Action
	resp := exporter.DB.Table("actions").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 2, len(rows))
	assert.Equal(t, "123", rows[0].ID)
}

func TestActionFeed_Export_ShouldNotFailWhen403(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())
	gock.New("http://localhost:9999").
		Get("/feed/actions").
		Reply(403)

	actionsFeed := feed.ActionFeed{
		Limit: 100,
	}

	err = actionsFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)
}
