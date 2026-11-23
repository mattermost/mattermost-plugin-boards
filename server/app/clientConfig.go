// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

func (a *App) GetClientConfig() *model.ClientConfig {
	return &model.ClientConfig{
		Telemetry:                a.config.Telemetry,
		TelemetryID:              a.config.TelemetryID,
		EnablePublicSharedBoards: a.config.EnablePublicSharedBoards,
		LegacyMode:               a.config.LegacyMode,
		MovedBoardsReadOnly:      a.config.MovedBoardsReadOnly,
		TeammateNameDisplay:      a.config.TeammateNameDisplay,
		FeatureFlags:             a.config.FeatureFlags,
		MaxFileSize:              a.config.MaxFileSize,
	}
}
