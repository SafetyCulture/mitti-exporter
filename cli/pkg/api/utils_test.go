package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	exporterAPI "github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

// GetTestClient creates a new test apiClient
func GetTestClient(opts ...httpapi.Opt) *httpapi.Client {
	cfg := httpapi.ClientCfg{
		Addr:                "http://localhost:9999",
		AuthorizationHeader: "abc123",
		IntegrationID:       "test",
		IntegrationVersion:  "dev",
	}

	apiClient := httpapi.NewClient(&cfg, opts...)
	apiClient.RetryWaitMin = 10 * time.Millisecond
	apiClient.RetryWaitMax = 10 * time.Millisecond
	apiClient.CheckForRetry = httpapi.DefaultRetryPolicy
	apiClient.RetryMax = 1
	return apiClient
}

// getInmemorySQLExporter creates a SQLExporter that uses an inmemory DB
func getInmemorySQLExporter(exportMediaPath string) (*feed.SQLExporter, error) {
	return feed.NewSQLExporter("sqlite", "file::memory:", true, exportMediaPath)
}

// getTemporaryReportExporter creates a ReportExporter that writes to a temp folder
func getTemporaryReportExporter(format []string, preferenceID string, filename string) (*feed.ReportExporter, error) {
	dir, err := os.MkdirTemp("", "export")
	if err != nil {
		log.Fatal(err)
	}

	cfg := &exporterAPI.ExporterConfiguration{}
	cfg.Report.Format = format
	cfg.Report.PreferenceID = preferenceID
	cfg.Report.FilenameConvention = filename
	cfg.Report.RetryTimeout = 10
	return exporterAPI.NewReportExporter(dir, cfg.ToReporterConfig())
}

// getTemporaryCSVExporter creates a CSVExporter that writes to a temp folder
func getTemporaryCSVExporter() (*feed.CSVExporter, error) {
	dir, err := os.MkdirTemp("", "export")
	if err != nil {
		log.Fatal(err)
	}

	return feed.NewCSVExporter(dir, "", 100000)
}

func fileExists(t *testing.T, expectedPath string) {
	_, err := os.Stat(expectedPath)
	assert.NoError(t, err)
}

func getFileModTime(filePath string) (time.Time, error) {
	file, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}
	return file.ModTime(), nil
}

// filesEqualish checks if files are equal enough (ignoring dates)
func filesEqualish(t *testing.T, expectedPath, actualPath string) {
	expectedFile, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)

	actualFile, err := os.ReadFile(actualPath)
	assert.NoError(t, err)

	assert.Equal(t,
		dateRegex.ReplaceAllLiteralString(strings.TrimSpace(string(expectedFile)), "--date--"),
		dateRegex.ReplaceAllLiteralString(strings.TrimSpace(string(actualFile)), "--date--"),
	)
}

func countFileLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := file.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// getTemporaryCSVExporterWithMaxRowsLimit creates a CSVExporter that writes to a temp folder with row limit
func getTemporaryCSVExporterWithMaxRowsLimit(maxRowsPerFile int) (*feed.CSVExporter, error) {
	dir, err := os.MkdirTemp("", "export")
	if err != nil {
		log.Fatal(err)
	}

	return feed.NewCSVExporter(dir, "", maxRowsPerFile)
}

var dateRegex = regexp.MustCompile(`(?m)(-?(?:[1-9][0-9]*)?[0-9]{4})-(1[0-2]|0[1-9])-(3[01]|0[1-9]|[12][0-9])T(2[0-3]|[01][0-9]):([0-5][0-9]):([0-5][0-9])(\.[0-9]+)?(\+|Z)(2[0-3]|[01][0-9])?:?([0-5][0-9])?`)

