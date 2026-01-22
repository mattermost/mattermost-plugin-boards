ALTER TABLE {{.prefix}}boards ADD COLUMN code VARCHAR(10);

ALTER TABLE {{.prefix}}boards_history ADD COLUMN code VARCHAR(10);

{{if .mysql}}
CREATE INDEX idx_boards_code ON {{.prefix}}boards(code);
{{else}}
CREATE INDEX IF NOT EXISTS idx_boards_code ON {{.prefix}}boards(code);
{{end}}

