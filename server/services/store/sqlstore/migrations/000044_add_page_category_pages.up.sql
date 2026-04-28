{{- /* Pages — per-user page→category assignment (Slice 2) */ -}}
{{- /* A user can place a page under at most one category. Absence of a */ -}}
{{- /* row means the page sits in the default "Pages" section. */ -}}

CREATE TABLE IF NOT EXISTS {{.prefix}}page_category_pages (
    user_id varchar(36) NOT NULL,
    page_id varchar(36) NOT NULL,
    category_id varchar(36) NOT NULL,
    sort_order BIGINT DEFAULT 0,
    create_at BIGINT,
    update_at BIGINT,
    PRIMARY KEY (user_id, page_id)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

{{ createIndexIfNeeded "page_category_pages" "category_id" }}
