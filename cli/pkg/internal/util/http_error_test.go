package util_test

import (
	"net/http"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestHTTPError_Error(t *testing.T) {
	e := util.HTTPError{
		StatusCode: http.StatusBadRequest,
		Resource:   "/test",
		Message:    "something went wrong",
	}

	assert.EqualValues(t, `{"status_code":400,"resource":"/test","message":"something went wrong"}`, e.Error())
}
