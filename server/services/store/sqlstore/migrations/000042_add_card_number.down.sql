-- Drop card_sequence table
DROP TABLE IF EXISTS {{.prefix}}card_sequence;

-- Drop number column from blocks tables
{{if .mysql}}
ALTER TABLE {{.prefix}}blocks DROP COLUMN number;
ALTER TABLE {{.prefix}}blocks_history DROP COLUMN number;
DROP INDEX idx_blocks_board_number ON {{.prefix}}blocks;
{{else}}
ALTER TABLE {{.prefix}}blocks DROP COLUMN number;
ALTER TABLE {{.prefix}}blocks_history DROP COLUMN number;
DROP INDEX IF EXISTS idx_blocks_board_number;
{{end}}

