package feed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SQLExporter is an interface to export data feeds to SQL databases
type SQLExporter struct {
	DB              *gorm.DB
	Logger          *zap.SugaredLogger
	AutoMigrate     bool
	ExportMediaPath string
	duration        time.Duration
	mu              sync.Mutex
}

// DBConnection db connection
type DBConnection struct {
	db  *gorm.DB
	err error
}

// SupportsUpsert returns a bool if the exporter supports upserts
func (e *SQLExporter) SupportsUpsert() bool {
	return true
}

// ParameterLimit returns the number of parameters supported by the target DB
func (e *SQLExporter) ParameterLimit() int {
	switch e.DB.Name() {
	case "sqlserver":
		return 2100
	case "sqlite":
		return 32768
	}

	return 65536
}

// CreateSchema creates the schema on the DB for the supplied feed
func (e *SQLExporter) CreateSchema(feed Feed, _ interface{}) error {
	return e.InitFeed(feed, &InitFeedOptions{
		Truncate: false,
	})
}

// InitFeed initialises any tables required to export
func (e *SQLExporter) InitFeed(feed Feed, opts *InitFeedOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	model := feed.Model()

	if e.AutoMigrate {
		err := e.DB.AutoMigrate(model)
		if err != nil {
			return events.NewEventError(err, events.ErrorSeverityError, events.ErrorSubSystemDB, true)
		}
	}

	if opts.Truncate {
		e.Logger.With(
			"feed", feed.Name(),
		).Info("truncating")
		result := e.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(model)
		if result.Error != nil {
			return events.NewEventError(result.Error, events.ErrorSeverityError, events.ErrorSubSystemDB, true)
		}
	}

	return nil
}

// GetDuration will return the duration for exporting a batch
func (e *SQLExporter) GetDuration() time.Duration {
	return e.duration
}

// DeleteRowsIfExist will delete the rows if already exist
func (e *SQLExporter) DeleteRowsIfExist(feed Feed, query string, args ...interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	del := e.DB.Table(feed.Name()).
		Clauses(clause.Where{
			Exprs: []clause.Expression{
				clause.Expr{
					SQL:  query,
					Vars: args,
				},
			},
		}).
		Delete(feed.Model())
	if del.Error != nil {
		return events.NewEventErrorWithMessage(del.Error, events.ErrorSeverityError, events.ErrorSubSystemDataIntegrity, false, "unable to delete rows")
	}

	return nil
}

// WriteRows writes out the rows to the DB
func (e *SQLExporter) WriteRows(feed Feed, rows interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var columns []clause.Column
	for _, column := range feed.PrimaryKey() {
		columns = append(columns, clause.Column{Name: column})
	}

	start := time.Now()
	insert := e.DB.
		Table(feed.Name()).
		Clauses(clause.OnConflict{
			Columns:   columns,
			DoUpdates: clause.AssignmentColumns(feed.Columns()),
		}).
		Create(rows)
	e.duration = time.Since(start)
	if insert.Error != nil {
		return events.NewEventErrorWithMessage(insert.Error, events.ErrorSeverityError, events.ErrorSubSystemDB, false, "unable to insert rows")
	}

	return nil
}

// UpdateRows batch updates. Returns number of rows updated or error. Works with single PKey, not with composed PKeys
func (e *SQLExporter) UpdateRows(feed Feed, primaryKeys []string, element map[string]interface{}) (int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := e.DB.
		Model(feed.Model()).
		Where(primaryKeys).
		Updates(element)
	if result.Error != nil {
		return 0, events.NewEventErrorWithMessage(result.Error, events.ErrorSeverityError, events.ErrorSubSystemDB, false, "unable to updare rows")
	}

	return result.RowsAffected, nil
}

type modifiedAtRow struct {
	// ExportedAt is here so gorm has an additional field to sort on in SQL Server
	ExportedAt time.Time
	ModifiedAt time.Time
}

// LastModifiedAt returns the latest stored modified at date for the feed
func (e *SQLExporter) LastModifiedAt(feed Feed, modifiedAfter time.Time, orgID string) (time.Time, error) {
	latestRow := modifiedAtRow{}

	var result *gorm.DB
	result = e.DB.Table(feed.Name()).
		Where("organisation_id = ?", orgID).
		Order("modified_at DESC").
		Limit(1).
		First(&latestRow)
	if result.RowsAffected == 0 {
		// This can happen when there is no org_id stored in the existing data.
		// In this case try to get the latest modifiedAt timestamp  from the table
		// where there is no org_id defined.
		result = e.DB.Table(feed.Name()).
			Where("organisation_id IS NULL OR organisation_id = ''").
			Order("modified_at DESC").
			Limit(1).
			First(&latestRow)
	}
	if result.RowsAffected != 0 && modifiedAfter.Before(latestRow.ModifiedAt) {
		return latestRow.ModifiedAt, nil
	}

	return modifiedAfter, nil
}

