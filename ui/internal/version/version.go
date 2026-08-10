package version

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	vv "github.com/hashicorp/go-version"
	"github.com/minio/selfupdate"
)

// This variable should be overridden at build time using ldflags.
var version string = "v0.0.0-dev"

const integrationID string = "mitti-exporter-ui"

// GetVersion returns current version of the app
func GetVersion() string {
	return version
}

// GetIntegrationID returns the integration id
func GetIntegrationID() string {
	return integrationID
}

// ShouldUpdate compares 2 versions (semver)
// return true if there is a major or 2 minor differences
func ShouldUpdate(current string, new string) bool {
	// if new version is malformed, should not update
	newVer, err := vv.NewVersion(new)
	if err != nil {
		return false
	}

	// don't update for new pre-releases
	if len(newVer.Prerelease()) > 0 {
		return false
	}

	// if current version is malformed, should update if the new version is different
	currentVer, err := vv.NewSemver(current)
	if err != nil {
		return new != current
	}

	// validate
	if len(newVer.Segments()) != 3 || len(currentVer.Segments()) != 3 {
		return false
	}

	// calculate diff in versions
	maj := newVer.Segments()[0] - currentVer.Segments()[0]
	min := newVer.Segments()[1] - currentVer.Segments()[1]
	patch := newVer.Segments()[2] - currentVer.Segments()[2]

	// current is prerelease, we will update only if there is a bigger version
	if len(currentVer.Prerelease()) > 0 {
		if isEqual(maj, min, patch) {
			return true
		}
		return isBigger(maj, min, patch)
	}

	// if they are equal don't update
	if isEqual(maj, min, patch) {
		return false
	}

	// error case when current major version is newer than the one available for download (unlikely)
	if maj < 0 {
		return false
	}

	// if there is a major version difference, we force the update
	if maj >= 1 {
		return true
	}

	// if there are 2 minor versions difference, we force the update
	if min >= 2 {
		return true
	}

	return false
}

func isBigger(maj, min, patch int) bool {
	switch {
	case maj > 0:
		return true
	case maj < 0:
		return false

	case min > 0:
		return true
	case min < 0:
		return false

	case patch > 0:
		return true
	case patch < 0:
		return false

	default:
		return false
	}
}

func isEqual(maj, min, patch int) bool {
	return maj == 0 && min == 0 && patch == 0
}

func DoUpdate(url string) error {
	var fReaderCloser io.ReadCloser
	var err error

	switch {
	case strings.HasSuffix(url, ".zip"):
		fReaderCloser, err = readZipFile(url)
		if err != nil {
			return err
		}
		defer fReaderCloser.Close()
	case strings.HasSuffix(url, ".exe"):
		fReaderCloser, err = getFileContentsFromURL(url)
		if err != nil {
			return err
		}
		defer fReaderCloser.Close()
	default:
		return fmt.Errorf("unrecognizable file extention for %v", url)
	}

	err = selfupdate.Apply(fReaderCloser, selfupdate.Options{})
	if err != nil {
		return err
	}
	return nil
}

func readZipFile(url string) (io.ReadCloser, error) {
	urlReaderCloser, err := getFileContentsFromURL(url)
	if err != nil {
		return nil, err
	}
	defer urlReaderCloser.Close()

	content, err := io.ReadAll(urlReaderCloser)
	if err != nil {
		return nil, err
	}

	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}

	var fReaderCloser io.ReadCloser
	var search string
	switch runtime.GOOS {
	case "darwin":
		search = "Mitti-Exporter.app/Contents/MacOS/Mitti-Exporter"
	case "windows":
		search = "build/bin/mitti-exporter.exe"
	default:
		return nil, fmt.Errorf("current architecture is not supported")
	}

	for _, f := range archive.File {
		if f.Name == search {
			fReaderCloser, err = f.Open()
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if fReaderCloser == nil {
		return nil, fmt.Errorf("could not find the binary in the provided archive")
	}
	return fReaderCloser, nil
}

// getFileContentsFromURL will return an io.ReadCloser for the given URL or error
func getFileContentsFromURL(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received %s status for %s", resp.Status, url)
	}

	return resp.Body, nil
}
