package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	osRuntime "runtime"
	"time"

	"github.com/SafetyCulture/mitti-exporter-ui/internal/version"
	exporterAPI "github.com/SafetyCulture/mitti-exporter/pkg/api"
	"github.com/SafetyCulture/mitti-exporter/pkg/httpapi"
	"github.com/SafetyCulture/mitti-exporter/pkg/update"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// gitRepoExporter is the consolidated monorepo that publishes both CLI and UI
// release assets under a single GitHub Release (see .github/workflows/release.yml).
const gitRepoExporter string = "mitti-exporter"

// App struct
type App struct {
	ctx      context.Context
	cm       *exporterAPI.ConfigurationManager
	exporter *exporterAPI.MittiExporter
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so, we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	settingsDir, err := GetSettingDirectoryPath()
	if err != nil {
		runtime.LogError(ctx, "failed to get settings directory")
		panic("failed to get settings directory")
	}

	configFile := resolveConfigFileName(settingsDir)

	if !checkForConfigFile(settingsDir, configFile) {
		runtime.LogInfof(ctx, "creating configuration file: %s/%s", settingsDir, configFile)
		cm := exporterAPI.NewConfigurationManager(settingsDir, configFile)
		if err := cm.SaveConfiguration(); err != nil {
			runtime.LogError(ctx, err.Error())
			panic("failed to save configuration")
		}
		a.cm = cm
	} else {
		runtime.LogInfof(ctx, "loading configuration file: %s/%s", settingsDir, configFile)
		cm, err := exporterAPI.NewConfigurationManagerFromFile(settingsDir, configFile)
		if err != nil {
			runtime.LogError(ctx, err.Error())
			panic("failed to load configuration")
		}
		a.cm = cm
	}

	ver := exporterAPI.AppVersion{
		IntegrationID:      version.GetIntegrationID(),
		IntegrationVersion: version.GetVersion(),
	}

	a.exporter, err = exporterAPI.NewMittiExporter(a.cm.Configuration, &ver)
	if err != nil {
		runtime.LogError(ctx, err.Error())
		panic("failed to load configuration")
	}
	a.ctx = ctx
}

func checkForConfigFile(basePath, configFile string) bool {
	if _, err := os.Stat(path.Join(basePath, configFile)); os.IsNotExist(err) {
		return false
	}
	return true
}

// resolveConfigFileName picks the configuration filename to use within basePath.
// It prefers the current "mitti-exporter.yaml" name, but falls back to the
// pre-rebrand "safetyculture-exporter.yaml" name if that's the one that
// actually exists, so existing installs keep working without a manual migration.
func resolveConfigFileName(basePath string) string {
	const configFileName = "mitti-exporter.yaml"
	const legacyConfigFileName = "safetyculture-exporter.yaml"

	if checkForConfigFile(basePath, configFileName) {
		return configFileName
	}
	if checkForConfigFile(basePath, legacyConfigFileName) {
		return legacyConfigFileName
	}
	return configFileName
}

// SelectDirectory opens a directory dialog and returns the path of the selected directory
func (a *App) SelectDirectory(currentDir string) string {
	var defaultDir string
	homeDir, err := os.UserHomeDir()
	if len(currentDir) == 0 {
		if err != nil {
			runtime.LogErrorf(a.ctx, "failed to get the working directory, %v", err)
			panic("failed to get working directory")
		}
		defaultDir = homeDir
	}

	defaultDir = currentDir

	directoryDialog, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory:     defaultDir,
		CanCreateDirectories: true,
	})

	if err != nil {
		runtime.LogErrorf(a.ctx, "can't open directory dialog, %v", err)
		return homeDir
	}

	return directoryDialog
}

func (a *App) ValidateExportDirectory() bool {
	exportPath := a.cm.Configuration.Export.Path
	if len(exportPath) == 0 {
		return false
	}

	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		err := os.MkdirAll(exportPath, os.ModePerm)
		if err != nil {
			runtime.LogErrorf(a.ctx, "can't create export directory: %v", err)
			return false
		}
	}

	return true
}

func (a *App) ExportCSV() error {
	return a.exporter.RunCSV()
}

func (a *App) ExportSQL() error {
	return a.exporter.RunSQL()
}

func (a *App) ExportSQLite() error {
	return a.exporter.RunSQLite()
}

func (a *App) ExportReports() error {
	return a.exporter.RunInspectionReports()
}

func (a *App) CheckDBConnection() error {
	return a.exporter.CheckDBConnection()
}

func (a *App) GetTemplates() []exporterAPI.TemplateResponseItem {
	return a.exporter.GetTemplateList()
}

func checkConn(ctx context.Context) (bool, string) {
	// Try to connect to google.com:80
	_, err := net.DialTimeout("tcp", "api.safetyculture.io:80", 1*time.Second)
	if err != nil {
		runtime.LogErrorf(ctx, "connection error: %s", err.Error())
		return false, "connection error"
	}
	return true, ""
}

