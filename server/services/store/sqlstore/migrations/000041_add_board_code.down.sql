{{if .mysql}}
DROP INDEX idx_boards_code ON {{.prefix}}boards;
{{else}}
DROP INDEX IF EXISTS {{.prefix}}idx_boards_code;
{{end}}

ALTER TABLE {{.prefix}}boards DROP COLUMN code;

ALTER TABLE {{.prefix}}boards_history DROP COLUMN code;

