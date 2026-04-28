// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

// Pages feature — application service (Model Y).
//
// Pages are team-scoped. Permissions: caller (API layer) checks team
// membership (HasPermissionToTeam, PermissionViewTeam). Page-level ACL
// override is Phase 2 (page_acl).
//
// See docs/PAGES_PLAN.md.

// CreatePage creates a new page under parentID within teamID.
// parentID '' means top-level. content is initial Tiptap JSON (may be nil).
func (a *App) CreatePage(p *model.Page, content []byte, userID string) (*model.Page, error) {
	if p == nil {
		return nil, errors.New("page is nil")
	}
	if p.TeamID == "" {
		return nil, errors.New("TeamID is required")
	}

	// Validate parent (if any) exists and belongs to the same team.
	if p.ParentID != "" {
		parent, err := a.store.GetPage(p.ParentID)
		if err != nil {
			return nil, fmt.Errorf("CreatePage: fetch parent %s: %w", p.ParentID, err)
		}
		if parent.TeamID != p.TeamID {
			return nil, errors.New("CreatePage: parent page belongs to a different team")
		}
	}

	p.CreatedBy = userID
	p.ModifiedBy = userID

	if len(content) == 0 {
		content = []byte(`{"type":"doc","content":[{"type":"paragraph"}]}`)
	}

	created, err := a.store.CreatePage(p, content)
	if err != nil {
		return nil, fmt.Errorf("CreatePage: store: %w", err)
	}

	a.broadcastPageChange(created)
	return created, nil
}

// GetPage returns a page by ID.
func (a *App) GetPage(id string) (*model.Page, error) {
	return a.store.GetPage(id)
}

// GetPagesForTeam returns all pages of a team (flat list).
func (a *App) GetPagesForTeam(teamID string) ([]*model.Page, error) {
	if teamID == "" {
		return nil, errors.New("teamID is required")
	}
	return a.store.GetPagesForTeam(teamID)
}

// GetChildPages lists immediate children of parentID under teamID.
func (a *App) GetChildPages(teamID, parentID string) ([]*model.Page, error) {
	if teamID == "" {
		return nil, errors.New("teamID is required")
	}
	return a.store.GetChildPages(teamID, parentID)
}

// GetPageContent loads body for rendering/editing.
func (a *App) GetPageContent(pageID string) (*model.PageContent, error) {
	return a.store.GetPageContent(pageID)
}

// SavePageContent persists Tiptap JSON. Yjs flow added in Phase 2.
func (a *App) SavePageContent(pageID string, tiptapJSON []byte, userID string) error {
	if pageID == "" {
		return errors.New("pageID is required")
	}
	if !json.Valid(tiptapJSON) {
		return errors.New("tiptapJSON is not valid JSON")
	}
	page, err := a.store.GetPage(pageID)
	if err != nil {
		return fmt.Errorf("SavePageContent: fetch page: %w", err)
	}
	if err := a.store.UpsertPageContent(pageID, tiptapJSON, userID); err != nil {
		return fmt.Errorf("SavePageContent: store: %w", err)
	}
	a.broadcastPageChange(page)
	return nil
}

// SaveYjsSnapshot persists a Y.Doc state blob for a page along with the
// derived Tiptap JSON (so non-collaborative readers and Markdown export keep
// working). Broadcasts a page-change so peers refetch on snapshot boundaries.
func (a *App) SaveYjsSnapshot(pageID string, yjsState []byte, derivedTiptapJSON []byte, userID string) error {
	if pageID == "" {
		return errors.New("pageID is required")
	}
	if len(yjsState) == 0 {
		return errors.New("yjsState is empty")
	}
	if !json.Valid(derivedTiptapJSON) {
		return errors.New("derivedTiptapJSON is not valid JSON")
	}
	page, err := a.store.GetPage(pageID)
	if err != nil {
		return fmt.Errorf("SaveYjsSnapshot: fetch page: %w", err)
	}
	if err := a.store.SaveYjsSnapshot(pageID, yjsState, derivedTiptapJSON, userID); err != nil {
		return fmt.Errorf("SaveYjsSnapshot: store: %w", err)
	}
	a.broadcastPageChange(page)
	return nil
}

// ─── Cross-references (board ↔ page) ───────────────────────────────

// LinkBoardToPage attaches a page reference to a board (board → page).
func (a *App) LinkBoardToPage(boardID, pageID, userID, label string, sortOrder int64) error {
	if boardID == "" || pageID == "" {
		return errors.New("boardID and pageID required")
	}
	return a.store.LinkBoardToPage(boardID, pageID, userID, label, sortOrder)
}

// UnlinkBoardFromPage removes a board → page reference.
func (a *App) UnlinkBoardFromPage(boardID, pageID string) error {
	return a.store.UnlinkBoardFromPage(boardID, pageID)
}

// GetBoardPageRefs lists page refs attached to a board.
func (a *App) GetBoardPageRefs(boardID string) ([]*model.BoardPageRef, error) {
	return a.store.GetBoardPageRefs(boardID)
}

// LinkPageToBoard attaches a board reference to a page (page → board).
func (a *App) LinkPageToBoard(pageID, boardID, userID, label string) error {
	if boardID == "" || pageID == "" {
		return errors.New("boardID and pageID required")
	}
	return a.store.LinkPageToBoard(pageID, boardID, userID, label)
}

