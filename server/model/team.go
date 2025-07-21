// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"encoding/json"
	"fmt"
	"io"

	mm_model "github.com/mattermost/mattermost/server/public/model"
)

// Team is information global to a team
// swagger:model
type Team struct {
	// ID of the team
	// required: true
	ID string `json:"id"`

	// Title of the team
	// required: false
	Title string `json:"title"`

	// Token required to register new users
	// required: true
	SignupToken string `json:"signupToken"`

	// Team settings
	// required: false
	Settings map[string]interface{} `json:"settings"`

	// ID of user who last modified this
	// required: true
	ModifiedBy string `json:"modifiedBy"`

	// Updated time in miliseconds since the current epoch
	// required: true
	UpdateAt int64 `json:"updateAt"`
}

func TeamFromJSON(data io.Reader) *Team {
	var team *Team
	_ = json.NewDecoder(data).Decode(&team)
	return team
}

func TeamsFromJSON(data io.Reader) []*Team {
	var teams []*Team
	_ = json.NewDecoder(data).Decode(&teams)
	return teams
}

func ValidateTeamID(teamID string, isTemplate bool) error {
	// Validate inputs to ensure proper file path handling
	// Only allow GlobalTeamID for template operations to prevent path traversal attacks
	if !mm_model.IsValidId(teamID) && (!isTemplate || teamID != GlobalTeamID) {
		return fmt.Errorf("invalid teamID in ValidateTeamID: %s", teamID) //nolint:err113
	}
	return nil
}
