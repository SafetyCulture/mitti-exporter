package feed

import (
	"sync"
)

var lock = &sync.Mutex{}
var statusInstance *ExportStatus

// GetExporterStatus will return a singleton instance of the ExporterStatus
func GetExporterStatus() *ExportStatus {
	lock.Lock()
	defer lock.Unlock()
	if statusInstance == nil {
		statusInstance = &ExportStatus{
			status: map[string]*ExportStatusItem{},
		}
	}
	return statusInstance
}

type ExportStatus struct {
	lock     sync.Mutex
	status   map[string]*ExportStatusItem
	finished bool
	started  bool
}

type ExportStatusItemStage string

const StageApi ExportStatusItemStage = "API_DOWNLOAD"
const StageCsv ExportStatusItemStage = "CSV_EXPORT"

type ExportStatusItem struct {
	Name               string
	Stage              ExportStatusItemStage
	Started            bool
	Finished           bool
	HasError           bool
	Counter            int64
	CounterDecremental bool
	StatusMessage      string
	DurationMs         int64
}

func (e *ExportStatus) Reset() {
	e.lock.Lock()
	e.started = false
	e.finished = false
	e.status = map[string]*ExportStatusItem{}
	e.lock.Unlock()
}

func (e *ExportStatus) StartFeedExport(feedName string, decremental bool) {
	e.lock.Lock()
	e.status[feedName] = &ExportStatusItem{
		Name:               feedName,
		Stage:              StageApi,
		Started:            true,
		Finished:           false,
		CounterDecremental: decremental,
	}
	e.lock.Unlock()
}

func (e *ExportStatus) UpdateStatus(feedName string, counter int64, durationMs int64) {
	e.lock.Lock()
	if _, ok := e.status[feedName]; ok {
		e.status[feedName].Counter = counter
		e.status[feedName].DurationMs = durationMs
	}
	e.lock.Unlock()
}

func (e *ExportStatus) IncrementStatus(feedName string, counter int64, durationMs int64) {
	e.lock.Lock()
	if _, ok := e.status[feedName]; ok {
		e.status[feedName].Counter += counter
		e.status[feedName].DurationMs = durationMs
	}
	e.lock.Unlock()
}

func (e *ExportStatus) UpdateStage(feedName string, stage ExportStatusItemStage, counterDecremental bool) {
	e.lock.Lock()
	if _, ok := e.status[feedName]; ok {
		e.status[feedName].Stage = stage
		e.status[feedName].CounterDecremental = counterDecremental
	}
	e.lock.Unlock()
}

func (e *ExportStatus) FinishFeedExport(feedName string, err error) {
	e.lock.Lock()
	if _, ok := e.status[feedName]; ok {
		if err != nil {
			e.status[feedName].HasError = true
			e.status[feedName].StatusMessage = err.Error()
		}
		e.status[feedName].Finished = true
	}
	e.lock.Unlock()
}

func (e *ExportStatus) ReadStatus() map[string]*ExportStatusItem {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.status
}

func (e *ExportStatus) ReadCounter(feedName string) int64 {
	e.lock.Lock()
	defer e.lock.Unlock()
	if _, ok := e.status[feedName]; ok {
		return e.status[feedName].Counter
	}
	return 0
}

func (e *ExportStatus) PurgeFinished() {
	e.lock.Lock()
	pendingFeeds := map[string]*ExportStatusItem{}
	for key, item := range e.status {
		if !item.Started || !item.Finished {
			pendingFeeds[key] = item
		}
	}
	e.status = pendingFeeds
	e.lock.Unlock()
}

func (e *ExportStatus) MarkExportCompleted() {
	e.lock.Lock()
	e.finished = true
	e.lock.Unlock()
}

func (e *ExportStatus) GetExportStarted() bool {
	return e.started
}

func (e *ExportStatus) GetExportCompleted() bool {
	return e.finished
}
