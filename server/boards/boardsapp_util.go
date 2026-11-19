// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package boards

import (
	"math"
	"path"
	"strings"

	"github.com/mattermost/mattermost-plugin-boards/server/services/config"

	mm_model "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
)

const defaultS3Timeout = 60 * 1000 // 60 seconds

// Mattermost v12 removed DataRetentionSettings.EnableBoardsDeletion and
// DataRetentionSettings.BoardsRetentionDays (MM-70018), so on every supported
// server Boards deletion is off and the retention period is the old default.
const defaultBoardsRetentionDays = 365

func createBoardsConfig(mmconfig mm_model.Config, baseURL string, serverID string) *config.Configuration {
	filesBackendSettings := createFilesBackendSettings(mmconfig)

	enableTelemetry := false
	if mmconfig.LogSettings.EnableDiagnostics != nil {
		enableTelemetry = *mmconfig.LogSettings.EnableDiagnostics
	}

	enablePublicSharedBoards := false
	if mmconfig.PluginSettings.Plugins[PluginName][SharedBoardsName] == true {
		enablePublicSharedBoards = true
	}

	featureFlags := parseFeatureFlags(mmconfig.FeatureFlags.ToMap())

	showEmailAddress := false
	if mmconfig.PrivacySettings.ShowEmailAddress != nil {
		showEmailAddress = *mmconfig.PrivacySettings.ShowEmailAddress
	}

	showFullName := false
	if mmconfig.PrivacySettings.ShowFullName != nil {
		showFullName = *mmconfig.PrivacySettings.ShowFullName
	}

	serverRoot := baseURL + "/plugins/focalboard"

	return &config.Configuration{
		ServerRoot:               serverRoot,
		Port:                     -1,
		DBType:                   *mmconfig.SqlSettings.DriverName,
		DBConfigString:           *mmconfig.SqlSettings.DataSource,
		DBTablePrefix:            "focalboard_",
		UseSSL:                   false,
		SecureCookie:             true,
		WebPath:                  path.Join(*mmconfig.PluginSettings.Directory, "focalboard", "pack"),
		FilesDriver:              filesBackendSettings.DriverName,
		FilesPath:                filesBackendSettings.Directory,
		FilesBackendSettings:     filesBackendSettings,
		MaxFileSize:              *mmconfig.FileSettings.MaxFileSize,
		Telemetry:                enableTelemetry,
		TelemetryID:              serverID,
		WebhookUpdate:            []string{},
		SessionExpireTime:        2592000,
		SessionRefreshTime:       18000,
		LocalOnly:                false,
		EnableLocalMode:          false,
		LocalModeSocketLocation:  "",
		AuthMode:                 "mattermost",
		EnablePublicSharedBoards: enablePublicSharedBoards,
		FeatureFlags:             featureFlags,
		NotifyFreqCardSeconds:    getPluginSettingInt(mmconfig, notifyFreqCardSecondsKey, 120),
		NotifyFreqBoardSeconds:   getPluginSettingInt(mmconfig, notifyFreqBoardSecondsKey, 86400),
		EnableDataRetention:      false,
		DataRetentionDays:        defaultBoardsRetentionDays,
		TeammateNameDisplay:      *mmconfig.TeamSettings.TeammateNameDisplay,
		ShowEmailAddress:         showEmailAddress,
		ShowFullName:             showFullName,
	}
}

func parseFeatureFlags(configFeatureFlags map[string]string) map[string]string {
	featureFlags := make(map[string]string)
	for key, value := range configFeatureFlags {
		// Break out FeatureFlags and pass remaining
		if key == boardsFeatureFlagName {
			for _, flag := range strings.Split(value, "-") {
				featureFlags[flag] = "true"
			}
		} else {
			featureFlags[key] = value
		}
	}
	return featureFlags
}

func getPluginSetting(mmConfig mm_model.Config, key string) (interface{}, bool) {
	plugin, ok := mmConfig.PluginSettings.Plugins[PluginName]
	if !ok {
		return nil, false
	}

	val, ok := plugin[key]
	if !ok {
		return nil, false
	}
	return val, true
}

func getPluginSettingInt(mmConfig mm_model.Config, key string, def int) int {
	val, ok := getPluginSetting(mmConfig, key)
	if !ok {
		return def
	}
	valFloat, ok := val.(float64)
	if !ok {
		return def
	}
	return int(math.Round(valFloat))
}

// createFilesBackendSettings builds the file backend settings from
// Mattermost's FileSettings using the same helper the Mattermost server uses,
// so every driver Mattermost supports (local, amazons3, azureblob, ...) is
// configured the same way here.
func createFilesBackendSettings(mmconfig mm_model.Config) filestore.FileBackendSettings {
	// Work on a copy so filling in defaults for unset fields never touches the
	// caller's config. SetDefaults only replaces nil pointers.
	fileSettings := mmconfig.FileSettings
	fileSettings.SetDefaults(false)

	skipVerify := mmconfig.ServiceSettings.EnableInsecureOutgoingConnections != nil &&
		*mmconfig.ServiceSettings.EnableInsecureOutgoingConnections

	allowedUntrustedInternalConnections := ""
	if mmconfig.ServiceSettings.AllowedUntrustedInternalConnections != nil {
		allowedUntrustedInternalConnections = *mmconfig.ServiceSettings.AllowedUntrustedInternalConnections
	}

	// Boards has always honored AmazonS3SSE as configured, so keep doing that
	// rather than gating it on the compliance license.
	settings := filestore.NewFileBackendSettingsFromConfig(&fileSettings, true, skipVerify, allowedUntrustedInternalConnections)

	if settings.DriverName == mm_model.ImageDriverS3 && settings.AmazonS3RequestTimeoutMilliseconds <= 0 {
		settings.AmazonS3RequestTimeoutMilliseconds = defaultS3Timeout
	}

	return settings
}
