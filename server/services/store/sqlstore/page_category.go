// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

// Pages feature — per-user page category store (Slice 1).

func pageCategoryColumns() []string {
	return []string{
		"id", "name", "user_id", "team_id",
		"COALESCE(sort_order, 0)",
		"COALESCE(collapsed, FALSE)",
		"COALESCE(create_at, 0)",
		"COALESCE(update_at, 0)",
		"COALESCE(delete_at, 0)",
	}
}

func scanPageCategory(row interface{ Scan(...interface{}) error }) (*model.PageCategory, error) {
	var c model.PageCategory
	if err := row.Scan(
		&c.ID, &c.Name, &c.UserID, &c.TeamID,
		&c.SortOrder, &c.Collapsed,
		&c.CreateAt, &c.UpdateAt, &c.DeleteAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

// GetPageCategories returns this user's non-deleted page categories for
// the given team, ordered by sort_order then create_at.
func (s *SQLStore) GetPageCategories(userID, teamID string) ([]*model.PageCategory, error) {
	if userID == "" || teamID == "" {
		return nil, errors.New("userID and teamID are required")
	}
	q := s.getQueryBuilder(s.db).
		Select(pageCategoryColumns()...).
		From(s.tablePrefix + "page_categories").
		Where(sq.Eq{"user_id": userID, "team_id": teamID, "delete_at": 0}).
		OrderBy("sort_order ASC", "create_at ASC")

	rows, err := q.Query()
	if err != nil {
		return nil, fmt.Errorf("GetPageCategories: %w", err)
	}
	defer rows.Close()
	var out []*model.PageCategory
	for rows.Next() {
		c, err := scanPageCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetPageCategory fetches a single non-deleted category by id.
func (s *SQLStore) GetPageCategory(id string) (*model.PageCategory, error) {
	q := s.getQueryBuilder(s.db).
		Select(pageCategoryColumns()...).
		From(s.tablePrefix + "page_categories").
		Where(sq.Eq{"id": id, "delete_at": 0})

	c, err := scanPageCategory(q.QueryRow())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.NewErrNotFound("page_category " + id)
		}
		return nil, err
	}
	return c, nil
}

// CreatePageCategory inserts a new page category. ID is generated if empty.
func (s *SQLStore) CreatePageCategory(c *model.PageCategory) (*model.PageCategory, error) {
	if c == nil {
		return nil, errors.New("category is nil")
	}
	if err := c.IsValid(); err != nil {
		return nil, err
	}
	if c.ID == "" {
		c.ID = utils.NewID(utils.IDTypeNone)
	}
	now := utils.GetMillis()
	c.CreateAt = now
	c.UpdateAt = now
	c.DeleteAt = 0

	q := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"page_categories").
		Columns(
			"id", "name", "user_id", "team_id",
			"sort_order", "collapsed",
			"create_at", "update_at", "delete_at",
		).
		Values(
			c.ID, c.Name, c.UserID, c.TeamID,
			c.SortOrder, c.Collapsed,
			c.CreateAt, c.UpdateAt, c.DeleteAt,
		)
	if _, err := q.Exec(); err != nil {
		return nil, fmt.Errorf("insert page_category: %w", err)
	}
	return c, nil
}

// UpdatePageCategory rewrites mutable columns (name, sort_order, collapsed).
// Returns the updated row.
func (s *SQLStore) UpdatePageCategory(c *model.PageCategory) (*model.PageCategory, error) {
	if c == nil || c.ID == "" {
		return nil, errors.New("category id required")
	}
	existing, err := s.GetPageCategory(c.ID)
	if err != nil {
		return nil, err
	}
	if existing.UserID != c.UserID {
		return nil, model.NewErrPermission("category does not belong to user")
	}
	now := utils.GetMillis()

	upd := s.getQueryBuilder(s.db).
		Update(s.tablePrefix + "page_categories").
		Set("name", c.Name).
		Set("sort_order", c.SortOrder).
		Set("collapsed", c.Collapsed).
		Set("update_at", now).
		Where(sq.Eq{"id": c.ID})
	if _, err := upd.Exec(); err != nil {
		return nil, fmt.Errorf("update page_category: %w", err)
	}
	return s.GetPageCategory(c.ID)
}

// ─── Page→Category assignments (Slice 2) ───────────────────────────

// SetPageCategory upserts a page's category assignment for a user. If the
// page was already in another category, the row is updated in place
// (PRIMARY KEY user_id, page_id enforces single assignment).
func (s *SQLStore) SetPageCategory(userID, pageID, categoryID string, sortOrder int64) error {
	if userID == "" || pageID == "" || categoryID == "" {
		return errors.New("userID, pageID, categoryID required")
	}
	now := utils.GetMillis()

	upd := s.getQueryBuilder(s.db).
		Update(s.tablePrefix + "page_category_pages").
		Set("category_id", categoryID).
		Set("sort_order", sortOrder).
		Set("update_at", now).
		Where(sq.Eq{"user_id": userID, "page_id": pageID})
	res, err := upd.Exec()
	if err != nil {
		return fmt.Errorf("update page_category_pages: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		return nil
	}

	ins := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"page_category_pages").
		Columns("user_id", "page_id", "category_id", "sort_order", "create_at", "update_at").
		Values(userID, pageID, categoryID, sortOrder, now, now)
	if _, err := ins.Exec(); err != nil {
		return fmt.Errorf("insert page_category_pages: %w", err)
	}
	return nil
}

// UnsetPageCategory removes the user's assignment for a page (back to
// the default uncategorized section).
func (s *SQLStore) UnsetPageCategory(userID, pageID string) error {
	if userID == "" || pageID == "" {
		return errors.New("userID and pageID required")
	}
	q := s.getQueryBuilder(s.db).
		Delete(s.tablePrefix + "page_category_pages").
		Where(sq.Eq{"user_id": userID, "page_id": pageID})
	_, err := q.Exec()
	return err
}

// GetPageCategoryAssignments returns all of this user's page→category
// assignments. Caller groups by category_id.
func (s *SQLStore) GetPageCategoryAssignments(userID, teamID string) ([]*model.PageCategoryAssignment, error) {
	if userID == "" || teamID == "" {
		return nil, errors.New("userID and teamID required")
	}
	// Join with page_categories so we filter to this team. Pages live in
	// page_categories rows scoped to (user_id, team_id) — assignments
	// pointing at categories outside this team are ignored.
	q := s.getQueryBuilder(s.db).
		Select(
			"pcp.user_id", "pcp.page_id", "pcp.category_id",
			"COALESCE(pcp.sort_order, 0)",
		).
		From(s.tablePrefix + "page_category_pages pcp").
		Join(s.tablePrefix + "page_categories pc ON pc.id = pcp.category_id").
		Where(sq.Eq{"pcp.user_id": userID, "pc.team_id": teamID, "pc.delete_at": 0}).
		OrderBy("pcp.sort_order ASC")

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.PageCategoryAssignment
	for rows.Next() {
		var a model.PageCategoryAssignment
		if err := rows.Scan(&a.UserID, &a.PageID, &a.CategoryID, &a.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// DeletePageCategory soft-deletes a user's category.
func (s *SQLStore) DeletePageCategory(id, userID string) error {
	existing, err := s.GetPageCategory(id)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return model.NewErrPermission("category does not belong to user")
	}
	now := utils.GetMillis()
	upd := s.getQueryBuilder(s.db).
		Update(s.tablePrefix + "page_categories").
		Set("delete_at", now).
		Set("update_at", now).
		Where(sq.Eq{"id": id})
	_, err = upd.Exec()
	return err
}
