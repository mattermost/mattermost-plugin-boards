// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"encoding/json"
	"io"

	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

// StatusTransitionRule represents a rule for transitioning between statuses
// swagger:model
type StatusTransitionRule struct {
	// The id for this rule
	// required: true
	ID string `json:"id"`

	// The id of the board this rule belongs to
	// required: true
	BoardID string `json:"boardId"`

	// The status option ID to transition from
	// required: true
	FromStatus string `json:"fromStatus"`

	// The status option ID to transition to
	// required: true
	ToStatus string `json:"toStatus"`

	// Whether this transition is allowed
	// required: true
	Allowed bool `json:"allowed"`

	// The creation time in milliseconds since the current epoch
	// required: true
	CreateAt int64 `json:"createAt"`

	// The last modified time in milliseconds since the current epoch
	// required: true
	UpdateAt int64 `json:"updateAt"`
}

// StatusTransitionRulesFromJSON decodes a json array of status transition rules
func StatusTransitionRulesFromJSON(data io.Reader) []*StatusTransitionRule {
	var rules []*StatusTransitionRule
	_ = json.NewDecoder(data).Decode(&rules)
	return rules
}

// Populate populates a StatusTransitionRule with default values
func (r *StatusTransitionRule) Populate() {
	if r.ID == "" {
		r.ID = utils.NewID(utils.IDTypeNone)
	}
	now := utils.GetMillis()
	if r.CreateAt == 0 {
		r.CreateAt = now
	}
	if r.UpdateAt == 0 {
		r.UpdateAt = now
	}
}

// IsValid validates the status transition rule
func (r *StatusTransitionRule) IsValid() error {
	if r.BoardID == "" {
		return NewErrBadRequest("board ID is required")
	}
	if r.FromStatus == "" {
		return NewErrBadRequest("from status is required")
	}
	if r.ToStatus == "" {
		return NewErrBadRequest("to status is required")
	}
	return nil
}

