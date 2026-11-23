// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

// GetBoardMigration returns whether a board was moved to Boards v2 and, when
// MovedBoardsReadOnly is on, whether that blocks changes to it. userID decides
// whether the board's new channel is reported as openable.
func (a *App) GetBoardMigration(boardID, userID string) model.BoardMigration {
	if a.migrationLookup == nil {
		return model.BoardMigration{}
	}

	info := a.migrationLookup.Get(boardID, userID)
	info.ReadOnly = info.Moved && a.config.MovedBoardsReadOnly
	return info
}
