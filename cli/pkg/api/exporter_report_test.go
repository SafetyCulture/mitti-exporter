package api_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	exporterAPI "github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/report"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

const (
	mockAPIBaseURL            string = "http://localhost:9999"
	initiateReportURL         string = "/audits/.*/report"
	reportExportCompletionURL string = "/audits/.*/report/.*"
	downloadReportURL         string = "/report-exports/abc"
)

func getReportExportCompletionMessage(status string) string {
	return fmt.Sprintf(`{"status": "%s", "url": "%s%s"}`, status, mockAPIBaseURL, downloadReportURL)
}

func TestExportReports_should_export_all_reports(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryReportExporter([]string{"PDF", "WORD"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(6).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(6).
		Reply(200).
		JSON(getReportExportCompletionMessage("SUCCESS"))

	gock.New(mockAPIBaseURL).
		Get(downloadReportURL).
		Times(6).
		Reply(200).
		Body(bytes.NewBuffer([]byte(`file content`)))

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NoError(t, err)

	fileExists(t, filepath.Join(exporter.ExportPath, "My-Audit.pdf"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4e28ab2cce8c44a781d376d0ac47dc92.pdf"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4d95cb4be1e7488bba5893fecd2379d2.pdf"))

	fileExists(t, filepath.Join(exporter.ExportPath, "My-Audit.docx"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4e28ab2cce8c44a781d376d0ac47dc92.docx"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4d95cb4be1e7488bba5893fecd2379d2.docx"))
}

func TestExportReports_should_export_all_reports_with_ID_filename(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryReportExporter([]string{"PDF", "WORD"}, "", "INSPECTION_ID")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(6).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(6).
		Reply(200).
		JSON(getReportExportCompletionMessage("SUCCESS"))

	gock.New(mockAPIBaseURL).
		Get(downloadReportURL).
		Times(6).
		Reply(200).
		Body(bytes.NewBuffer([]byte(`file content`)))

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NoError(t, err)

	fileExists(t, filepath.Join(exporter.ExportPath, "audit_47ac0dce16f94d73b5178372368af162.pdf"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4e28ab2cce8c44a781d376d0ac47dc92.pdf"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4d95cb4be1e7488bba5893fecd2379d2.pdf"))

	fileExists(t, filepath.Join(exporter.ExportPath, "audit_47ac0dce16f94d73b5178372368af162.docx"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4e28ab2cce8c44a781d376d0ac47dc92.docx"))
	fileExists(t, filepath.Join(exporter.ExportPath, "audit_4d95cb4be1e7488bba5893fecd2379d2.docx"))
}

func TestExportReports_should_not_run_if_all_exported(t *testing.T) {
	defer gock.Off()

	exporter, err := getTemporaryReportExporter([]string{"PDF"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	// Making sure the endpoints have been called only 3 times
	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Times(3).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"2014-03-17T00:35:40Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		BodyString(`{"activites": []}`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(3).
		Reply(200).
		JSON(getReportExportCompletionMessage("SUCCESS"))

	gock.New(mockAPIBaseURL).
		Get(downloadReportURL).
		Times(3).
		Reply(200).
		Body(bytes.NewBuffer([]byte(`file content`)))

	cfg := &exporterAPI.ExporterConfiguration{}
	cfg.Export.Incremental = true

	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NoError(t, err)

	file1ModTime1, _ := getFileModTime(filepath.Join(exporter.ExportPath, "My-Audit.pdf"))
	file2ModTime1, _ := getFileModTime(filepath.Join(exporter.ExportPath, "audit_2.pdf"))
	file3ModTime1, _ := getFileModTime(filepath.Join(exporter.ExportPath, "audit_3.pdf"))

	// run the export process again
	initMockFeedsSet1(apiClient.HTTPClient())
	exporterApp = feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NoError(t, err)

	file1ModTime2, _ := getFileModTime(filepath.Join(exporter.ExportPath, "My-Audit.pdf"))
	file2ModTime2, _ := getFileModTime(filepath.Join(exporter.ExportPath, "audit_2.pdf"))
	file3ModTime2, _ := getFileModTime(filepath.Join(exporter.ExportPath, "audit_3.pdf"))

	assert.Equal(t, file1ModTime1, file1ModTime2)
	assert.Equal(t, file2ModTime1, file2ModTime2)
	assert.Equal(t, file3ModTime1, file3ModTime2)
}

func TestExportReports_should_take_care_of_invalid_file_names(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"PDF"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New("http://localhost:9999").
		Get("/feed/inspections").
		Reply(200).
		File("mocks/set_4/feed_inspections_1.json")

	// Making sure the endpoints have been called only 3 times
	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(3).
		Reply(200).
		JSON(getReportExportCompletionMessage("SUCCESS"))

	gock.New(mockAPIBaseURL).
		Get(downloadReportURL).
		Times(3).
		Reply(200).
		Body(bytes.NewBuffer([]byte(`file content`)))

	gock.New("http://localhost:9999").
		Post("/accounts/history/v1/activity_log/list").
		BodyString(`{"org_id":"","page_size":0,"page_token":"","filters":{"timeframe":{"from":"0001-01-01T00:00:00Z"},"event_types":["inspection.deleted"],"limit":0}}`).
		Reply(http.StatusOK).
		BodyString(`{"activites": []}`)

	cfg := &exporterAPI.ExporterConfiguration{}
	cfg.Export.Incremental = true

	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NoError(t, err)

	fileExists(t, filepath.Join(exporter.ExportPath, "My-Audit-1.pdf"))
	fileExists(t, filepath.Join(exporter.ExportPath, "My-Audit-1 (1).pdf"))
	var files []string
	filepath.Walk(exporter.ExportPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		assert.LessOrEqual(t, len(path), 255)
		return nil
	})
	assert.Equal(t, 5, len(files))
}

func TestExportReports_should_fail_after_retries(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"PDF"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	// Making sure the endpoints is called 15 times for each inspection
	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(45).
		Reply(200).
		JSON(getReportExportCompletionMessage("IN_PROGRESS"))

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to generate 3 PDF reports and 0 WORD reports")
}

func TestExportReports_should_fail_if_report_status_fails(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"WORD"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Times(3).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(3).
		Reply(200).
		JSON(getReportExportCompletionMessage("FAILED"))

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to generate 0 PDF reports and 3 WORD reports")
}

func TestExportReports_should_fail_if_init_report_reply_is_not_success(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"WORD"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(500).
		JSON(`{"error": "something went wrong"}`)

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to generate 0 PDF reports and 3 WORD reports")
}

func TestExportReports_should_fail_if_report_completion_reply_is_not_success(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"WORD"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Times(3).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(3).
		Reply(500).
		JSON(`{"error": "something went wrong"}`)

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to generate 0 PDF reports and 3 WORD reports")
}

func TestExportReports_should_fail_if_download_report_reply_is_not_success(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"PDF"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	defer resetMocks(apiClient.HTTPClient())
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Times(3).
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	gock.New(mockAPIBaseURL).
		Post(initiateReportURL).
		Times(3).
		Reply(200).
		JSON(`{"messageId": "abc"}`)

	gock.New(mockAPIBaseURL).
		Get(reportExportCompletionURL).
		Times(3).
		Reply(200).
		JSON(getReportExportCompletionMessage("SUCCESS"))

	gock.New(mockAPIBaseURL).
		Get(downloadReportURL).
		Times(3).
		Reply(500).
		JSON(`{"error": "something went wrong"}`)

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to generate 3 PDF reports and 0 WORD reports")
}

func TestExportReports_should_return_error_for_unsupported_format(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"PNG"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	apiClient := GetTestClient()
	initMockFeedsSet1(apiClient.HTTPClient())

	gock.New(mockAPIBaseURL).
		Get("/accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`
		{
			"user_id": "user_123",
			"organisation_id": "role_123",
			"firstname": "Test",
			"lastname": "Test"
		  }
		`)

	cfg := &exporterAPI.ExporterConfiguration{}
	exporterApp := feed.NewExporterApp(apiClient, nil, cfg.ToExporterConfig())
	err = exporterApp.ExportInspectionReports(exporter, context.Background())
	assert.EqualError(t, err, "save reports: no valid export format specified")
}

func TestNewReportExporter_should_set_busy_timeout(t *testing.T) {
	exporter, err := getTemporaryReportExporter([]string{"PDF"}, "", "INSPECTION_TITLE")
	assert.NoError(t, err)

	var busyTimeout int
	err = exporter.DB.Raw("PRAGMA busy_timeout").Scan(&busyTimeout).Error
	assert.NoError(t, err)
	assert.Equal(t, 20000, busyTimeout)
}

func Test_GetWaitTime(t *testing.T) {
	assert.Equal(t, time.Duration(1), feed.GetWaitTime(0))
	assert.Equal(t, time.Duration(1), feed.GetWaitTime(10))
	assert.Equal(t, time.Duration(2), feed.GetWaitTime(30))
	assert.Equal(t, time.Duration(3), feed.GetWaitTime(50))
	assert.Equal(t, time.Duration(4), feed.GetWaitTime(60))
	assert.Equal(t, time.Duration(4), feed.GetWaitTime(90))
}

func TestAPIClientInitiateInspectionReportExport_should_return_messageID(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Post("/audits/audit_123/report").
		JSON(`{
			"format": "PDF",
			"preference_id": "p123"
		}`).
		Reply(200).
		JSON(`{
			"messageId": "abc"
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	mId, err := report.InitiateInspectionReportExport(context.Background(), apiClient, "audit_123", "PDF", "p123")

	assert.NoError(t, err)
	assert.Equal(t, "abc", mId)
}

func TestAPIClientInitiateInspectionReportExport_should_return_error_on_failure(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Post("/audits/audit_123/report").
		JSON(`{"format": "PDF"}`).
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

			_, err := report.InitiateInspectionReportExport(context.Background(), apiClient, "audit_123", "PDF", "")
			if err == nil || !strings.HasSuffix(err.Error(), tt.err) {
				t.Fatalf("expected giving up error, got: %#v", err)
			}
		})
	}
}

func TestAPIClientCheckInspectionReportExportCompletion_should_return_status(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/audit_123/report/abc").
		Reply(200).
		JSON(`{
			"status": "SUCCESS",
			"url": "http://domain.com/report"
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	res, err := httpapi.CheckInspectionReportExportCompletion(context.Background(), apiClient, "audit_123", "abc")

	assert.NoError(t, err)
	assert.Equal(t, res.Status, "SUCCESS")
	assert.Equal(t, res.URL, "http://domain.com/report")
}

func TestAPIClientCheckInspectionReportExportCompletion_should_return_error_on_failure(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/audit_123/report/abc").
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

			_, err := httpapi.CheckInspectionReportExportCompletion(context.Background(), apiClient, "audit_123", "abc")
			if err == nil || !strings.HasSuffix(err.Error(), tt.err) {
				t.Fatalf("expected giving up error, got: %#v", err)
			}
		})
	}
}

func TestAPIClientDownloadInspectionReportFile_should_return_status(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/report-exports/abc").
		Reply(200).
		Body(bytes.NewBuffer([]byte(`file content`)))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	res, err := httpapi.ExecuteRawGet(context.Background(), apiClient, "http://localhost:9999/report-exports/abc")

	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	assert.Equal(t, buf.String(), "file content")
}

func TestAPIClientDownloadInspectionReportFile_should_return_error_on_failure(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/report-exports/abc").
		Reply(500).
		BodyString("somthing bad happened")

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

			_, err := httpapi.ExecuteRawGet(context.Background(), apiClient, "http://localhost:9999/report-exports/abc")
			if err == nil || !strings.HasSuffix(err.Error(), tt.err) {
				t.Fatalf("expected giving up error, got: %#v", err)
			}
		})
	}
}
