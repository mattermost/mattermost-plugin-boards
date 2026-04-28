// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import "strings"

// PageCategory groups pages in a user's sidebar. Mirrors `Category` (used
// by Boards) but lives in its own table so Pages and Boards categories
// don't share storage. Per-user, per-team.
type PageCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UserID    string `json:"userID"`
	TeamID    string `json:"teamID"`
	SortOrder int64  `json:"sortOrder"`
	Collapsed bool   `json:"collapsed"`
	CreateAt  int64  `json:"createAt"`
	UpdateAt  int64  `json:"updateAt"`
	DeleteAt  int64  `json:"deleteAt"`
}

// PageCategoryAssignment — a user's page→category assignment row.
// A page is in at most one category per user. Absence = uncategorized.
type PageCategoryAssignment struct {
	UserID     string `json:"userID"`
	PageID     string `json:"pageID"`
	CategoryID string `json:"categoryID"`
	SortOrder  int64  `json:"sortOrder"`
}

func (c *PageCategory) IsValid() error {
	if strings.TrimSpace(c.Name) == "" {
		return NewErrInvalidCategory("page category name cannot be empty")
	}
	if strings.TrimSpace(c.UserID) == "" {
		return NewErrInvalidCategory("page category user ID cannot be empty")
	}
	if strings.TrimSpace(c.TeamID) == "" {
		return NewErrInvalidCategory("page category team ID cannot be empty")
	}
	return nil
}
