package feed

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
)

// GetMedia fetches the media object from SafetyCulture.
func GetMedia(ctx context.Context, apiClient *httpapi.Client, request *GetMediaRequest) (*GetMediaResponse, error) {
	baseURL := strings.TrimPrefix(request.URL, apiClient.BaseURL)

	// The mediaURL will be in the following format:
	// https://api.eu.safetyculture.com/audits/audit_xxx/media/4c83fcf2-180b-4d3e-958f-389f7ac49777
	// The string that is after the word "media/" is the ID of it.
	mediaIDURL := strings.Split(request.URL, "/")
	mediaID := mediaIDURL[len(mediaIDURL)-1]

	httpRes, err := httpapi.ExecuteRawGet(ctx, apiClient, baseURL)
	if err != nil {
		return nil, err
	}

	defer httpRes.Body.Close()

	if httpRes.StatusCode == 204 {
		return nil, nil
	}

	contentType := httpRes.Header.Get("Content-Type")
	if contentType == "" {
		return nil, fmt.Errorf("failed to get content-type of media")
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(httpRes.Body)

	resp := &GetMediaResponse{
		ContentType: contentType,
		Body:        buf.Bytes(),
		MediaID:     mediaID,
	}
	return resp, nil
}

// GetMediaRequest has all the data needed to make a request to get a media
type GetMediaRequest struct {
	URL     string
	AuditID string
}

// GetMediaResponse is a representation of the data returned when fetching media
type GetMediaResponse struct {
	ContentType string
	Body        []byte
	MediaID     string
}
