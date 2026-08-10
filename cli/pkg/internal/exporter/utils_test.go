package exporter_test

import (
	"log"
	"os"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/exporter"
)

// getTemporaryJSONExporter creates a JSONExporter that writes to a temp folder
func getTemporaryJSONExporter() exporter.MittiJSONExporter {
	dir, err := os.MkdirTemp("", "export")
	if err != nil {
		log.Fatal(err)
	}

	return exporter.NewJSONExporter(dir)
}
