package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestAPIClientGetInspection(t *testing.T) {
	defer gock.Off()

	auditID := "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
	gock.New("http://localhost:9999").
		Get(fmt.Sprintf("/audits/%s", auditID)).
		Reply(200).
		BodyString(`{
			"audit_id": "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
		}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	resp, err := httpapi.GetRawInspection(context.Background(), apiClient, auditID)
	assert.NoError(t, err)

	rows := map[string]string{}
	err = json.Unmarshal(*resp, &rows)
	assert.NoError(t, err)

	expected, ok := rows["audit_id"]
	assert.Equal(t, true, ok)
	assert.Equal(t, expected, auditID)
}

func TestAPIClientGetInspectionWithError(t *testing.T) {
	defer gock.Off()

	auditID := "audit_8E2B1F3CB9C94D8792957F9F99E2E4BD"
	gock.New("http://localhost:9999").
		Get(fmt.Sprintf("/audits/%s", auditID)).
		ReplyError(fmt.Errorf("test error"))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := httpapi.GetRawInspection(context.Background(), apiClient, auditID)
	assert.NotNil(t, err)
}

func TestAPIClientListInspectionWithError(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("/audits/search").
		ReplyError(fmt.Errorf("test error"))

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	_, err := httpapi.ListInspections(context.Background(), apiClient, nil)
	assert.NotNil(t, err)
}

func TestAPIClientBackoff429TooManyRequest(t *testing.T) {
	defer gock.Off()

	req := gock.New("http://localhost:9999").
		Get(fmt.Sprintf("/audits/%s", "1234")).
		Reply(429)
	req.SetHeader("X-RateLimit-Reset", "1")

	tests := []struct {
		name string
		bo   httpapi.Backoff
		err  string
	}{
		{
			name: "default_backoff_policy",
			bo:   httpapi.DefaultBackoff,
			err:  "giving up after 2 attempt(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiClient := GetTestClient()
			gock.InterceptClient(apiClient.HTTPClient())
			apiClient.RetryMax = 1

			_, err := httpapi.GetRawInspection(context.Background(), apiClient, "1234")
			if err == nil || !strings.HasSuffix(err.Error(), tt.err) {
				t.Fatalf("expected giving up error, got: %#v", err)
			}
		})
	}
}
