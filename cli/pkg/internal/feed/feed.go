package feed

//go:generate mockery --name Exporter --case underscore --output mocks --outpkg mocks --filename exporter_mock.go

import (
	"context"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
)

// Feed is an interface to a data feed. It provides methods to export the data to an exporter
type Feed interface {
	Name() string
	Model() interface{}
	RowsModel() interface{}

	PrimaryKey() []string
	Columns() []string
	Order() string

	CreateSchema(exporter Exporter) error
	Export(ctx context.Context, apiClient *httpapi.Client, exporter Exporter, orgID string) error

	// HasRemainingInformation - true if the feed data source provides remaining number of items
	HasRemainingInformation() bool
}

// InitFeedOptions contains the options used when initialising a feed
type InitFeedOptions struct {
	Truncate bool
}

// Exporter is an interface to a Feed exporter. It provides methods to write rows out to an implemented format
type Exporter interface {
	InitFeed(feed Feed, opts *InitFeedOptions) error
	CreateSchema(feed Feed, rows interface{}) error

	WriteRows(feed Feed, rows interface{}) error
	UpdateRows(feed Feed, primaryKeys []string, element map[string]interface{}) (int64, error)

	FinaliseExport(feed Feed, rows interface{}) error
	LastModifiedAt(feed Feed, modifiedAfter time.Time, orgID string) (time.Time, error)
	LastRecord(feed Feed, fallbackTime time.Time, orgID string, sortColumn string) time.Time
	WriteMedia(auditID string, mediaID string, contentType string, body []byte) error
	DeleteRowsIfExist(feed Feed, query string, args ...interface{}) error
	GetDuration() time.Duration

	SupportsUpsert() bool
	ParameterLimit() int
}

// resolveModifiedAfter returns the modified-after cutoff for an incremental
// run: the configured backfill date when set (ignoring the stored watermark),
// otherwise the feed's incremental watermark.
func resolveModifiedAfter(exporter Exporter, feed Feed, modifiedAfter, backfillFrom time.Time, orgID string) (time.Time, error) {
	if !backfillFrom.IsZero() {
		logger.GetLogger().With("feed", feed.Name(), "backfill_from", backfillFrom).
			Warn("backfill_from is set, ignoring stored incremental watermark")
		return backfillFrom, nil
	}
	return exporter.LastModifiedAt(feed, modifiedAfter, orgID)
}
