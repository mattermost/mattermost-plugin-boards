// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:generate mockgen -destination=mocks/mockstore.go -package mocks . Store
package ws

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

const (
	websocketActionAuth                     = "AUTH"
	websocketActionSubscribeTeam            = "SUBSCRIBE_TEAM"
	websocketActionUnsubscribeTeam          = "UNSUBSCRIBE_TEAM"
	websocketActionSubscribeBlocks          = "SUBSCRIBE_BLOCKS"
	websocketActionUnsubscribeBlocks        = "UNSUBSCRIBE_BLOCKS"
	websocketActionUpdateBoard              = "UPDATE_BOARD"
	websocketActionUpdateMember             = "UPDATE_MEMBER"
	websocketActionDeleteMember             = "DELETE_MEMBER"
	websocketActionUpdateBlock              = "UPDATE_BLOCK"
	websocketActionUpdateConfig             = "UPDATE_CLIENT_CONFIG"
	websocketActionUpdateCategory           = "UPDATE_CATEGORY"
	websocketActionUpdateCategoryBoard      = "UPDATE_BOARD_CATEGORY"
	websocketActionUpdateSubscription       = "UPDATE_SUBSCRIPTION"
	websocketActionUpdateCardLimitTimestamp = "UPDATE_CARD_LIMIT_TIMESTAMP"
	websocketActionReorderCategories        = "REORDER_CATEGORIES"
	websocketActionReorderCategoryBoards    = "REORDER_CATEGORY_BOARDS"
	websocketActionSendYjsUpdate            = "SEND_YJS_UPDATE"            // client → server: relay a Y.Doc update
	websocketActionUpdatePageYjs            = "UPDATE_PAGE_YJS"            // server → clients: a Y.Doc update for a page
	websocketActionSendYjsAwareness         = "SEND_YJS_AWARENESS"         // client → server: relay an Awareness update
	websocketActionUpdatePageYjsAwareness   = "UPDATE_PAGE_YJS_AWARENESS"  // server → clients: an Awareness update for a page
	websocketActionUpdatePageCategory       = "UPDATE_PAGE_CATEGORY"       // server → user: page category created/updated/deleted
	websocketActionUpdatePageCategoryAssign = "UPDATE_PAGE_CATEGORY_ASSIGN" // server → user: a page's category assignment changed
	websocketActionUpdatePageChannelLink    = "UPDATE_PAGE_CHANNEL_LINK"    // server → team: page⇄channel link added or removed
)

type Store interface {
	GetBlock(blockID string) (*model.Block, error)
	GetMembersForBoard(boardID string) ([]*model.BoardMember, error)
}

type Adapter interface {
	BroadcastPageCategoryChange(c *model.PageCategory)
	BroadcastPageCategoryAssignment(userID, teamID, pageID, categoryID string)
	BroadcastPageChannelLink(teamID, channelID, pageID string, linked bool)
	BroadcastBlockChange(teamID string, block *model.Block)
	BroadcastBlockDelete(teamID, blockID, boardID string)
	BroadcastBoardChange(teamID string, board *model.Board)
	BroadcastBoardDelete(teamID, boardID string)
	BroadcastMemberChange(teamID, boardID string, member *model.BoardMember)
	BroadcastMemberDelete(teamID, boardID, userID string)
	BroadcastConfigChange(clientConfig model.ClientConfig)
	BroadcastCategoryChange(category model.Category)
	BroadcastCategoryBoardChange(teamID, userID string, blockCategory []*model.BoardCategoryWebsocketData)
	BroadcastCardLimitTimestampChange(cardLimitTimestamp int64)
	BroadcastSubscriptionChange(teamID string, subscription *model.Subscription)
	BroadcastCategoryReorder(teamID, userID string, categoryOrder []string)
	BroadcastCategoryBoardsReorder(teamID, userID, categoryID string, boardsOrder []string)
}
