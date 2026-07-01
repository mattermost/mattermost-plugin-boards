// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"

	mm_model "github.com/mattermost/mattermost/server/public/model"
)

func (a *App) HasPermissionToBoard(userID, boardID string, permission *mm_model.Permission) bool {
	return a.permissions.HasPermissionToBoard(userID, boardID, permission)
}

func (a *App) checkBoardCreationPermission(userID, teamID string, boardType model.BoardType) error {
	if userID == model.SystemUserID {
		return nil
	}

	if boardType == model.BoardTypeOpen {
		if !a.permissions.HasPermissionToTeam(userID, teamID, model.PermissionCreatePublicChannel) {
			return model.NewErrPermission("access denied to create public boards")
		}
	} else if !a.permissions.HasPermissionToTeam(userID, teamID, model.PermissionCreatePrivateChannel) {
		return model.NewErrPermission("access denied to create private boards")
	}
	return nil
}