// LastRecord returns the latest stored record the feed
func (e *SQLExporter) LastRecord(feed Feed, fallbackTime time.Time, orgID string, sortColumn string) time.Time {
	var latestRow = time.Time{}

	result := e.DB.Table(feed.Name()).
		Select(sortColumn).
		Where("organisation_id = ?", orgID).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: sortColumn,
				Raw:  false,
			},
			Desc:    true,
			Reorder: false,
		}).
		Limit(1).
		Scan(&latestRow)

	if result.RowsAffected != 0 {
		return latestRow
	}

	return fallbackTime
}

// FinaliseExport closes out an export
func (e *SQLExporter) FinaliseExport(Feed, interface{}) error {
	return nil
}

// WriteMedia writes the media to a file
func (e *SQLExporter) WriteMedia(auditID, mediaID, contentType string, body []byte) error {
	exportMediaDir := filepath.Join(e.ExportMediaPath, auditID)
	if err := os.MkdirAll(exportMediaDir, os.ModePerm); err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false, fmt.Sprintf("create directory %s", exportMediaDir))
	}

	ext := strings.Split(contentType, "/")
	exportFilePath := filepath.Join(exportMediaDir, fmt.Sprintf("%s.%s", mediaID, ext[1]))

	file, err := os.OpenFile(exportFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false, fmt.Sprintf("open file %v", exportFilePath))
	}
	defer file.Close()

	_, err = file.WriteAt(body, 0)
	if err != nil {
		return events.NewEventErrorWithMessage(err, events.ErrorSeverityError, events.ErrorSubSystemFileOperations, false, "write media to file")
	}

	return nil
}

// NewSQLExporter creates a new instance of the SQLExporter
func NewSQLExporter(dialect, connectionString string, autoMigrate bool, exportMediaPath string) (*SQLExporter, error) {
	db, err := GetDatabase(dialect, connectionString)
	if err != nil {
		return nil, errors.Wrap(err, "connect to DB")
	}

	return &SQLExporter{
		DB:              db,
		Logger:          logger.GetLogger(),
		AutoMigrate:     autoMigrate,
		ExportMediaPath: exportMediaPath,
		duration:        0,
	}, nil
}

// NewSQLiteExporter creates a new instance of SQLExporter for SQLITE
func NewSQLiteExporter(exportPath string, exportMediaPath string) (*SQLExporter, error) {
	sqlExporter, err := NewSQLExporter("sqlite", filepath.Join(exportPath, "sqlite_export.db"), true, exportMediaPath)
	if err != nil {
		return nil, err
	}
	if res := sqlExporter.DB.Exec("PRAGMA busy_timeout = 20000"); res.Error != nil {
		return nil, res.Error
	}

	return sqlExporter, nil
}

// GetDatabase validates the db credentials and return a DB connection
func GetDatabase(dialect string, connectionString string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch dialect {
	case "mysql":
		dialector = mysql.Open(connectionString)
	case "postgres":
		dialector = postgres.Open(connectionString)
	case "sqlserver":
		dialector = sqlserver.Open(connectionString)
	case "sqlite":
		dialector = sqlite.Open(connectionString)
	default:
		return nil, fmt.Errorf("invalid database dialect %s", dialect)
	}

	l := logger.GetLogger()
	gormLogger := &logger.GormLogger{
		SugaredLogger: l,
		SlowThreshold: 30 * time.Second,
	}

	gormConfig := gorm.Config{
		Logger: gormLogger, // use logger.Default.LogMode(logger.Info) for checking the statements (gorm.io/logger)
	}

	conn := make(chan DBConnection)
	go connectToDB(dialector, &gormConfig, conn)

	select {
	case dbResult := <-conn:
		if dbResult.err != nil {
			return nil, dbResult.err
		} else {
			return dbResult.db, nil
		}
	case <-time.After(5 * time.Second):
		return nil, errors.New("connection timed out")
	}
}

func connectToDB(d gorm.Dialector, g *gorm.Config, result chan<- DBConnection) {
	db, err := gorm.Open(d, g)
	result <- DBConnection{
		db:  db,
		err: err,
	}
}
