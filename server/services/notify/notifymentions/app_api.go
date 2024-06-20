// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
package notifymentions

import "github.com/mattermost/mattermost-plugin-boards/server/model"

type AppAPI interface {
	GetMemberForBoard(boardID, userID string) (*model.BoardMember, error)
	AddMemberToBoard(member *model.BoardMember) (*model.BoardMember, error)
}
