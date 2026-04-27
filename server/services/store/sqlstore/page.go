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

// Pages feature — store layer (Model Y).
//
// Pages live in the `pages` table (team-scoped). Tree via parent_id.
// Boards ↔ Pages cross-references in board_page_refs and page_board_refs.
// Page content (Tiptap JSON, Yjs state) in page_content.
//
// See docs/PAGES_PLAN.md.

var errPageNotImplemented = errors.New("page store: not implemented")

func pageColumns(prefix string) string {
	return prefix + "id, " +
		prefix + "team_id, " +
		"COALESCE(" + prefix + "parent_id, '') AS parent_id, " +
		"COALESCE(" + prefix + "title, '') AS title, " +
		"COALESCE(" + prefix + "icon, '') AS icon, " +
		"COALESCE(" + prefix + "cover, '') AS cover, " +
		"COALESCE(" + prefix + "sort_order, 0) AS sort_order, " +
		"COALESCE(" + prefix + "created_by, ''), " +
		"COALESCE(" + prefix + "modified_by, ''), " +
		"COALESCE(" + prefix + "create_at, 0), " +
		"COALESCE(" + prefix + "update_at, 0), " +
		"COALESCE(" + prefix + "delete_at, 0)"
}

func scanPage(row interface{ Scan(...interface{}) error }) (*model.Page, error) {
	var p model.Page
	err := row.Scan(
		&p.ID, &p.TeamID, &p.ParentID, &p.Title, &p.Icon, &p.Cover, &p.SortOrder,
		&p.CreatedBy, &p.ModifiedBy, &p.CreateAt, &p.UpdateAt, &p.DeleteAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePage inserts a new page + empty page_content row.
// If p.ID is empty, a new ID is generated. If p.TeamID is empty, returns error.
func (s *SQLStore) CreatePage(p *model.Page, content []byte) (*model.Page, error) {
	if p == nil {
		return nil, errors.New("page is nil")
	}
	if p.TeamID == "" {
		return nil, errors.New("page.TeamID is required")
	}
	if p.ID == "" {
		p.ID = utils.NewID(utils.IDTypeBlock)
	}

	now := utils.GetMillis()
	p.CreateAt = now
	p.UpdateAt = now

	q1 := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"pages").
		Columns(
			"id", "team_id", "parent_id",
			"title", "icon", "cover", "sort_order",
			"created_by", "modified_by",
			"create_at", "update_at", "delete_at",
		).
		Values(
			p.ID, p.TeamID, p.ParentID,
			p.Title, p.Icon, p.Cover, p.SortOrder,
			p.CreatedBy, p.ModifiedBy,
			now, now, 0,
		)
	if _, err := q1.Exec(); err != nil {
		return nil, fmt.Errorf("insert page: %w", err)
	}

	// page_content insert with Postgres JSONB string-cast (see Phase 1 fix).
	q2 := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"page_content").
		Columns(
			"page_id", "tiptap_json", "yjs_updates_count",
			"create_at", "update_at", "update_by",
		).
		Values(
			p.ID, string(content), 0,
			now, now, p.CreatedBy,
		)
	if _, err := q2.Exec(); err != nil {
		_, _ = s.getQueryBuilder(s.db).
			Delete(s.tablePrefix + "pages").
			Where(sq.Eq{"id": p.ID}).Exec()
		return nil, fmt.Errorf("insert page_content: %w", err)
	}

	return p, nil
}

// GetPage fetches a single page by id (excludes soft-deleted).
func (s *SQLStore) GetPage(id string) (*model.Page, error) {
	q := s.getQueryBuilder(s.db).
		Select(pageColumns("")).
		From(s.tablePrefix + "pages").
		Where(sq.Eq{"id": id})

	p, err := scanPage(q.QueryRow())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.NewErrNotFound("page " + id)
		}
		return nil, err
	}
	if p.DeleteAt != 0 {
		return nil, model.NewErrNotFound("page " + id + " (deleted)")
	}
	return p, nil
}

