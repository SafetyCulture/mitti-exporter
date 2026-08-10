package httpapi_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

// GetTestClient creates a new test apiClient
func GetTestClient(opts ...httpapi.Opt) *httpapi.Client {
	cfg := httpapi.ClientCfg{
		Addr:                "http://localhost:9999",
		AuthorizationHeader: "abc123",
		IntegrationID:       "mitti-exporter",
		IntegrationVersion:  "dev",
	}

	apiClient := httpapi.NewClient(&cfg, opts...)
	apiClient.RetryWaitMin = 10 * time.Millisecond
	apiClient.RetryWaitMax = 10 * time.Millisecond
	apiClient.CheckForRetry = httpapi.DefaultRetryPolicy
	apiClient.RetryMax = 1
	return apiClient
}

func TestApiOptSetTimeout_should_set_timeout(t *testing.T) {
	apiClient := GetTestClient(httpapi.OptSetTimeout(time.Second * 21))

	assert.Equal(t, time.Second*21, apiClient.HTTPClient().Timeout)
}

func TestClient_OptSetTimeout(t *testing.T) {
	cfg := httpapi.ClientCfg{
		Addr:                "fake_addr",
		AuthorizationHeader: "fake_token",
		IntegrationID:       "test",
		IntegrationVersion:  "dev",
	}

	client := httpapi.NewClient(&cfg)
	require.NotNil(t, client)

	opt := httpapi.OptSetTimeout(time.Second * 10)
	opt(client)
	require.NotNil(t, opt)
	assert.EqualValues(t, time.Second*10, client.HTTPClient().Timeout)
}

func TestClient_OptAddTLSCert_WhenEmptyPath(t *testing.T) {
	cfg := httpapi.ClientCfg{
		Addr:                "fake_addr",
		AuthorizationHeader: "fake_token",
		IntegrationID:       "test",
		IntegrationVersion:  "dev",
	}

	client := httpapi.NewClient(&cfg)
	require.NotNil(t, client)

	opt := httpapi.OptAddTLSCert("")
	opt(client)
	require.NotNil(t, opt)
	assert.Nil(t, client.HTTPTransport().TLSClientConfig)
}

func TestClient_OptSetProxy(t *testing.T) {
	cfg := httpapi.ClientCfg{
		Addr:                "fake_addr",
		AuthorizationHeader: "fake_token",
		IntegrationID:       "test",
		IntegrationVersion:  "dev",
	}

	client := httpapi.NewClient(&cfg)
	require.NotNil(t, client)

	u := url.URL{
		Scheme: "https",
		Host:   "fake.com",
	}
	opt := httpapi.OptSetProxy(&u)
	opt(client)

	require.NotNil(t, opt)
}

func TestClient_OptSetInsecureTLS_WhenTrue(t *testing.T) {
	cfg := httpapi.ClientCfg{
		Addr:                "fake_addr",
		AuthorizationHeader: "fake_token",
		IntegrationID:       "test",
		IntegrationVersion:  "dev",
	}

	client := httpapi.NewClient(&cfg)
	require.NotNil(t, client)

	opt := httpapi.OptSetInsecureTLS(true)
	opt(client)
	require.NotNil(t, opt)
	assert.True(t, client.HTTPTransport().TLSClientConfig.InsecureSkipVerify)
}

func TestClient_WhoAmI_WhenOK(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:9999").
		Get("accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`{}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	r, err := httpapi.WhoAmI(context.Background(), apiClient)
	require.Nil(t, err)
	require.NotNil(t, r)
}

func TestClient_WhoAmI_WhenNotOK(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)

	gock.New("http://localhost:9999").
		Persist().
		Get("accounts/user/v1/user:WhoAmI").
		Reply(500).
		BodyString(`{}`)

	apiClient := GetTestClient()
	apiClient.RetryMax = 3
	gock.InterceptClient(apiClient.HTTPClient())

	r, err := httpapi.WhoAmI(context.Background(), apiClient)
	require.NotNil(t, err)
	require.Nil(t, r)
	assert.EqualValues(t, "api request: http://localhost:9999/accounts/user/v1/user:WhoAmI giving up after 4 attempt(s)", err.Error())
}

func TestClient_WhoAmI_WhenNotOK_ContextCancelled(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)

	gock.New("http://localhost:9999").
		Persist().
		Get("accounts/user/v1/user:WhoAmI").
		Reply(200).
		BodyString(`{}`)

	apiClient := GetTestClient()
	apiClient.RetryMax = 3
	gock.InterceptClient(apiClient.HTTPClient())

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	r, err := httpapi.WhoAmI(ctx, apiClient)
	require.NotNil(t, err)
	require.Nil(t, r)
	assert.EqualValues(t, "api request: context canceled", err.Error())
}

func TestClient_HeadersShouldMatch(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)

	gock.New("http://localhost:9999").
		Get("accounts/user/v1/user:WhoAmI").
		MatchHeader("sc-integration-id", "mitti-exporter").
		MatchHeader("sc-integration-version", "dev").
		Reply(200).
		BodyString(`{}`)

	apiClient := GetTestClient()
	gock.InterceptClient(apiClient.HTTPClient())

	r, err := httpapi.WhoAmI(context.Background(), apiClient)
	require.Nil(t, err)
	require.NotNil(t, r)
}
