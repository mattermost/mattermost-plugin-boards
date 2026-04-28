{{- /* Pages feature — Phase 1 + Phase 2 schema */ -}}
{{- /* See docs/PAGES_PLAN.md for design rationale */ -}}

{{- /* boards.layout: 'board' (default, existing card boards) or 'doc' (new pages workspace) */ -}}
{{ addColumnIfNeeded "boards" "layout" "varchar(16)" "DEFAULT 'board'" }}
{{ addColumnIfNeeded "boards_history" "layout" "varchar(16)" "DEFAULT 'board'" }}

{{- /* page_content — Tiptap document + Yjs binary state */ -}}
{{- /* tiptap_json: source of truth for non-Yjs (Phase 1) and snapshot reference (Phase 2+) */ -}}
{{- /* yjs_state: compacted Yjs state vector (Phase 2+) */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}page_content (
    page_id VARCHAR(36) NOT NULL PRIMARY KEY,
    {{if .postgres}}tiptap_json JSONB,{{end}}
    {{if .mysql}}tiptap_json JSON,{{end}}
    {{if .sqlite}}tiptap_json TEXT,{{end}}
    {{if .postgres}}yjs_state BYTEA,{{end}}
    {{if .mysql}}yjs_state MEDIUMBLOB,{{end}}
    {{if .sqlite}}yjs_state BLOB,{{end}}
    yjs_updates_count INT DEFAULT 0,
    last_snapshot_at BIGINT,
    create_at BIGINT,
    update_at BIGINT,
    update_by VARCHAR(36)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{- /* page_yjs_updates — incremental Yjs update log between snapshots (Phase 2+) */ -}}
{{- /* compaction job collapses these into page_content.yjs_state and deletes processed rows */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}page_yjs_updates (
    {{if .postgres}}id BIGSERIAL PRIMARY KEY,{{end}}
    {{if .mysql}}id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,{{end}}
    {{if .sqlite}}id INTEGER PRIMARY KEY AUTOINCREMENT,{{end}}
    page_id VARCHAR(36) NOT NULL,
    {{if .postgres}}update_blob BYTEA NOT NULL,{{end}}
    {{if .mysql}}update_blob MEDIUMBLOB NOT NULL,{{end}}
    {{if .sqlite}}update_blob BLOB NOT NULL,{{end}}
    client_id VARCHAR(36),
    create_at BIGINT
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_yjs_updates" "page_id, id" }}

{{- /* page_channels — many-to-many: pin a page to one or more channels */ -}}
{{- /* used by channel header "Pages" button to list pinned pages */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}page_channels (
    page_id VARCHAR(36) NOT NULL,
    channel_id VARCHAR(36) NOT NULL,
    pinned_by VARCHAR(36),
    pinned_at BIGINT,
    PRIMARY KEY (page_id, channel_id)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_channels" "channel_id" }}

{{- /* page_acl — per-page ACL override (Phase 2) */ -}}
{{- /* mirrors board_members 4-flag scheme; absence of row means inherit from doc-board */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}page_acl (
    page_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    scheme_admin BOOLEAN,
    scheme_editor BOOLEAN,
    scheme_commenter BOOLEAN,
    scheme_viewer BOOLEAN,
    PRIMARY KEY (page_id, user_id)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_acl" "user_id" }}

{{- /* page_links — backlink index (Phase 2) */ -}}
{{- /* maintained by editor on save: parses content for page mentions and writes rows */ -}}
{{- /* anchor: '' for page-level link, otherwise heading slug for section anchor */ -}}
CREATE TABLE IF NOT EXISTS {{.prefix}}page_links (
    src_page_id VARCHAR(36) NOT NULL,
    dst_page_id VARCHAR(36) NOT NULL,
    anchor VARCHAR(255) DEFAULT '',
    PRIMARY KEY (src_page_id, dst_page_id, anchor)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_links" "dst_page_id" }}