// UnlinkPageFromBoard removes a page → board reference.
func (a *App) UnlinkPageFromBoard(pageID, boardID string) error {
	return a.store.UnlinkPageFromBoard(pageID, boardID)
}

// GetPageBoardRefs lists board refs attached to a page.
func (a *App) GetPageBoardRefs(pageID string) ([]*model.PageBoardRef, error) {
	return a.store.GetPageBoardRefs(pageID)
}

// broadcastPageChange notifies clients via the existing block-change channel.
// Pages no longer live in `blocks`, but the WS protocol uses Block payloads;
// we send a synthetic Block with type='page' so existing client handlers can
// route on it.
func (a *App) broadcastPageChange(p *model.Page) {
	if a.wsAdapter == nil {
		return
	}
	a.wsAdapter.BroadcastBlockChange(p.TeamID, &model.Block{
		ID:         p.ID,
		ParentID:   p.ParentID,
		BoardID:    "",
		Type:       model.TypePage,
		Title:      p.Title,
		CreatedBy:  p.CreatedBy,
		ModifiedBy: p.ModifiedBy,
		CreateAt:   p.CreateAt,
		UpdateAt:   p.UpdateAt,
		DeleteAt:   p.DeleteAt,
	})
}

// ─── Phase 2+ stubs ──────────────────────────────────────────────────

func (a *App) UpdatePage(id string, patch *model.PagePatch, userID string) (*model.Page, error) {
	return nil, errPageSkeleton
}

// MovePage reparents a page within the same team.
func (a *App) MovePage(id, newParentID string, sortOrder int64, userID string) error {
	if id == "" {
		return errors.New("pageID is required")
	}
	page, err := a.store.GetPage(id)
	if err != nil {
		return fmt.Errorf("MovePage: fetch page: %w", err)
	}
	if err := a.store.MovePage(id, newParentID, sortOrder, userID); err != nil {
		return fmt.Errorf("MovePage: store: %w", err)
	}
	a.broadcastPageChange(page)
	return nil
}

// DuplicatePage clones a page (and descendants if cascade=true) under
// newParentID. Returns the new page's ID.
func (a *App) DuplicatePage(id, newParentID string, cascade bool, userID string) (string, error) {
	if id == "" {
		return "", errors.New("pageID is required")
	}
	srcPage, err := a.store.GetPage(id)
	if err != nil {
		return "", fmt.Errorf("DuplicatePage: fetch source: %w", err)
	}
	// validate new parent (if any) is in same team
	if newParentID != "" {
		parent, err := a.store.GetPage(newParentID)
		if err != nil {
			return "", fmt.Errorf("DuplicatePage: fetch new parent: %w", err)
		}
		if parent.TeamID != srcPage.TeamID {
			return "", errors.New("DuplicatePage: new parent in a different team")
		}
	}
	newID, err := a.store.DuplicatePage(id, newParentID, cascade, userID)
	if err != nil {
		return "", fmt.Errorf("DuplicatePage: store: %w", err)
	}
	if newPage, e := a.store.GetPage(newID); e == nil {
		a.broadcastPageChange(newPage)
	}
	return newID, nil
}

// DeletePage soft-deletes a page. If cascade is false, children are
// reparented to the deleted page's current parent.
func (a *App) DeletePage(id string, cascade bool, userID string) error {
	if id == "" {
		return errors.New("pageID is required")
	}
	page, err := a.store.GetPage(id)
	if err != nil {
		return fmt.Errorf("DeletePage: fetch page: %w", err)
	}
	if err := a.store.DeletePage(id, cascade, userID); err != nil {
		return fmt.Errorf("DeletePage: store: %w", err)
	}
	a.broadcastPageChange(page)
	return nil
}

// LinkPageToChannel pins a page to a channel (page_channels) and
// broadcasts so any open RHS lists refresh.
func (a *App) LinkPageToChannel(pageID, channelID, userID string) error {
	if pageID == "" || channelID == "" {
		return errors.New("pageID and channelID required")
	}
	if err := a.store.LinkPageToChannel(pageID, channelID, userID); err != nil {
		return err
	}
	if page, e := a.store.GetPage(pageID); e == nil && a.wsAdapter != nil {
		a.wsAdapter.BroadcastPageChannelLink(page.TeamID, channelID, pageID, true)
	}
	return nil
}

// UnlinkPageFromChannel removes a (page, channel) pin and broadcasts.
func (a *App) UnlinkPageFromChannel(pageID, channelID, userID string) error {
	if pageID == "" || channelID == "" {
		return errors.New("pageID and channelID required")
	}
	if err := a.store.UnlinkPageFromChannel(pageID, channelID); err != nil {
		return err
	}
	if page, e := a.store.GetPage(pageID); e == nil && a.wsAdapter != nil {
		a.wsAdapter.BroadcastPageChannelLink(page.TeamID, channelID, pageID, false)
	}
	return nil
}

// GetPagesForChannel lists pages pinned to a channel.
func (a *App) GetPagesForChannel(channelID string) ([]*model.Page, error) {
	if channelID == "" {
		return nil, errors.New("channelID required")
	}
	return a.store.GetPagesForChannel(channelID)
}

// GetPageChannelLinks lists channels a page is pinned to.
func (a *App) GetPageChannelLinks(pageID string) ([]*model.PageChannelLink, error) {
	return a.store.GetPageChannelLinks(pageID)
}

var errPageSkeleton = errors.New("pages: not implemented (skeleton)")
