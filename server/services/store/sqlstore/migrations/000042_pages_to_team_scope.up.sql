{{- /* Pages — re-architect to team-scoped (Model Y).                          */ -}}
{{- /* Pages were stored as blocks (type='page') under doc-boards in 000041.   */ -}}
{{- /* This migration moves pages out of `blocks` into a dedicated `pages`     */ -}}
{{- /* table owned by a team, plus two cross-reference tables that allow       */ -}}
{{- /* boards and pages to link to each other bidirectionally.                 */ -}}
{{- /*                                                                         */ -}}
{{- /* page_content / page_yjs_updates / page_channels / page_acl /            */ -}}
{{- /* page_links keep the same shape — they still key by page_id, which       */ -}}
{{- /* now references pages.id instead of blocks.id. Same UUID space.          */ -}}
{{- /*                                                                         */ -}}
{{- /* See docs/PAGES_PLAN.md (Model Y).                                       */ -}}

{{- /* Clean up legacy page rows from the type='page' approach. */ -}}
DELETE FROM {{.prefix}}page_content
WHERE page_id IN (SELECT id FROM {{.prefix}}blocks WHERE type='page');

DELETE FROM {{.prefix}}blocks WHERE type='page';

{{- /* New: team-scoped pages table */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}pages (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    team_id VARCHAR(36) NOT NULL,
    parent_id VARCHAR(36) NOT NULL DEFAULT '',
    title TEXT,
    icon VARCHAR(64),
    cover VARCHAR(512),
    sort_order BIGINT DEFAULT 0,
    created_by VARCHAR(36),
    modified_by VARCHAR(36),
    create_at BIGINT,
    update_at BIGINT,
    delete_at BIGINT DEFAULT 0
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "pages" "team_id, parent_id" }}
{{ createIndexIfNeeded "pages" "team_id, delete_at" }}

{{- /* New: board references page (a board lists/links pages) */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}board_page_refs (
    board_id VARCHAR(36) NOT NULL,
    page_id VARCHAR(36) NOT NULL,
    sort_order BIGINT DEFAULT 0,
    label VARCHAR(255) DEFAULT '',
    added_by VARCHAR(36),
    added_at BIGINT,
    PRIMARY KEY (board_id, page_id)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "board_page_refs" "page_id" }}

{{- /* New: page references board (a page mentions/embeds boards) */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}page_board_refs (
    page_id VARCHAR(36) NOT NULL,
    board_id VARCHAR(36) NOT NULL,
    label VARCHAR(255) DEFAULT '',
    added_by VARCHAR(36),
    added_at BIGINT,
    PRIMARY KEY (page_id, board_id)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_board_refs" "board_id" }}
