package inspections

import (
	"context"
	"fmt"
	"sync"
	"time"

	util2 "github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/exporter"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"go.uber.org/zap"
)

const maxGoRoutines = 10

// Client to be used with inspections
type Client struct {
	*zap.SugaredLogger

	apiClient      *httpapi.Client
	exporter       exporter.MittiJSONExporter
	SkipIDs        []string
	ModifiedAfter  time.Time
	ModifiedBefore time.Time
	BlockSize      string
	TemplateIDs    []string
	Archived       string
	Completed      string
	Incremental    bool
}

// InspectionClient is an interface to get inspections from server
type InspectionClient interface {
	Export(ctx context.Context) error
}

type InspectionClientCfg struct {
	SkipIDs        []string
	ModifiedAfter  time.Time
	ModifiedBefore time.Time
	BlockSize      string
	TemplateIDs    []string
	Archived       string
	Completed      string
	Incremental    bool
}

// NewInspectionClient returns a new instance of InspectionClient
func NewInspectionClient(cfg *InspectionClientCfg, apiClient *httpapi.Client, exporter exporter.MittiJSONExporter) InspectionClient {
	return &Client{
		apiClient:      apiClient,
		exporter:       exporter,
		SkipIDs:        cfg.SkipIDs,
		ModifiedAfter:  cfg.ModifiedAfter,
		ModifiedBefore: cfg.ModifiedBefore,
		BlockSize:      cfg.BlockSize,
		TemplateIDs:    cfg.TemplateIDs,
		Archived:       cfg.Archived,
		Completed:      cfg.Completed,
		Incremental:    cfg.Incremental,
		SugaredLogger:  util2.GetLogger(),
	}
}

// Name returns the name of the client
func (client *Client) Name() string {
	return "inspections"
}

// throttleFunc throttles the given function
func throttleFunc(tfunc func(httpapi.Inspection), inspection httpapi.Inspection, guard chan struct{}) {
	// If the channel is full then this go-routine will be in waiting state.
	// Multiple go routines can access this channel without any lock as the
	// the channel internally maintains the lock for send and receive operations
	guard <- struct{}{}
	tfunc(inspection)
	<-guard
}

// Export triggers the export
func (client *Client) Export(ctx context.Context) error {
	var wg sync.WaitGroup

	skipIDs := map[string]bool{}
	for _, id := range client.SkipIDs {
		skipIDs[id] = true
	}

	// Create a buffered channel of length maxGoroutines
	guard := make(chan struct{}, maxGoRoutines)

	operation := func(row httpapi.Inspection) {
		inspection, err := httpapi.GetRawInspection(ctx, client.apiClient, row.ID)
		util.Check(err, fmt.Sprintf("Failed to get inspection with id: %s", row.ID))

		client.exporter.WriteRow(row.ID, inspection)
	}

	callback := func(resp *httpapi.ListInspectionsResponse, l *zap.SugaredLogger) error {
		remaining := resp.Total - resp.Count
		if remaining < 0 {
			remaining = resp.Count
		}
		l.
			With("total", resp.Total).
			With("processing", resp.Count).
			With("remaining", remaining).
			Info("export")

		n := len(resp.Inspections)
		for _, row := range resp.Inspections {
			skip := skipIDs[row.ID]
			if skip {
				continue
			}

			wg.Add(1)
			go func(row httpapi.Inspection) {
				defer wg.Done()
				throttleFunc(operation, row, guard)
			}(row)
		}
		wg.Wait()

		if n > 0 {
			client.exporter.SetLastModifiedAt(resp.Inspections[n-1].ModifiedAt)
		}
		return nil
	}

	params := &httpapi.ListInspectionsParams{
		TemplateIDs: client.TemplateIDs,
		Archived:    client.Archived,
		Completed:   client.Completed,
	}

	modifiedAt := client.exporter.GetLastModifiedAt(client.ModifiedAfter)
	if modifiedAt != nil {
		params.ModifiedAfter = *modifiedAt
	}
	client.Infof("%s: exporting since %s", client.Name(), params.ModifiedAfter.Format(time.RFC1123))

	err := DrainInspections(ctx, client.apiClient, params, callback)
	util.Check(err, "failed to list inspections")

	client.Info("Export finished")

	return nil
}