// getTestingSQLExporter creates a temporary DB on the target SQL Database
func getTestingSQLExporter() (*feed.SQLExporter, error) {
	dialect := os.Getenv("TEST_DB_DIALECT")
	connectionString := os.Getenv("TEST_DB_CONN_STRING")

	exporter, err := feed.NewSQLExporter(dialect, connectionString, true, "")
	if err != nil {
		return nil, err
	}

	dbName := strings.ReplaceAll(fmt.Sprintf("iaud_exporter_%s", uuid.Must(uuid.NewV4()).String()), "-", "")

	switch dialect {
	case "postgres", "mysql", "sqlserver":
		dbResp := exporter.DB.Exec(fmt.Sprintf(`CREATE DATABASE %s;`, dbName))
		err = dbResp.Error
	case "sqlite":
		return exporter, nil
	default:
		return nil, fmt.Errorf("invalid DB dialect %s", dialect)
	}
	if err != nil {
		return nil, err
	}

	connectionString = strings.Replace(connectionString, "safetyculture_exporter_db", dbName, 1)
	connectionString = strings.Replace(connectionString, "master", dbName, 1)

	return feed.NewSQLExporter(dialect, connectionString, true, "")
}

// getTemporaryCSVExporterWithRealSQLExporter creates a CSV exporter that writes a temporary folder
// but also uses a real DB as an intermediary
func getTemporaryCSVExporterWithRealSQLExporter(sqlExporter *feed.SQLExporter) (*feed.CSVExporter, error) {
	dir, err := os.MkdirTemp("", "export")
	if err != nil {
		return nil, err
	}

	exporter, err := feed.NewCSVExporter(dir, "", 100000)
	if err != nil {
		return nil, err
	}

	exporter.SQLExporter = sqlExporter

	return exporter, err
}

func TestClient_DrainDeletedInspections(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":4,"page_token":"","filters":{"timeframe":{"from":"2022-06-30T10:43:17Z"},"event_types":["inspection.deleted"],"limit":4}}`).
		Reply(http.StatusOK).
		File(path.Join("fixtures", "inspections_deleted_page_1.json"))

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":4,"page_token":"eyJldmVudF90eXBlcyI6WyJpbnNwZWN0aW9uLmFyY2hpdmVkIl0sImxpbWl0Ijo0LCJvZmZzZXQiOjR9","filters":{"timeframe":{"from":"2022-06-30T10:43:17Z"},"event_types":["inspection.deleted"],"limit":4}}`).
		Reply(http.StatusOK).
		File(path.Join("fixtures", "inspections_deleted_page_2.json"))

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":4,"page_token":"eyJldmVudF90eXBlcyI6WyJpbnNwZWN0aW9uLmFyY2hpdmVkIl0sImxpbWl0Ijo0LCJvZmZzZXQiOjh9","filters":{"timeframe":{"from":"2022-06-30T10:43:17Z"},"event_types":["inspection.deleted"],"limit":4}}`).
		Reply(http.StatusOK).
		File(path.Join("fixtures", "inspections_deleted_page_3.json"))

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":4,"page_token":"eyJldmVudF90eXBlcyI6WyJpbnNwZWN0aW9uLmFyY2hpdmVkIl0sImxpbWl0Ijo0LCJvZmZzZXQiOjEyfQ==","filters":{"timeframe":{"from":"2022-06-30T10:43:17Z"},"event_types":["inspection.deleted"],"limit":4}}`).
		Reply(http.StatusOK).
		File(path.Join("fixtures", "inspections_deleted_page_4.json"))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	fakeTime, err := time.Parse(time.RFC3339, "2022-06-30T10:43:17Z")
	require.Nil(t, err)
	req := httpapi.NewGetAccountsActivityLogRequest(4, fakeTime, []string{"inspection.deleted"})

	calls := 0
	var deletedIds = make([]string, 0, 15)
	fn := func(res *httpapi.GetAccountsActivityLogResponse) error {
		calls++
		for _, a := range res.Activities {
			deletedIds = append(deletedIds, a.Metadata["inspection_id"])
		}
		return nil
	}
	err = feed.DrainAccountActivityHistoryLog(context.TODO(), apiClient, req, fn)
	require.Nil(t, err)
	assert.EqualValues(t, 4, calls)
	require.EqualValues(t, 15, len(deletedIds))
	assert.EqualValues(t, "3b8ac4f4-e904-453e-b5a0-b5cceedb0ee1", deletedIds[0])
	assert.EqualValues(t, "4b3bc1d5-3011-4f81-94d4-125d2bce7ca8", deletedIds[1])
	assert.EqualValues(t, "6bd628a6-5188-425f-89ef-81f9dfcdf5cd", deletedIds[2])
	assert.EqualValues(t, "d722fc86-defa-4de2-b8d7-c0a3e0ec6ce4", deletedIds[3])
	assert.EqualValues(t, "ed8b3911-4141-41c4-946c-167bb6f61109", deletedIds[4])
	assert.EqualValues(t, "fd95cb4b-e1e7-488b-ba58-93fecd2379dc", deletedIds[5])
	assert.EqualValues(t, "1878c1e2-8a42-4f63-9e07-2e605f76762b", deletedIds[6])
	assert.EqualValues(t, "9e28ab2c-ce8c-44a7-81d3-76d0ac47dc91", deletedIds[7])
	assert.EqualValues(t, "48d61915-98c8-4d05-b786-4948dad199be", deletedIds[8])
	assert.EqualValues(t, "331727d2-4976-45da-857a-6d080dc645a9", deletedIds[9])
	assert.EqualValues(t, "1f2c9c1b-6f35-4bae-9b38-4094b40e13c1", deletedIds[10])
	assert.EqualValues(t, "35583d49-6421-40a8-a6f5-591c718c6025", deletedIds[11])
	assert.EqualValues(t, "eb49e9f8-4a3c-4b8f-a180-7ba0d171e93d", deletedIds[12])
	assert.EqualValues(t, "47ac0dce-16f9-4d73-b517-8372368af162", deletedIds[13])
	assert.EqualValues(t, "6d2f8bd5-a965-4046-b2b4-ccdf8341c9f0", deletedIds[14])
}

