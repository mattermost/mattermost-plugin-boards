ALTER TABLE {{.prefix}}blocks ADD COLUMN number BIGINT DEFAULT 0;

ALTER TABLE {{.prefix}}blocks_history ADD COLUMN number BIGINT DEFAULT 0;

{{if .mysql}}
CREATE INDEX idx_blocks_board_number ON {{.prefix}}blocks(board_id, number);
{{else}}
CREATE INDEX IF NOT EXISTS idx_blocks_board_number ON {{.prefix}}blocks(board_id, number);
{{end}}

