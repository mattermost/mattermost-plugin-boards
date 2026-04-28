// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import "encoding/json"

// Page is a team-scoped collaborative document. Pages live in their own table
// (not under boards) and form a tree via parent_id (self-reference).
//
// Boards and pages cross-reference via board_page_refs and page_board_refs;
// neither owns the other.
//
// See docs/PAGES_PLAN.md (Model Y).
type Page struct {
	// ID is the page's UUID.
	ID string `json:"id"`

	// TeamID is the Mattermost team that owns this page.
	TeamID string `json:"teamId"`

	// ParentID is the parent page's ID. Empty string for top-level pages.
	ParentID string `json:"parentId"`

	// Title is shown in the sidebar tree and used to derive the URL slug.
	Title string `json:"title"`

	// Icon is an optional emoji or icon identifier shown next to the title.
	Icon string `json:"icon"`

	// Cover is an optional cover image URL.
	Cover string `json:"cover"`

	// SortOrder controls sibling order within a parent (smaller first).
	SortOrder int64 `json:"sortOrder"`

	// CreatedBy / ModifiedBy are user IDs.
	CreatedBy  string `json:"createdBy"`
	ModifiedBy string `json:"modifiedBy"`

	CreateAt int64 `json:"createAt"`
	UpdateAt int64 `json:"updateAt"`
	DeleteAt int64 `json:"deleteAt"`
}

// PagePatch describes a partial update to a Page.
type PagePatch struct {
	Title     *string `json:"title,omitempty"`
	Icon      *string `json:"icon,omitempty"`
	Cover     *string `json:"cover,omitempty"`
	ParentID  *string `json:"parentId,omitempty"`
	SortOrder *int64  `json:"sortOrder,omitempty"`
}

// PageContent holds the rich-text body of a Page.
//
// Phase 1: only TiptapJSON is used (single-editor mode).
// Phase 2: YjsState (compacted snapshot) and incremental updates in
// page_yjs_updates become the source of truth; TiptapJSON is rendered from
// the Yjs state on snapshot.
//
// TiptapJSON uses json.RawMessage so it serializes as raw JSON to clients.
// YjsState stays []byte — it's binary, base64 wire encoding is correct.
type PageContent struct {
	PageID          string          `json:"pageId"`
	TiptapJSON      json.RawMessage `json:"tiptapJson,omitempty"`
	YjsState        []byte          `json:"yjsState,omitempty"`
	YjsUpdatesCount int             `json:"yjsUpdatesCount"`
	LastSnapshotAt  int64           `json:"lastSnapshotAt"`
	UpdateAt        int64           `json:"updateAt"`
	UpdateBy        string          `json:"updateBy"`
}

// PageMember represents an explicit ACL entry for a page (Phase 2).
//
// Absence of a row means the page inherits from its nearest ancestor with a
// non-empty page_acl, falling back to the team-level permission baseline.
type PageMember struct {
	PageID          string `json:"pageId"`
	UserID          string `json:"userId"`
	SchemeAdmin     bool   `json:"schemeAdmin"`
	SchemeEditor    bool   `json:"schemeEditor"`
	SchemeCommenter bool   `json:"schemeCommenter"`
	SchemeViewer    bool   `json:"schemeViewer"`
}

// PageChannelLink ties a page to one or more Mattermost channels (many-to-many).
// Used to render "Pages in this channel" in the channel header.
type PageChannelLink struct {
	PageID    string `json:"pageId"`
	ChannelID string `json:"channelId"`
	PinnedBy  string `json:"pinnedBy"`
	PinnedAt  int64  `json:"pinnedAt"`
}

// BoardPageRef — a board references a page (board → page link).
type BoardPageRef struct {
	BoardID   string `json:"boardId"`
	PageID    string `json:"pageId"`
	SortOrder int64  `json:"sortOrder"`
	Label     string `json:"label"`
	AddedBy   string `json:"addedBy"`
	AddedAt   int64  `json:"addedAt"`
}

// PageBoardRef — a page references a board (page → board link).
type PageBoardRef struct {
	PageID  string `json:"pageId"`
	BoardID string `json:"boardId"`
	Label   string `json:"label"`
	AddedBy string `json:"addedBy"`
	AddedAt int64  `json:"addedAt"`
}

// QueryPageOptions filters page lookups.
type QueryPageOptions struct {
	TeamID         string // required
	ParentID       string // optional: list children of this parent ('' = top-level)
	IncludeDeleted bool
	Limit          uint64
}
