package templates

import (
	"context"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/logger"
)

// Client to be used with inspections
type Client struct {
	apiClient *httpapi.Client
}

func NewTemplatesClient(apiClient *httpapi.Client) *Client {
	return &Client{
		apiClient: apiClient,
	}
}

// GetTemplateList will gracefully return template list. On Error will return empty or what was temporarily downloaded
func (c *Client) GetTemplateList(ctx context.Context, pageSize int) []TemplateResponseItem {
	l := logger.GetLogger()
	var items []TemplateResponseItem
	callback := func(resp *listTemplatesResponse) {
		items = append(items, resp.Templates...)
	}

	// gracefully return what was downloaded
	if err := c.drainTemplates(ctx, pageSize, callback); err != nil {
		l.Error(err)
		return items
	}
	return items
}

// DrainTemplates will process a paginated response and collect all responses before returning the response
func (c *Client) drainTemplates(ctx context.Context, pageSize int, callbackFn func(response *listTemplatesResponse)) error {
	nextToken := ""

	for {
		params := &templateSearchRequest{
			PageSize: pageSize,
			Token:    nextToken,
		}
		resp, err := httpapi.ExecuteGet[listTemplatesResponse](ctx, c.apiClient, "/templates/v1/templates/search", params)

		if err != nil {
			return err
		}

		callbackFn(resp)

		if resp.NextPageToken == "" {
			// no more pages left
			break
		}
		nextToken = resp.NextPageToken
	}
	return nil
}

// listTemplatesResponse list of templates
type listTemplatesResponse struct {
	NextPageToken string                 `json:"next_page_token"`
	Templates     []TemplateResponseItem `json:"items"`
}

// TemplateResponseItem simple representation of a template date
type TemplateResponseItem struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
}

// templateSearchRequest contains parameters for calling api-templates template search
type templateSearchRequest struct {
	PageSize int    `url:"page_size"`
	Token    string `url:"page_token,omitempty"`
}
