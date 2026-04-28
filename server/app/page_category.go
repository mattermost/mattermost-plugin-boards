// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

// Pages — per-user page categories (Slice 1).

func (a *App) GetPageCategories(userID, teamID string) ([]*model.PageCategory, error) {
	if userID == "" || teamID == "" {
		return nil, errors.New("userID and teamID required")
	}
	cats, err := a.store.GetPageCategories(userID, teamID)
	if err != nil {
		return nil, err
	}
	if cats == nil {
		cats = []*model.PageCategory{}
	}
	return cats, nil
}

func (a *App) CreatePageCategory(c *model.PageCategory) (*model.PageCategory, error) {
	if c == nil {
		return nil, errors.New("category is nil")
	}
	if err := c.IsValid(); err != nil {
		return nil, err
	}
	created, err := a.store.CreatePageCategory(c)
	if err != nil {
		return nil, fmt.Errorf("CreatePageCategory: %w", err)
	}
	a.broadcastPageCategoryChange(created)
	return created, nil
}

func (a *App) UpdatePageCategory(c *model.PageCategory) (*model.PageCategory, error) {
	if c == nil || c.ID == "" {
		return nil, errors.New("category id required")
	}
	if err := c.IsValid(); err != nil {
		return nil, err
	}
	updated, err := a.store.UpdatePageCategory(c)
	if err != nil {
		return nil, err
	}
	a.broadcastPageCategoryChange(updated)
	return updated, nil
}

func (a *App) DeletePageCategory(id, userID string) error {
	cat, err := a.store.GetPageCategory(id)
	if err != nil {
		return err
	}
	if cat.UserID != userID {
		return model.NewErrPermission("category does not belong to user")
	}
	if err := a.store.DeletePageCategory(id, userID); err != nil {
		return err
	}
	// signal deletion via a payload with delete_at populated.
	cat.DeleteAt = 1
	a.broadcastPageCategoryChange(cat)
	return nil
}

// SetPageCategory assigns a page to one of the user's categories. If
// the page was previously in a different category, the row is replaced.
func (a *App) SetPageCategory(userID, pageID, categoryID string, sortOrder int64) error {
	if userID == "" || pageID == "" || categoryID == "" {
		return errors.New("userID, pageID, categoryID required")
	}
	cat, err := a.store.GetPageCategory(categoryID)
	if err != nil {
		return err
	}
	if cat.UserID != userID {
		return model.NewErrPermission("category does not belong to user")
	}
	if err := a.store.SetPageCategory(userID, pageID, categoryID, sortOrder); err != nil {
		return err
	}
	a.broadcastPageCategoryAssignment(userID, cat.TeamID, pageID, categoryID)
	return nil
}

// UnsetPageCategory removes a user's category assignment for a page.
// teamID is needed for the WS broadcast scope.
func (a *App) UnsetPageCategory(userID, teamID, pageID string) error {
	if userID == "" || pageID == "" {
		return errors.New("userID and pageID required")
	}
	if err := a.store.UnsetPageCategory(userID, pageID); err != nil {
		return err
	}
	a.broadcastPageCategoryAssignment(userID, teamID, pageID, "")
	return nil
}

// GetPageCategoryAssignments fetches the user's full page→category map.
func (a *App) GetPageCategoryAssignments(userID, teamID string) ([]*model.PageCategoryAssignment, error) {
	if userID == "" || teamID == "" {
		return nil, errors.New("userID and teamID required")
	}
	out, err := a.store.GetPageCategoryAssignments(userID, teamID)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []*model.PageCategoryAssignment{}
	}
	return out, nil
}

// broadcastPageCategoryAssignment notifies the user that one of their
// page→category assignments changed. categoryID == "" signals removal.
func (a *App) broadcastPageCategoryAssignment(userID, teamID, pageID, categoryID string) {
	if a.wsAdapter == nil {
		return
	}
	a.wsAdapter.BroadcastPageCategoryAssignment(userID, teamID, pageID, categoryID)
}

// broadcastPageCategoryChange notifies the owning user only — page
// categories are per-user, so other team members must not receive them.
func (a *App) broadcastPageCategoryChange(c *model.PageCategory) {
	if a.wsAdapter == nil || c == nil {
		return
	}
	a.wsAdapter.BroadcastPageCategoryChange(c)
}
