{{- /* Pages feature — per-user page categories (Slice 1) */ -}}
{{- /* Mirrors {{.prefix}}categories but separate so Boards/Pages don't share */ -}}
{{- /* user-scoped views of the same category. */ -}}

CREATE TABLE IF NOT EXISTS {{.prefix}}page_categories (
    id varchar(36) NOT NULL,
    user_id varchar(36) NOT NULL,
    team_id varchar(36) NOT NULL,
    name varchar(100) NOT NULL,
    sort_order BIGINT DEFAULT 0,
    collapsed BOOLEAN DEFAULT FALSE,
    create_at BIGINT,
    update_at BIGINT,
    delete_at BIGINT DEFAULT 0,
    PRIMARY KEY (id)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_categories" "user_id, team_id" }}
