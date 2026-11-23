// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

// BoardMigration describes whether a board was moved to Boards v2.
// swagger:model
type BoardMigration struct {
	// Whether the board was moved to Boards v2
	// required: true
	Moved bool `json:"moved"`

	// The Boards v2 channel the board moved to
	ChannelID string `json:"channel_id,omitempty"`

	// The name of the team that owns the channel
	TeamName string `json:"team_name,omitempty"`

	// The channel name, omitted when the user can't open the channel
	ChannelName string `json:"channel_name,omitempty"`

	// The channel display name, omitted when the user can't open the channel
	ChannelDisplayName string `json:"channel_display_name,omitempty"`

	// The time the board moved, in milliseconds since the Unix epoch
	MovedAt int64 `json:"moved_at,omitempty"`

	// Whether the requesting user can open the new channel
	// required: true
	CanOpen bool `json:"can_open"`

	// Whether changes to the board are blocked because it moved
	// required: true
	ReadOnly bool `json:"read_only"`
}
