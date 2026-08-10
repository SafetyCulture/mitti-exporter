package exporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
)

const (
	lastModified = "last-modified"
	layout       = time.RFC3339Nano
)

// JSONExporter is an interface to export data feeds to json files
type JSONExporter struct {
	exportPath       string
	lastModifiedFile *os.File
}

// NewJSONExporter creates new instance of JSONExporter
func NewJSONExporter(exportPath string) MittiJSONExporter {
	return &JSONExporter{
		exportPath: exportPath,
	}
}

// SetLastModifiedAt writes last modified date to a file
func (e *JSONExporter) SetLastModifiedAt(modifiedAt time.Time) {
	exportFilePath := filepath.Join(e.exportPath, lastModified)
	if e.lastModifiedFile == nil {
		var err error
		e.lastModifiedFile, err = os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE, 0666)
		util.Check(err, "Failed to open last-modified file")
	}

	err := e.lastModifiedFile.Truncate(0)
	util.Check(err, "Failed to truncate last-modified to a file")
	_, err = e.lastModifiedFile.WriteAt([]byte(modifiedAt.Format(layout)), 0)
	util.Check(err, "Failed to write last-modified to a file")

	err = e.lastModifiedFile.Sync()
	util.Check(err, "Failed to write last-modified to a file")
}

// GetLastModifiedAt returns last modified timestamp
// Value from config-path(modifiedAfter) -> (A)
// Value from last-modified file -> (B)
func (e *JSONExporter) GetLastModifiedAt(modifiedAfter time.Time) *time.Time {
	exportFilePath := filepath.Join(e.exportPath, lastModified)
	_, err := os.Stat(exportFilePath)
	if os.IsNotExist(err) {
		return &modifiedAfter
	}

	if e.lastModifiedFile == nil {
		var err error
		e.lastModifiedFile, err = os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE, 0666)
		util.Check(err, "Failed to open last-modified file")
	}

	b := make([]byte, 50)
	_, err = e.lastModifiedFile.Read(b)
	util.Check(err, "Failed to read last-modified")

	modifiedAt, err := time.Parse(layout, strings.TrimSpace(string(bytes.Trim(b, "\x00"))))
	util.Check(err, "Failed to convert last-modified to iso format")

	// If (A) is less than (B) then return (B)
	if !modifiedAt.IsZero() && modifiedAfter.Before(modifiedAt) {
		return &modifiedAt
	}

	return &modifiedAfter
}

// WriteRow writes the json response into a file
func (e *JSONExporter) WriteRow(name string, row *json.RawMessage) {
	str, err := json.MarshalIndent(row, "", " ")
	util.Check(err, "Failed to marshal inspection to JSON")

	exportFilePath := filepath.Join(e.exportPath, fmt.Sprintf("%s.json", name))
	file, err := os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE, 0666)
	util.Check(err, "Failed to open file")
	defer file.Close()

	_, err = file.WriteAt(str, 0)
	util.Check(err, "Failed to write inspection to a file")
}
