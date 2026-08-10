package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/stretchr/testify/require"

	exportermock "github.com/SafetyCulture/mitti-exporter/pkg/internal/exporter/mocks"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/inspections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/h2non/gock.v1"
)

func initMockInspections(httpClient *http.Client) {
	gock.InterceptClient(httpClient)

	gock.New("http://localhost:9999").
		Get("/audits/search").
		Reply(200).
		File("mocks/inspections.json")

	gock.New("http://localhost:9999").
		Get("/audits/audit_d7e2f55b95094bd48fac601850e1db63").
		Reply(200).
		File("mocks/audits/audit_d7e2f55b95094bd48fac601850e1db63.json")

	gock.New("http://localhost:9999").
		Get("/audits/audit_872d26e479924353b957b832cd98a7f6").
		Reply(200).
		File("mocks/audits/audit_872d26e479924353b957b832cd98a7f6.json")

	gock.New("http://localhost:9999").
		Get("/audits/audit_7f1ecd2e66474418b1e062657aeb1389").
		Reply(200).
		File("mocks/audits/audit_7f1ecd2e66474418b1e062657aeb1389.json")
}

func TestInspectionsExport(t *testing.T) {

	apiClient := GetTestClient()
	initMockInspections(apiClient.HTTPClient())

	exporterMock := new(exportermock.Exporter)
	exporterMock.On("WriteRow", mock.Anything, mock.Anything)
	exporterMock.On("SetLastModifiedAt", mock.Anything)
	exporterMock.On("GetLastModifiedAt", mock.Anything).Return(nil)

	exporterAppCfg := api.BuildConfigurationWithDefaults()
	inspectionClient := inspections.NewInspectionClient(exporterAppCfg.ToInspectionConfig(), apiClient, exporterMock)
	err := inspectionClient.Export(context.Background())
	assert.NoError(t, err)
}

func TestInspectionsExport_WhenSkipID(t *testing.T) {
	apiClient := GetTestClient()
	initMockInspections(apiClient.HTTPClient())

	exporterMock := new(exportermock.Exporter)
	exporterMock.On("WriteRow", mock.Anything, mock.Anything)
	exporterMock.On("SetLastModifiedAt", mock.Anything)
	exporterMock.On("GetLastModifiedAt", mock.Anything).Return(nil)

	exporterAppCfg := api.BuildConfigurationWithDefaults()
	inspectionClient := inspections.NewInspectionClient(exporterAppCfg.ToInspectionConfig(), apiClient, exporterMock)
	inspectionClient.(*inspections.Client).SkipIDs = []string{"audit_d7e2f55b95094bd48fac601850e1db63"}
	err := inspectionClient.Export(context.Background())
	assert.NoError(t, err)
}

func TestInspectionsExport_WhenModifiedAtIsNotNil(t *testing.T) {
	apiClient := GetTestClient()
	initMockInspections(apiClient.HTTPClient())

	exporterMock := new(exportermock.Exporter)
	exporterMock.On("WriteRow", mock.Anything, mock.Anything)
	exporterMock.On("SetLastModifiedAt", mock.Anything)
	exporterMock.On("GetLastModifiedAt", mock.Anything).Return(&time.Time{})

	exporterAppCfg := api.BuildConfigurationWithDefaults()
	inspectionClient := inspections.NewInspectionClient(exporterAppCfg.ToInspectionConfig(), apiClient, exporterMock)
	err := inspectionClient.Export(context.Background())
	assert.NoError(t, err)
}

func TestNewInspectionClient(t *testing.T) {
	exporterAppCfg := api.BuildConfigurationWithDefaults()
	res := inspections.NewInspectionClient(exporterAppCfg.ToInspectionConfig(), nil, nil)
	require.NotNil(t, res)

	client, ok := res.(*inspections.Client)
	require.True(t, ok)
	require.NotNil(t, client)

	assert.EqualValues(t, "inspections", client.Name())
}
