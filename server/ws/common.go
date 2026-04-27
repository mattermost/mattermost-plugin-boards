// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ws

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

// UpdateCategoryMessage is sent on block updates.
type UpdateCategoryMessage struct {
	Action          string                              `json:"action"`
	TeamID          string                              `json:"teamId"`
	Category        *model.Category                     `json:"category,omitempty"`
	BoardCategories []*model.BoardCategoryWebsocketData `json:"blockCategories,omitempty"`
}

// UpdateBlockMsg is sent on block updates.
type UpdateBlockMsg struct {
	Action string       `json:"action"`
	TeamID string       `json:"teamId"`
	Block  *model.Block `json:"block"`
}

// UpdateBoardMsg is sent on block updates.
type UpdateBoardMsg struct {
	Action string       `json:"action"`
	TeamID string       `json:"teamId"`
	Board  *model.Board `json:"board"`
}

// UpdateMemberMsg is sent on membership updates.
type UpdateMemberMsg struct {
	Action string             `json:"action"`
	TeamID string             `json:"teamId"`
	Member *model.BoardMember `json:"member"`
}

// UpdateSubscription is sent on subscription updates.
type UpdateSubscription struct {
	Action       string              `json:"action"`
	Subscription *model.Subscription `json:"subscription"`
}

// UpdateClientConfig is sent on block updates.
type UpdateClientConfig struct {
	Action       string             `json:"action"`
	ClientConfig model.ClientConfig `json:"clientconfig"`
}

// UpdateClientConfig is sent on block updates.
type UpdateCardLimitTimestamp struct {
	Action    string `json:"action"`
	Timestamp int64  `json:"timestamp"`
}

// WebsocketCommand is an incoming command from the client.
type WebsocketCommand struct {
	Action    string   `json:"action"`
	TeamID    string   `json:"teamId"`
	Token     string   `json:"token"`
	ReadToken string   `json:"readToken"`
	BlockIDs  []string `json:"blockIds"`

	// Pages — Yjs realtime relay (Phase B).
	PageID       string `json:"pageId"`
	UpdateB64    string `json:"updateB64"`
	AwarenessB64 string `json:"awarenessB64"`
	ClientID     string `json:"clientId"`
}

// UpdatePageYjsMsg relays a Y.Doc update for a page to other team members.
// originClientId lets the sender's editor skip its own echo (Y.Doc dedupes by
// state vector, but skipping echoes early avoids redundant work).
type UpdatePageYjsMsg struct {
	Action         string `json:"action"`
	TeamID         string `json:"teamId"`
	PageID         string `json:"pageId"`
	UpdateB64      string `json:"updateB64"`
	OriginClientID string `json:"originClientId"`
	OriginUserID   string `json:"originUserId"`
}

// UpdatePageYjsAwarenessMsg relays an Awareness update (cursor position,
// user metadata) for a page to other team members.
type UpdatePageYjsAwarenessMsg struct {
	Action         string `json:"action"`
	TeamID         string `json:"teamId"`
	PageID         string `json:"pageId"`
	AwarenessB64   string `json:"awarenessB64"`
	OriginClientID string `json:"originClientId"`
	OriginUserID   string `json:"originUserId"`
}

type CategoryReorderMessage struct {
	Action        string   `json:"action"`
	CategoryOrder []string `json:"categoryOrder"`
	TeamID        string   `json:"teamId"`
}

type CategoryBoardReorderMessage struct {
	Action     string   `json:"action"`
	CategoryID string   `json:"CategoryId"`
	BoardOrder []string `json:"BoardOrder"`
	TeamID     string   `json:"teamId"`
}
