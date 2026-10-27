package update_test

import (
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/update"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_versionGreaterThanOrEqual(t *testing.T) {
	type args struct {
		v string
		w string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "when smaller",
			args: args{v: "0.0.0", w: "0.0.1"},
			want: false,
		},
		{
			name: "when greater",
			args: args{v: "0.0.2", w: "0.0.1"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := update.VersionGreaterThanOrEqual(tt.args.v, tt.args.w); got != tt.want {
				t.Errorf("versionGreaterThanOrEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapAssets_WhenEmpty(t *testing.T) {
	r := update.MapAssets([]github.ReleaseAsset{})
	assert.Empty(t, r)
}

func TestMapAssets_WhenAllArePresent(t *testing.T) {
	inputData := []github.ReleaseAsset{
		{
			Name:               pString("exporter-darwin-amd64.zip"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-darwin-amd64.zip"),
		},
		{
			Name:               pString("exporter-darwin-arm64.zip"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-darwin-arm64.zip"),
		},
		{
			Name:               pString("exporter-linux-amd64.tar.gz"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-linux-amd64.tar.gz"),
		},
		{
			Name:               pString("exporter-windows-x86_64.tar.gz"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-windows-x86_64.tar.gz"),
		},
	}

	r := update.MapAssets(inputData)
	require.EqualValues(t, 4, len(inputData))
	assert.EqualValues(t, r["darwin-amd64"], "https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-darwin-amd64.zip")
	assert.EqualValues(t, r["darwin-arm64"], "https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-darwin-arm64.zip")
	assert.EqualValues(t, r["linux-amd64"], "https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-linux-amd64.tar.gz")
	assert.EqualValues(t, r["windows-amd64"], "https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-windows-x86_64.tar.gz")
}

func TestMapAssets_WhenOnlyOneIsPresent(t *testing.T) {
	inputData := []github.ReleaseAsset{
		{
			Name:               pString("exporter-windows-x86_64.tar.gz"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-windows-x86_64.tar.gz"),
		},
	}

	r := update.MapAssets(inputData)
	require.EqualValues(t, 1, len(inputData))
	assert.EqualValues(t, r["windows-amd64"], "https://github.com/SafetyCulture/mitti-exporter-ui/releases/download/v.0.10.2-alpha.5/exporter-windows-x86_64.tar.gz")
}

func TestMapUIAssets_WhenAllArePresent(t *testing.T) {
	inputData := []github.ReleaseAsset{
		{
			Name:               pString("mitti-exporter-darwin-amd64.zip"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-darwin-amd64.zip"),
		},
		{
			Name:               pString("mitti-exporter-darwin-arm64.zip"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-darwin-arm64.zip"),
		},
		{
			Name:               pString("mitti-exporter-ui-darwin-universal.zip"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-ui-darwin-universal.zip"),
		},
		{
			Name:               pString("mitti-exporter-ui-windows-x86_64.exe"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-ui-windows-x86_64.exe"),
		},
		{
			Name:               pString("mitti-exporter-windows-x86_64.zip"),
			BrowserDownloadURL: pString("https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-windows-x86_64.zip"),
		},
	}

	r := update.MapUIAssets(inputData)
	require.EqualValues(t, 2, len(r))
	// the universal installer, not the CLI darwin builds, is selected for macOS
	assert.EqualValues(t, r["darwin"], "https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-ui-darwin-universal.zip")
	// the UI executable, not the CLI windows zip, is selected for windows
	assert.EqualValues(t, r["windows"], "https://github.com/SafetyCulture/mitti-exporter/releases/download/v0.10.2/mitti-exporter-ui-windows-x86_64.exe")
}

func TestMapUIAssets_WhenEmpty(t *testing.T) {
	r := update.MapUIAssets([]github.ReleaseAsset{})
	assert.Empty(t, r)
}

func pString(s string) *string {
	return &s
}
