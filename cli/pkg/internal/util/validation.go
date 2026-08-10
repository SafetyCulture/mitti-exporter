package util

import (
	"net/http"

	"github.com/SafetyCulture/mitti-exporter/pkg/logger"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var lgr *zap.SugaredLogger

// Check the error and exit the process when not nil
func Check(err error, msg string) {
	if lgr == nil {
		lgr = logger.GetLogger()
	}

	if err != nil {
		lgr.Fatal(errors.Wrapf(err, msg))
	}
}

// CheckFeedError - checks the Feed for errors, except for 403's
func CheckFeedError(logger *zap.SugaredLogger, err error, msg string) {
	if err == nil {
		return
	}

	switch e := err.(type) {
	case HTTPError:
		if e.StatusCode == http.StatusForbidden {
			logger.Error(errors.Wrapf(err, msg))
			return
		}
	}
	logger.Fatal(errors.Wrapf(err, msg))
}
