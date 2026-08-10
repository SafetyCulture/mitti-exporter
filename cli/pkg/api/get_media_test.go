package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestGetMediaWithAPIError(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		ReplyError(fmt.Errorf("test error"))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NotNil(t, err)
}

func TestGetMediaWith403Error(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		Reply(403).
		JSON(`{"error": "something bad happened"}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NoError(t, err)
}

func TestGetMediaWith404Error(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		Reply(404).
		JSON(`{"error": "something bad happened"}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NoError(t, err)
}

func TestGetMediaWith405Error(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		Reply(405).
		JSON(`{"error": "something bad happened"}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NotNil(t, err)
}

func TestGetMediaWith204Error(t *testing.T) {
	defer gock.Off()

	result := `{id:"test-id"}`
	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		Reply(204).
		BodyString(result)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	resp, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

func TestGetMediaWithNoContentType(t *testing.T) {
	defer gock.Off()

	result := `{id:"test-id"}`
	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		Reply(200).
		BodyString(result)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NotNil(t, err)
}

func TestGetMedia(t *testing.T) {
	defer gock.Off()

	result := `{id:"test-id"}`
	gock.New("http://localhost:9999").
		Get("/audits/1234/media/12345").
		Reply(200).
		BodyString(result).
		SetHeader("Content-Type", "test-content")

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	expected := &feed.GetMediaResponse{
		ContentType: "test-content",
		Body:        []byte(result),
		MediaID:     "12345",
	}
	resp, err := feed.GetMedia(
		context.Background(),
		apiClient,
		&feed.GetMediaRequest{
			URL:     "http://localhost:9999/audits/1234/media/12345",
			AuditID: "1234",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, resp, expected)
}
