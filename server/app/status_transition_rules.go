// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import "github.com/mattermost/mattermost-plugin-boards/server/model"

// GetStatusTransitionRules returns the status transition rules for a board.
func (a *App) GetStatusTransitionRules(boardID string) ([]*model.StatusTransitionRule, error) {
	return a.store.GetStatusTransitionRules(boardID)
}

// SaveStatusTransitionRules saves the status transition rules.
func (a *App) SaveStatusTransitionRules(rules []*model.StatusTransitionRule) error {
	return a.store.SaveStatusTransitionRules(rules)
}

// DeleteStatusTransitionRulesForBoard deletes all status transition rules for a board.
func (a *App) DeleteStatusTransitionRulesForBoard(boardID string) error {
	return a.store.DeleteStatusTransitionRulesForBoard(boardID)
}

// IsStatusTransitionAllowed checks if a status transition is allowed.
func (a *App) IsStatusTransitionAllowed(boardID, fromStatus, toStatus string) (bool, error) {
	return a.store.IsStatusTransitionAllowed(boardID, fromStatus, toStatus)
}
