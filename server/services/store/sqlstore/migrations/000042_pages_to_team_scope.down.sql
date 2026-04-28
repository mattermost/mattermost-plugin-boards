{{- /* Reverse 000042 — drop team-scoped pages model. */ -}}
{{- /* Note: page rows in the new `pages` table are NOT migrated back into    */ -}}
{{- /* blocks — they would not match the legacy schema. This is a one-way     */ -}}
{{- /* downgrade for development environments only.                           */ -}}

DROP TABLE IF EXISTS {{.prefix}}page_board_refs;
DROP TABLE IF EXISTS {{.prefix}}board_page_refs;
DROP TABLE IF EXISTS {{.prefix}}pages;
