// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package boards

import (
	"fmt"
	"reflect"

	"github.com/mattermost/mattermost-plugin-boards/server/model"

	mm_model "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	EnablePublicSharedBoards bool
	LegacyMode               bool
	MovedBoardsReadOnly      bool
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (b *BoardsApp) getConfiguration() *configuration {
	b.configurationLock.RLock()
	defer b.configurationLock.RUnlock()

	if b.configuration == nil {
		return &configuration{}
	}

	return b.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (b *BoardsApp) setConfiguration(configuration *configuration) {
	b.configurationLock.Lock()
	defer b.configurationLock.Unlock()

	if configuration != nil && b.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	b.configuration = configuration
}

// syncBotDisplayName makes the Boards bot display name follow LegacyMode. It uses the plugin API,
// so it must not be called while holding the configurationLock.
func (b *BoardsApp) syncBotDisplayName(legacyMode bool) error {
	model.SetBotLegacyMode(legacyMode)
	botID, err := b.servicesAPI.EnsureBot(model.FocalboardBot())
	if err != nil {
		return fmt.Errorf("failed to ensure bot: %w", err)
	}

	bot, err := b.servicesAPI.GetBot(botID, false)
	if err != nil {
		return fmt.Errorf("failed to get bot: %w", err)
	}

	displayName := model.BotDisplayName(legacyMode)
	if bot.DisplayName == displayName {
		return nil
	}

	if _, err := b.servicesAPI.PatchBot(botID, &mm_model.BotPatch{DisplayName: &displayName}); err != nil {
		return fmt.Errorf("failed to patch bot: %w", err)
	}
	return nil
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (b *BoardsApp) OnConfigurationChange() error {
	// Have we been setup by OnActivate?
	if b.server == nil {
		return nil
	}
	mmconfig := b.servicesAPI.GetConfig()

	enableShareBoards := false
	if mmconfig.PluginSettings.Plugins[PluginName][SharedBoardsName] == true {
		enableShareBoards = true
	}

	legacyMode := false
	if mmconfig.PluginSettings.Plugins[PluginName][LegacyModeName] == true {
		legacyMode = true
	}

	movedBoardsReadOnly := getPluginSettingBool(*mmconfig, MovedBoardsReadOnlyName, MovedBoardsReadOnlyDefault)

	configuration := &configuration{
		EnablePublicSharedBoards: enableShareBoards,
		LegacyMode:               legacyMode,
		MovedBoardsReadOnly:      movedBoardsReadOnly,
	}
	b.setConfiguration(configuration)
	b.server.Config().EnablePublicSharedBoards = enableShareBoards
	b.server.Config().LegacyMode = legacyMode
	b.server.Config().MovedBoardsReadOnly = movedBoardsReadOnly

	// handle Data Retention settings
	enableBoardsDeletion := false
	if mmconfig.DataRetentionSettings.EnableBoardsDeletion != nil {
		enableBoardsDeletion = true
	}
	b.server.Config().EnableDataRetention = enableBoardsDeletion

	// Removed from the server config in v12, so the pointer is nil there.
	boardsRetentionDays := mm_model.DataRetentionSettingsDefaultBoardsRetentionDays
	if mmconfig.DataRetentionSettings.BoardsRetentionDays != nil {
		boardsRetentionDays = *mmconfig.DataRetentionSettings.BoardsRetentionDays
	}
	b.server.Config().DataRetentionDays = boardsRetentionDays

	b.server.Config().TeammateNameDisplay = *mmconfig.TeamSettings.TeammateNameDisplay
	showEmailAddress := false
	if mmconfig.PrivacySettings.ShowEmailAddress != nil {
		showEmailAddress = *mmconfig.PrivacySettings.ShowEmailAddress
	}
	b.server.Config().ShowEmailAddress = showEmailAddress
	showFullName := false
	if mmconfig.PrivacySettings.ShowFullName != nil {
		showFullName = *mmconfig.PrivacySettings.ShowFullName
	}
	b.server.Config().ShowFullName = showFullName
	maxFileSize := int64(0)
	if mmconfig.FileSettings.MaxFileSize != nil {
		maxFileSize = *mmconfig.FileSettings.MaxFileSize
	}
	b.server.Config().MaxFileSize = maxFileSize

	b.server.UpdateAppConfig()
	b.wsPluginAdapter.BroadcastConfigChange(*b.server.App().GetClientConfig())

	if err := b.syncBotDisplayName(legacyMode); err != nil {
		b.logger.Warn("Unable to update the Boards bot display name", mlog.Err(err))
	}
	return nil
}