// GetPagesForTeam returns all top-level + nested pages of a team (flat list).
// Caller builds the tree from parent_id.
func (s *SQLStore) GetPagesForTeam(teamID string) ([]*model.Page, error) {
	if teamID == "" {
		return nil, errors.New("teamID is required")
	}
	q := s.getQueryBuilder(s.db).
		Select(pageColumns("")).
		From(s.tablePrefix + "pages").
		Where(sq.Eq{"team_id": teamID}).
		Where(sq.Eq{"delete_at": 0}).
		OrderBy("create_at ASC")

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetChildPages lists immediate children of parentID under teamID.
// parentID '' means top-level pages.
func (s *SQLStore) GetChildPages(teamID, parentID string) ([]*model.Page, error) {
	if teamID == "" {
		return nil, errors.New("teamID is required")
	}
	q := s.getQueryBuilder(s.db).
		Select(pageColumns("")).
		From(s.tablePrefix + "pages").
		Where(sq.Eq{"team_id": teamID}).
		Where(sq.Eq{"parent_id": parentID}).
		Where(sq.Eq{"delete_at": 0}).
		OrderBy("sort_order ASC", "create_at ASC")

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPageContent loads tiptap_json + Yjs state for a page.
func (s *SQLStore) GetPageContent(pageID string) (*model.PageContent, error) {
	if pageID == "" {
		return nil, errors.New("pageID is required")
	}
	q := s.getQueryBuilder(s.db).
		Select(
			"page_id", "tiptap_json", "yjs_state", "yjs_updates_count",
			"COALESCE(last_snapshot_at, 0)",
			"COALESCE(update_at, 0)",
			"COALESCE(update_by, '')",
		).
		From(s.tablePrefix + "page_content").
		Where(sq.Eq{"page_id": pageID})

	var c model.PageContent
	var tiptap, yjs []byte
	row := q.QueryRow()
	if err := row.Scan(
		&c.PageID, &tiptap, &yjs, &c.YjsUpdatesCount,
		&c.LastSnapshotAt, &c.UpdateAt, &c.UpdateBy,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.NewErrNotFound("page_content " + pageID)
		}
		return nil, err
	}
	c.TiptapJSON = tiptap
	c.YjsState = yjs
	return &c, nil
}

// UpsertPageContent writes the Tiptap JSON. Yjs state untouched.
func (s *SQLStore) UpsertPageContent(pageID string, tiptapJSON []byte, modifiedBy string) error {
	if pageID == "" {
		return errors.New("pageID is required")
	}
	now := utils.GetMillis()
	tjStr := string(tiptapJSON)

	upd := s.getQueryBuilder(s.db).
		Update(s.tablePrefix + "page_content").
		Set("tiptap_json", tjStr).
		Set("update_at", now).
		Set("update_by", modifiedBy).
		Where(sq.Eq{"page_id": pageID})
	res, err := upd.Exec()
	if err != nil {
		return fmt.Errorf("update page_content: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		return nil
	}

	ins := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"page_content").
		Columns(
			"page_id", "tiptap_json", "yjs_updates_count",
			"create_at", "update_at", "update_by",
		).
		Values(
			pageID, tjStr, 0,
			now, now, modifiedBy,
		)
	if _, err := ins.Exec(); err != nil {
		return fmt.Errorf("insert page_content: %w", err)
	}
	return nil
}

// ─── Cross-references (board ↔ page) ───────────────────────────────

// LinkBoardToPage adds a board → page reference.
func (s *SQLStore) LinkBoardToPage(boardID, pageID, addedBy, label string, sortOrder int64) error {
	now := utils.GetMillis()
	// Upsert: try INSERT, if conflict UPDATE label/sort_order.
	q := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"board_page_refs").
		Columns("board_id", "page_id", "sort_order", "label", "added_by", "added_at").
		Values(boardID, pageID, sortOrder, label, addedBy, now)
	if _, err := q.Exec(); err != nil {
		// Conflict — update existing
		upd := s.getQueryBuilder(s.db).
			Update(s.tablePrefix + "board_page_refs").
			Set("label", label).
			Set("sort_order", sortOrder).
			Where(sq.Eq{"board_id": boardID, "page_id": pageID})
		if _, err2 := upd.Exec(); err2 != nil {
			return fmt.Errorf("link board→page: %w (insert err: %v)", err2, err)
		}
	}
	return nil
}

// UnlinkBoardFromPage removes a board → page reference.
func (s *SQLStore) UnlinkBoardFromPage(boardID, pageID string) error {
	q := s.getQueryBuilder(s.db).
		Delete(s.tablePrefix + "board_page_refs").
		Where(sq.Eq{"board_id": boardID, "page_id": pageID})
	_, err := q.Exec()
	return err
}