func TestClient_DrainDeletedInspections_WhenApiReturnsError(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Persist().
		Post("/accounts/history/v1/activity_log/list").
		Reply(http.StatusInternalServerError).
		JSON(`{"error": "something bad happened"}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	fakeTime, err := time.Parse(time.RFC3339, "2022-06-30T10:43:17Z")
	require.Nil(t, err)
	req := httpapi.NewGetAccountsActivityLogRequest(14, fakeTime, []string{"inspection.deleted"})
	fn := func(res *httpapi.GetAccountsActivityLogResponse) error {
		return nil
	}
	err = feed.DrainAccountActivityHistoryLog(context.TODO(), apiClient, req, fn)
	require.NotNil(t, err)
	assert.EqualValues(t, "unable to access Accounts Activity Logs: api request: http://localhost:9999/accounts/history/v1/activity_log/list giving up after 2 attempt(s)", err.Error())
}

func TestClient_DrainDeletedInspections_WhenFeedFnReturnsError(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":4,"page_token":"","filters":{"timeframe":{"from":"2022-06-30T10:43:17Z"},"event_types":["inspection.deleted"],"limit":4}}`).
		Reply(http.StatusOK).
		File(path.Join("fixtures", "inspections_deleted_page_1.json"))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	fakeTime, err := time.Parse(time.RFC3339, "2022-06-30T10:43:17Z")
	require.Nil(t, err)
	req := httpapi.NewGetAccountsActivityLogRequest(4, fakeTime, []string{"inspection.deleted"})

	fn := func(res *httpapi.GetAccountsActivityLogResponse) error {
		return fmt.Errorf("ERROR_GetAccountsActivityLogResponse")
	}
	err = feed.DrainAccountActivityHistoryLog(context.TODO(), apiClient, req, fn)
	require.NotNil(t, err)
	assert.EqualValues(t, "ERROR_GetAccountsActivityLogResponse", err.Error())
}

