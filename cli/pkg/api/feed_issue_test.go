package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
)

func TestIssueFeed_Export_ShouldExportRows(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockIssuesFeed(apiClient.HTTPClient())

	actionsFeed := feed.IssueFeed{
		Limit: 100,
	}

	err = actionsFeed.Export(context.Background(), apiClient, exporter, "")
	assert.NoError(t, err)

	var rows []feed.Issue
	resp := exporter.DB.Table("issues").Scan(&rows)

	assert.Nil(t, resp.Error)
	assert.Equal(t, 39, len(rows))
	testAllValues(t, &rows[0])
	testAllNulls(t, &rows[1])
}

func testAllNulls(t *testing.T, issue *feed.Issue) {
	assert.Equal(t, "52a88aeb-5ec6-4876-8c6c-85a642e4bddc", issue.ID)
	assert.Equal(t, "", issue.Title)
	assert.Equal(t, "", issue.Description)
	assert.Equal(t, "user_0590e8a0dfbc64798a2426c2fa76a7415", issue.CreatorID)
	assert.Equal(t, "", issue.CreatorUserName)

	// uses .Now() if missing
	assert.NotEqual(t, time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), issue.CreatedAt)

	// not sure how correct is this approach
	// assert.Equal(t, time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), issue.DueAt)
	assert.Nil(t, issue.DueAt)

	assert.Equal(t, "", issue.Priority)
	assert.Equal(t, "", issue.Status)
	assert.Equal(t, "", issue.TemplateID)
	assert.Equal(t, "", issue.InspectionID)
	assert.Equal(t, "", issue.InspectionName)
	assert.Equal(t, "", issue.SiteID)
	assert.Equal(t, "", issue.SiteName)
	assert.Equal(t, "", issue.LocationName)
	assert.Equal(t, "", issue.CategoryID)
	assert.Equal(t, "", issue.CategoryLabel)
}

func testAllValues(t *testing.T, issue *feed.Issue) {
	assert.Equal(t, "56bc5efa-2420-483d-bad1-27b35922c403", issue.ID)
	assert.Equal(t, "Injury - 14 Apr 2020, 10:36 am", issue.Title)
	assert.Equal(t, "some description", issue.Description)
	assert.Equal(t, "user_51d3dbc686eb4790980f6414513d1c05", issue.CreatorID)
	assert.Equal(t, "🦄", issue.CreatorUserName)
	assert.Equal(t, time.Date(2020, time.April, 14, 0, 36, 53, 304000000, time.UTC), issue.CreatedAt)
	expected := time.Date(2020, time.April, 14, 0, 36, 53, 304000000, time.UTC)
	assert.Equal(t, &expected, issue.DueAt)
	assert.Equal(t, "NONE", issue.Priority)
	assert.Equal(t, "OPEN", issue.Status)
	assert.Equal(t, "55bc5efa-2420-483d-bad1-27b35922c455", issue.TemplateID)
	assert.Equal(t, "66bc5efa-2420-483d-bad1-27b35922c466", issue.InspectionID)
	assert.Equal(t, "some name", issue.InspectionName)
	assert.Equal(t, "77bc5efa-2420-483d-bad1-27b35922c477", issue.SiteID)
	assert.Equal(t, "site name", issue.SiteName)
	assert.Equal(t, "88bc5efa-2420-483d-bad1-27b35922c488", issue.LocationName)
	assert.Equal(t, "592ec130-90e0-4c0e-a1c0-1f37f12f5fb5", issue.CategoryID)
	assert.Equal(t, "Tow Trucks", issue.CategoryLabel)
	assert.Equal(t, time.Date(2020, time.April, 14, 2, 36, 53, 304000000, time.UTC), issue.ModifiedAt)
	completedAt := time.Date(2020, time.April, 14, 2, 36, 53, 304000000, time.UTC)
	assert.Equal(t, &completedAt, issue.CompletedAt)
}
