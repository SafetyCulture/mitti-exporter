package exporter_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/exporter"
	"github.com/stretchr/testify/assert"
)

func TestGetLastModifiedAt(t *testing.T) {
	tests := [...]struct {
		name     string
		expected *time.Time
		fileName string
	}{
		{
			name:     "LastModifiedFilePathDontExist",
			expected: &time.Time{},
			fileName: "random",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonExporter := exporter.NewJSONExporter("")
			resp := jsonExporter.GetLastModifiedAt(time.Time{})
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestLastModifiedAt(t *testing.T) {
	tmpExporter := getTemporaryJSONExporter()
	now := time.Now()
	tmpExporter.SetLastModifiedAt(now)

	lastModified := tmpExporter.GetLastModifiedAt(time.Time{})
	assert.NotNil(t, lastModified)

	expected := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	actual := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		lastModified.Year(), lastModified.Month(), lastModified.Day(),
		lastModified.Hour(), lastModified.Minute(), lastModified.Second())
	assert.Equal(t, expected, actual)
}

func TestLastModifiedAtWithConfig(t *testing.T) {
	tmpExporter1 := getTemporaryJSONExporter()
	now := time.Now()
	tmpExporter1.SetLastModifiedAt(now)

	// config timestamp(modifiedAfter) < last modified timestamp in the file
	layout := "2006-01-02T15:04:05.000Z"
	str := "2021-03-1T11:45:26.371Z"
	modifiedAfter, _ := time.Parse(layout, str)
	lastModified := tmpExporter1.GetLastModifiedAt(modifiedAfter)
	assert.NotNil(t, lastModified)

	expected := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	actual := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		lastModified.Year(), lastModified.Month(), lastModified.Day(),
		lastModified.Hour(), lastModified.Minute(), lastModified.Second())
	assert.Equal(t, expected, actual)

	// config timestamp(now) > last modified timestamp in the file
	layout = "2006-01-02T15:04:05.000Z"
	str = "2010-11-1T11:45:26.371Z"
	modifiedAfter, _ = time.Parse(layout, str)
	tmpExporter2 := getTemporaryJSONExporter()
	tmpExporter2.SetLastModifiedAt(modifiedAfter)
	lastModified = tmpExporter2.GetLastModifiedAt(now)
	assert.NotNil(t, lastModified)

	expected = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	actual = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		lastModified.Year(), lastModified.Month(), lastModified.Day(),
		lastModified.Hour(), lastModified.Minute(), lastModified.Second())
	assert.Equal(t, expected, actual)
}

func TestLastModifiedAtAfterRestart(t *testing.T) {
	dir, err := os.MkdirTemp("", "export")
	assert.NoError(t, err)

	tmpExporter1 := exporter.NewJSONExporter(dir)
	now := time.Now()
	tmpExporter1.SetLastModifiedAt(now)

	tmpExporter2 := exporter.NewJSONExporter(dir)
	lastModified := tmpExporter2.GetLastModifiedAt(time.Time{})
	assert.NotNil(t, lastModified)

	expected := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	actual := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		lastModified.Year(), lastModified.Month(), lastModified.Day(),
		lastModified.Hour(), lastModified.Minute(), lastModified.Second())
	assert.Equal(t, expected, actual)
}

func TestHandlesOverWrittingLongString(t *testing.T) {
	dir, err := os.MkdirTemp("", "export")
	assert.NoError(t, err)

	exportFilePath := filepath.Join(dir, "last-modified")
	lastModifiedFile, err := os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE, 0666)
	assert.NoError(t, err)

	_, err = lastModifiedFile.Write([]byte("a-really-long-string-abc-123-456-789-longer-than-a-time-stamp"))
	assert.NoError(t, err)

	tmpExporter1 := exporter.NewJSONExporter(dir)
	now := time.Now()
	tmpExporter1.SetLastModifiedAt(now)

	tmpExporter2 := exporter.NewJSONExporter(dir)
	lastModified := tmpExporter2.GetLastModifiedAt(time.Time{})
	assert.NotNil(t, lastModified)

	expected := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	actual := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		lastModified.Year(), lastModified.Month(), lastModified.Day(),
		lastModified.Hour(), lastModified.Minute(), lastModified.Second())
	assert.Equal(t, expected, actual)
}

func TestWriteRow(t *testing.T) {
	tmpExporter := getTemporaryJSONExporter()
	str := `{"abc": 123}`
	var tmp json.RawMessage = []byte(str)
	tmpExporter.WriteRow("tmp-file", &tmp)
}