// ValidateApiKey validates the api, returns true if valid, false otherwise
func (a *App) ValidateApiKey(apiKey string) string {
	result, errMsg := checkConn(a.ctx)
	if result {
		var apiOpts []httpapi.Opt

		cfg := httpapi.ClientCfg{
			Addr:                a.cm.Configuration.API.URL,
			AuthorizationHeader: fmt.Sprintf("Bearer %s", apiKey),
			IntegrationID:       version.GetIntegrationID(),
			IntegrationVersion:  version.GetVersion(),
		}
		c := httpapi.NewClient(&cfg, apiOpts...)
		res, err := httpapi.WhoAmI(a.ctx, c)

		if err != nil {
			runtime.LogErrorf(a.ctx, "cannot check WhoAmI: %s", err.Error())
			return "cannot validate the credentials for the given ApiKey"
		}

		if res == nil || (res != nil && res.UserID == "" || res.OrganisationID == "") {
			runtime.LogErrorf(a.ctx, "cannot validate the credentials for the given ApiKey: %s", apiKey)
			return "cannot validate the credentials for the given ApiKey"
		}

		runtime.LogInfo(a.ctx, "saving the key")

		if apiKey != a.cm.Configuration.AccessToken {
			// save configuration
			a.cm.Configuration.AccessToken = apiKey
			if err := a.cm.SaveConfiguration(); err != nil {
				runtime.LogErrorf(a.ctx, "cannot save configuration: %s", err.Error())
			}

			ver := exporterAPI.AppVersion{
				IntegrationID:      version.GetIntegrationID(),
				IntegrationVersion: version.GetVersion(),
			}

			a.exporter, err = exporterAPI.NewMittiExporter(a.cm.Configuration, &ver)
			if err != nil {
				runtime.LogError(a.ctx, err.Error())
				panic("failed to re-load configuration")
			}
		}
		return ""
	}
	return errMsg
}

// GetSettings gets the configuration from the YAML file
func (a *App) GetSettings() *exporterAPI.ExporterConfiguration {
	return a.cm.Configuration
}

// SaveSettings saves the configuration to the YAML file
func (a *App) SaveSettings(cfg *exporterAPI.ExporterConfiguration) {
	a.cm.Configuration = cfg
	if err := a.cm.SaveConfiguration(); err != nil {
		runtime.LogErrorf(a.ctx, "cannot save configuration: %s", err.Error())
	}
	a.exporter.SetConfiguration(cfg)
	a.exporter.CleanExportStatus()
}

func (a *App) GetUserHomeDirectory() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		runtime.LogErrorf(a.ctx, "failed to find user's home directory, %v", err)
	}
	return dir
}

func (a *App) ReadExportStatus() {
	for {
		//fmt.Println(" ")
		exportStatus := a.exporter.GetExportStatus()

		for _, item := range exportStatus.Feeds {
			//fmt.Printf("> %s\n - %s - %d", "update-"+item.FeedName, item.FeedName, item.Counter)
			runtime.EventsEmit(a.ctx, "update-"+item.FeedName, item)
		}

		if exportStatus.ExportStarted && exportStatus.ExportCompleted {
			runtime.EventsEmit(a.ctx, "finished-export", true)
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

type VersionResponse struct {
	OS           string `json:"os"`
	Current      string `json:"current"`
	Latest       string `json:"latest"`
	DownloadURL  string `json:"download_url"`
	ShouldUpdate bool   `json:"should_update"`
}

func (a *App) ReadVersion() *VersionResponse {
	var current = version.GetVersion()
	var latest string
	var downloadURL string
	var shouldUpdate bool

	releaseInfo := update.Check(current, gitRepoExporter)
	if releaseInfo != nil {
		latest = releaseInfo.Version
		downloadURL = releaseInfo.DownloadURL
		shouldUpdate = version.ShouldUpdate(current, latest) && downloadURL != ""
	}

	return &VersionResponse{
		OS:           osRuntime.GOOS,
		Current:      current,
		Latest:       latest,
		DownloadURL:  downloadURL,
		ShouldUpdate: shouldUpdate,
	}
}

func (a *App) TriggerUpdate(url string) bool {
	runtime.LogInfof(a.ctx, "triggering auto-update from this source: %v", url)
	if err := version.DoUpdate(url); err != nil {
		runtime.LogErrorf(a.ctx, "error during triggering auto-update for %v: %v", url, err.Error())
		return false
	}
	return true
}

func (a *App) ReadBuild() string {
	return osRuntime.GOOS
}

func (a *App) CancelExport() {
	a.exporter.CancelExport()
}

func CreateSettingsDirectory() (string, error) {
	settingDir, err := GetSettingDirectoryPath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(settingDir); os.IsNotExist(err) {
		err := os.MkdirAll(settingDir, 0700)
		if err != nil {
			return "", errors.New("can't create settings directory")
		}
	}

	return settingDir, nil
}

func GetSettingDirectoryPath() (string, error) {
	switch osRuntime.GOOS {
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", errors.New("can't get user's home directory")
		}
		newDir := filepath.Join(homeDir, "/Library/Application Support/mitti-exporter")
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			// Existing installs may still have their settings under the
			// pre-rebrand directory name; keep using it if the new one
			// hasn't been created yet, so settings aren't lost.
			legacyDir := filepath.Join(homeDir, "/Library/Application Support/safetyculture-exporter")
			if _, err := os.Stat(legacyDir); err == nil {
				return legacyDir, nil
			}
		}
		return newDir, nil
	default:
		wd, err := os.Getwd()
		if err != nil {
			return "", errors.New("can't get user's home directory")
		}
		return wd, nil
	}
}

func (a *App) GetSettingDir() string {
	result, err := GetSettingDirectoryPath()
	if err != nil {
		return a.GetUserHomeDirectory()
	}
	return result
}

func (a *App) OpenDirectory(dir string) {
	var cmd *exec.Cmd
	if osRuntime.GOOS == "windows" {
		cmd = exec.Command("explorer", dir)
	} else {
		cmd = exec.Command("open", dir)
	}

	err := cmd.Start()
	if err != nil {
		runtime.LogErrorf(a.ctx, "can't open directory %s, %v", dir, err)
	}
}
