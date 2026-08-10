//go:build soak
// +build soak

package api_test

import (
	"os"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationDbSoakExportFeeds_should_successfully_export_with_significant_data(t *testing.T) {
	exporter, err := getTestingSQLExporter()
	assert.NoError(t, err)
	exporter.AutoMigrate = true

	apiClient := httpapi.NewClient(os.Getenv("TEST_API_HOST"), os.Getenv("TEST_ACCESS_TOKEN"))

	cfg := &feed.ExporterFeedCfg{
		AccessToken: "token-123",
	}

	exporterApp := feed.NewExporterApp(apiClient, nil, cfg)
	err = exporterApp.ExportFeeds(exporter, context.Background())
	assert.NoError(t, err)
}
