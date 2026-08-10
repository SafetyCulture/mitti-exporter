// Copyright (c) 2020 SafetyCulture Pty Ltd. All Rights Reserved.

package update

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
)

const RepoExporter string = "mitti-exporter"

const archDarwinAmd64 string = "darwin-amd64"
const archDarwinArm64 string = "darwin-arm64"
const archLinuxAmd64 string = "linux-amd64"
const archWindows string = "windows-amd64"

// ReleaseInfo is the details of an available release.
type ReleaseInfo struct {
	Version      string
	ChangelogURL string
	DownloadURL  string
}

// Check returns release info of a new version of this tool if available.
func Check(currentVersion string, repoName string) *ReleaseInfo {
	ctx := context.Background()
	g := github.NewClient(&http.Client{})
	res, _, err := g.Repositories.GetLatestRelease(ctx, "SafetyCulture", repoName)
	if err != nil {
		return nil
	}

	v := res.GetTagName()
	if VersionGreaterThanOrEqual(currentVersion, v) {
		return nil
	}

	filteredAssets := MapAssets(res.Assets)
	dURL, ok := filteredAssets[fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)]
	if !ok {
		dURL = ""
	}

	return &ReleaseInfo{
		Version:      v,
		ChangelogURL: res.GetHTMLURL(),
		DownloadURL:  dURL,
	}
}

// VersionGreaterThanOrEqual returns true if version is greater or equal
func VersionGreaterThanOrEqual(v, w string) bool {
	vv, ve := version.NewVersion(v)
	vw, we := version.NewVersion(w)

	return ve == nil && we == nil && vv.GreaterThanOrEqual(vw)
}

// MapAssets Maps name to browser_download_url
func MapAssets(assets []github.ReleaseAsset) map[string]string {
	if assets == nil {
		return map[string]string{}
	}

	var result = make(map[string]string, 4)
	for _, asset := range assets {
		switch {
		case strings.Contains(*asset.Name, archDarwinAmd64):
			result[archDarwinAmd64] = *asset.BrowserDownloadURL
		case strings.Contains(*asset.Name, archDarwinArm64):
			result[archDarwinArm64] = *asset.BrowserDownloadURL
		case strings.Contains(*asset.Name, archLinuxAmd64):
			result[archLinuxAmd64] = *asset.BrowserDownloadURL
		case strings.Contains(*asset.Name, "windows"):
			result[archWindows] = *asset.BrowserDownloadURL
		}
	}
	return result
}