func TestAPIClientDrainFeed_should_return_for_as_long_next_page_set(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/feed/inspections").
		Reply(200).
		BodyString(`{
			"metadata": {
				"next_page": "/feed/inspections/next",
				"remaining_records": 0
			},
			"data": [
				{
					"id": "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
				},
				{
					"id": "audit_1743ae1aaa8741e6a23db83300e56efe"
				}
			]
		}`)

	gock.New("http://localhost:9999").
		Get("/feed/inspections/next").
		Reply(200).
		BodyString(`{
			"metadata": {
				"next_page": null,
				"remaining_records": 0
			},
			"data": [
				{
					"id": "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
				},
				{
					"id": "audit_abc"
				}
			]
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	calls := 0
	var auditIDs []string
	err := feed.DrainFeed(context.Background(), apiClient, &feed.GetFeedRequest{
		InitialURL: "/feed/inspections",
	}, func(data *feed.GetFeedResponse) error {
		calls += 1

		var rows []map[string]string
		err := json.Unmarshal(data.Data, &rows)
		assert.NoError(t, err)

		for _, row := range rows {
			auditIDs = append(auditIDs, row["id"])
		}

		return nil
	})
	assert.NoError(t, err)

	assert.Equal(t, 2, calls)
	assert.Equal(t, []string{
		"audit_8E2B1F3CB9C94D8792957F9F99E2E4BD",
		"audit_1743ae1aaa8741e6a23db83300e56efe",
		"audit_8E2B1F3CB9C94D8792957F9F99E2E4BD",
		"audit_abc",
	}, auditIDs)
}

func TestAPIClientDrainFeed_should_bubble_up_errors_from_callback(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/feed/inspections").
		Reply(200).
		BodyString(`{
			"metadata": {
				"next_page": "/feed/inspections/next",
				"remaining_records": 0
			},
			"data": []
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	expectedErr := errors.New("test error")
	err := feed.DrainFeed(context.Background(), apiClient, &feed.GetFeedRequest{
		InitialURL: "/feed/inspections",
	}, func(data *feed.GetFeedResponse) error {
		return expectedErr
	})
	assert.EqualValues(t, expectedErr.Error(), err.Error())
}

func TestClient_DrainFeed_WhenApiReturns403Error(t *testing.T) {
	defer gock.Off()
	//gock.Observe(gock.DumpRequest)

	gock.New("http://localhost:9999").
		Get("/feed/inspections").
		Reply(403).
		BodyString(`{"statusCode":403,"error":"Forbidden","message":"The caller does not have permission to execute the specified operation"}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	err := feed.DrainFeed(context.Background(), apiClient, &feed.GetFeedRequest{
		InitialURL: "/feed/inspections",
	}, func(data *feed.GetFeedResponse) error {
		return nil
	})
	assert.EqualValues(t, `{"status_code":403,"resource":"/feed/inspections","message":"{\"statusCode\":403,\"error\":\"Forbidden\",\"message\":\"The caller does not have permission to execute the specified operation\"}"}`, err.Error())
}

func TestAPIClientDrainFeed_should_return_api_errors(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/feed/inspections").
		Reply(500).
		JSON(`{"error": "something bad happened"}`)

	tests := []struct {
		name string
		cr   httpapi.CheckForRetry
		err  string
	}{
		{
			name: "default_retry_policy",
			cr:   httpapi.DefaultRetryPolicy,
			err:  "giving up after 2 attempt(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiClient := GetTestClient()
			gock.InterceptClient(apiClient.HTTPClient())

			err := feed.DrainFeed(context.Background(), apiClient, &feed.GetFeedRequest{
				InitialURL: "/feed/inspections",
			}, func(data *feed.GetFeedResponse) error {
				return nil
			})
			if err == nil || !strings.HasSuffix(err.Error(), tt.err) {
				t.Fatalf("expected giving up error, got: %#v", err)
			}
		})
	}
}
