CREATE TABLE IF NOT EXISTS {{.prefix}}status_transition_rules (
    id VARCHAR(36) PRIMARY KEY,
    board_id VARCHAR(36) NOT NULL,
    from_status VARCHAR(36) NOT NULL,
    to_status VARCHAR(36) NOT NULL,
    allowed BOOLEAN NOT NULL DEFAULT true,
    create_at BIGINT,
    update_at BIGINT,
    UNIQUE(board_id, from_status, to_status)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{- /* createIndexIfNeeded tableName columns */ -}}
{{ createIndexIfNeeded "status_transition_rules" "board_id" }}
{{ createIndexIfNeeded "status_transition_rules" "board_id, from_status" }}