// GetBoardPageRefs lists pages referenced by a board.
func (s *SQLStore) GetBoardPageRefs(boardID string) ([]*model.BoardPageRef, error) {
	q := s.getQueryBuilder(s.db).
		Select("board_id", "page_id",
			"COALESCE(sort_order, 0)",
			"COALESCE(label, '')",
			"COALESCE(added_by, '')",
			"COALESCE(added_at, 0)",
		).
		From(s.tablePrefix + "board_page_refs").
		Where(sq.Eq{"board_id": boardID}).
		OrderBy("sort_order ASC")

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.BoardPageRef
	for rows.Next() {
		var r model.BoardPageRef
		if err := rows.Scan(&r.BoardID, &r.PageID, &r.SortOrder, &r.Label, &r.AddedBy, &r.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// LinkPageToBoard adds a page → board reference.
func (s *SQLStore) LinkPageToBoard(pageID, boardID, addedBy, label string) error {
	now := utils.GetMillis()
	q := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"page_board_refs").
		Columns("page_id", "board_id", "label", "added_by", "added_at").
		Values(pageID, boardID, label, addedBy, now)
	if _, err := q.Exec(); err != nil {
		upd := s.getQueryBuilder(s.db).
			Update(s.tablePrefix + "page_board_refs").
			Set("label", label).
			Where(sq.Eq{"page_id": pageID, "board_id": boardID})
		if _, err2 := upd.Exec(); err2 != nil {
			return fmt.Errorf("link page→board: %w (insert err: %v)", err2, err)
		}
	}
	return nil
}

// UnlinkPageFromBoard removes a page → board reference.
func (s *SQLStore) UnlinkPageFromBoard(pageID, boardID string) error {
	q := s.getQueryBuilder(s.db).
		Delete(s.tablePrefix + "page_board_refs").
		Where(sq.Eq{"page_id": pageID, "board_id": boardID})
	_, err := q.Exec()
	return err
}

// GetPageBoardRefs lists boards referenced by a page.
func (s *SQLStore) GetPageBoardRefs(pageID string) ([]*model.PageBoardRef, error) {
	q := s.getQueryBuilder(s.db).
		Select("page_id", "board_id",
			"COALESCE(label, '')",
			"COALESCE(added_by, '')",
			"COALESCE(added_at, 0)",
		).
		From(s.tablePrefix + "page_board_refs").
		Where(sq.Eq{"page_id": pageID})

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.PageBoardRef
	for rows.Next() {
		var r model.PageBoardRef
		if err := rows.Scan(&r.PageID, &r.BoardID, &r.Label, &r.AddedBy, &r.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// ─── Phase 2+ stubs ──────────────────────────────────────────────────

func (s *SQLStore) UpdatePage(id string, patch *model.PagePatch, modifiedBy string) (*model.Page, error) {
	return nil, errPageNotImplemented
}
func (s *SQLStore) MovePage(id, newParentID string, sortOrder int64, modifiedBy string) error {
	return errPageNotImplemented
}
func (s *SQLStore) DeletePage(id string, cascade bool, modifiedBy string) error {
	return errPageNotImplemented
}
func (s *SQLStore) AppendYjsUpdate(pageID string, updateBlob []byte, clientID string) error {
	return errPageNotImplemented
}
func (s *SQLStore) GetYjsUpdatesSince(pageID string, sinceID int64) ([][]byte, int64, error) {
	return nil, 0, errPageNotImplemented
}
func (s *SQLStore) CompactYjsUpdates(pageID string, snapshot []byte) error {
	return errPageNotImplemented
}
func (s *SQLStore) GetPageACL(pageID string) ([]*model.PageMember, error) {
	return nil, errPageNotImplemented
}
func (s *SQLStore) SetPageMember(m *model.PageMember) error { return errPageNotImplemented }
func (s *SQLStore) DeletePageMember(pageID, userID string) error {
	return errPageNotImplemented
}
// GetPageChannelLinks lists channels a page is pinned to.
func (s *SQLStore) GetPageChannelLinks(pageID string) ([]*model.PageChannelLink, error) {
	q := s.getQueryBuilder(s.db).
		Select("page_id", "channel_id",
			"COALESCE(pinned_by, '')",
			"COALESCE(pinned_at, 0)",
		).
		From(s.tablePrefix + "page_channels").
		Where(sq.Eq{"page_id": pageID})

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.PageChannelLink
	for rows.Next() {
		var r model.PageChannelLink
		if err := rows.Scan(&r.PageID, &r.ChannelID, &r.PinnedBy, &r.PinnedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// LinkPageToChannel pins a page to a channel.
func (s *SQLStore) LinkPageToChannel(pageID, channelID, pinnedBy string) error {
	now := utils.GetMillis()
	q := s.getQueryBuilder(s.db).
		Insert(s.tablePrefix+"page_channels").
		Columns("page_id", "channel_id", "pinned_by", "pinned_at").
		Values(pageID, channelID, pinnedBy, now)
	if _, err := q.Exec(); err != nil {
		// On conflict (already pinned), ignore — pin is idempotent.
		return nil
	}
	return nil
}

// UnlinkPageFromChannel removes a (page, channel) pin.
func (s *SQLStore) UnlinkPageFromChannel(pageID, channelID string) error {
	q := s.getQueryBuilder(s.db).
		Delete(s.tablePrefix + "page_channels").
		Where(sq.Eq{"page_id": pageID, "channel_id": channelID})
	_, err := q.Exec()
	return err
}

// GetPagesForChannel lists pages pinned to a channel (joined with page metadata,
// only non-deleted pages, ordered by pin time descending).
func (s *SQLStore) GetPagesForChannel(channelID string) ([]*model.Page, error) {
	q := s.getQueryBuilder(s.db).
		Select(pageColumns("p.")).
		From(s.tablePrefix + "pages p").
		Join(s.tablePrefix + "page_channels c ON c.page_id = p.id").
		Where(sq.Eq{"c.channel_id": channelID}).
		Where(sq.Eq{"p.delete_at": 0}).
		OrderBy("c.pinned_at DESC")

	rows, err := q.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
