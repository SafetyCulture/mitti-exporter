package feed

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/gocarina/gocsv"

	"go.uber.org/zap"
)

// CSVExporter is an interface to export data feeds to CSV files
type CSVExporter struct {
	*SQLExporter
	ExportPath     string
	MaxRowsPerFile int
	Logger         *zap.SugaredLogger
	duration       time.Duration
}

// CreateSchema generated schema for a feed in csv format
func (e *CSVExporter) CreateSchema(feed Feed, rows interface{}) error {
	logger := e.Logger.With(
		"feed", feed.Name(),
	)
	logger.Info("writing out CSV schema file")

	exportFilePath := filepath.Join(e.ExportPath, fmt.Sprintf("%s.csv", feed.Name()))
	_, err := os.Stat(exportFilePath)

	if os.IsNotExist(err) {
		file, err := os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
		}

		err = gocsv.Marshal(rows, file)
		if err != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false)
		}
	}

	logger.Info("CSV file already exists, skipping")
	return nil
}

// FinaliseExport closes out an export
func (e *CSVExporter) FinaliseExport(feed Feed, rows interface{}) error {
	logger := e.Logger.With("feed", feed.Name())
	logger.Info("writing out CSV file")
	status := GetExporterStatus()
	status.UpdateStage(feed.Name(), StageCsv, false)

	err := e.cleanOldFiles(feed.Name())
	if err != nil {
		return err
	}

	limit := 10000
	if limit > e.MaxRowsPerFile {
		limit = e.MaxRowsPerFile
	}
	offset := 0
	rowsAdded := 0
	var file *os.File
	for {
		preTime := time.Now()
		resp := e.DB.Table(feed.Name()).
			Order(feed.Order()).
			Limit(limit).
			Offset(offset).
			Scan(rows)
		if resp.Error != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemDB, false)
		}
		postQueryTime := time.Now()

		if resp.RowsAffected == 0 || resp.RowsAffected == -1 {
			break
		}

		if rowsAdded >= e.MaxRowsPerFile {
			err = e.createRolloverFile(file, feed.Name())
			if err != nil {
				return err
			}
			file = nil
		}

		if file == nil {
			file, err = e.createNewFile(feed.Name())
			if err != nil {
				return err
			}

			err = gocsv.Marshal(rows, file)
			if err != nil {
				return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false)
			}
			rowsAdded = 0
		} else {
			err = gocsv.MarshalWithoutHeaders(rows, file)
			if err != nil {
				return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false)
			}
		}

		postWriteTime := time.Now()

		offset = offset + limit
		rowsAdded += int(resp.RowsAffected)

		status.UpdateStatus(feed.Name(), int64(rowsAdded), postWriteTime.Sub(preTime).Milliseconds())

		logger.With(
			"rows_added", rowsAdded,
			"total_time_ms", postWriteTime.Sub(preTime).Milliseconds(),
			"query_time_ms", postQueryTime.Sub(preTime).Milliseconds(),
			"write_time_ms", postWriteTime.Sub(postQueryTime).Milliseconds(),
		).Info("writing out CSV batch")
	}

	logger.Info("CSV exported")
	return nil
}

func (e *CSVExporter) createNewFile(feedName string) (*os.File, error) {
	exportFilePath := filepath.Join(e.ExportPath, fmt.Sprintf("%s.csv", feedName))
	file, err := os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_TRUNC, 0666)
	if err != nil {
		return nil, events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
	}
	return file, nil
}

func (e *CSVExporter) createRolloverFile(file *os.File, feedName string) error {
	/* 	IMPORTANT NOTE: this is important for `windows` builds. Linux/Unix handles this scenario differently.
	If there is an existing handler for this file, the error will be:
	`The process cannot access the file because it is being used by another process.`
	Therefore, the `close` is important
	FYI: https://github.com/golang/go/issues/8914
	*/
	if file != nil {
		err := file.Close()
		if err != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
		}
	}

	exportFilePath := filepath.Join(e.ExportPath, fmt.Sprintf("%s.csv", feedName))
	newFilePath := filepath.Join(e.ExportPath, fmt.Sprintf("%s-%s.csv", feedName, time.Now().Format("20060102150405.999999")))

	_, err := fileExists(exportFilePath)
	if err != nil {
		return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
	}

	err = os.Rename(exportFilePath, newFilePath)
	if err != nil {
		return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
	}

	return nil
}

func (e *CSVExporter) cleanOldFiles(feedName string) error {
	files, err := filepath.Glob(filepath.Join(e.ExportPath, fmt.Sprintf("%s*.csv", feedName)))
	if err != nil {
		return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
	}

	for _, f := range files {
		err = os.Remove(f)
		if err != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false)
		}
	}

	return nil
}

// GetDuration will return the duration for exporting a batch
func (e *CSVExporter) GetDuration() time.Duration {
	// NOT IMPLEMENTED
	return e.duration
}

func fileExists(filename string) (bool, error) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !info.IsDir(), nil
}

// NewCSVExporter creates a new instance of CSVExporter
func NewCSVExporter(exportPath, exportMediaPath string, maxRowsPerFile int) (*CSVExporter, error) {
	sqlExporter, err := NewSQLExporter("sqlite", filepath.Join(exportPath, "sqlite.db"), true, exportMediaPath)
	if err != nil {
		return nil, err
	}
	if res := sqlExporter.DB.Exec("PRAGMA busy_timeout = 20000"); res.Error != nil {
		return nil, res.Error
	}

	return &CSVExporter{
		SQLExporter:    sqlExporter,
		ExportPath:     exportPath,
		MaxRowsPerFile: maxRowsPerFile,
		Logger:         sqlExporter.Logger,
		duration:       0,
	}, nil
}
