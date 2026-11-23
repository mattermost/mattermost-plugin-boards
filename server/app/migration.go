// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

// GetBoardMigration returns whether a board was moved to the new version of
// Boards and, when moved boards are read-only, whether that blocks changes to
// it. userID decides whether the board's new channel is reported as openable.
// Outside Legacy Mode no board is reported as moved and Boards v2 isn't called.
func (a *App) GetBoardMigration(boardID, userID string) model.BoardMigration {
	if !a.config.LegacyMode || a.migrationLookup == nil {
		return model.BoardMigration{}
	}

	info := a.migrationLookup.Get(boardID, userID)
	info.ReadOnly = info.Moved && a.config.MovedBoardsReadOnly
	return info
}
