package api_test

import (
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func TestMittiExporter_GetTemplateList(t *testing.T) {
	cfg := api.ExporterConfiguration{}

	apiClient := GetTestClient()
	defer gock.Off()
	//gock.Observe(gock.DumpRequest)
	gock.InterceptClient(apiClient.HTTPClient())

	gock.New("http://localhost:9999").
		Get("/templates/v1/templates/search").
		MatchParam("page_size", "1000").
		Reply(200).
		BodyString(`
			{
			  "next_page_token": "",
			  "items": [
				{
				  "id": "template_8ddf395bfd484e978d31d3afbdae3863",
				  "name": "T7",
				  "modified_at": "2022-12-19T02:31:50.792Z"
				}
			  ]
			}
		`)

	exporter, err := api.NewMittiExporter(&cfg, &api.AppVersion{})
	require.Nil(t, err)
	exporter.SetApiClient(apiClient)
	exporter.SetSheqsyApiClient(apiClient)

	res := exporter.GetTemplateList()
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, "template_8ddf395bfd484e978d31d3afbdae3863", res[0].ID)
	assert.EqualValues(t, "T7", res[0].Name)
	assert.True(t, time.Date(2022, 12, 19, 2, 31, 50, 792000000, time.UTC).Equal(res[0].ModifiedAt))
}
