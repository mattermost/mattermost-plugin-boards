-- Create card_sequence table for atomic card number generation
{{if .postgres}}
CREATE TABLE IF NOT EXISTS {{.prefix}}card_sequence (
    id SERIAL PRIMARY KEY
);
{{end}}

{{if .mysql}}
CREATE TABLE IF NOT EXISTS {{.prefix}}card_sequence (
    id BIGINT AUTO_INCREMENT PRIMARY KEY
) DEFAULT CHARACTER SET utf8mb4;
{{end}}

{{if .sqlite}}
CREATE TABLE IF NOT EXISTS {{.prefix}}card_sequence (
    id INTEGER PRIMARY KEY AUTOINCREMENT
);
{{end}}

-- Add number column to blocks tables
ALTER TABLE {{.prefix}}blocks ADD COLUMN number BIGINT DEFAULT 0;

ALTER TABLE {{.prefix}}blocks_history ADD COLUMN number BIGINT DEFAULT 0;

-- Populate numbers for existing cards
-- Insert one row per existing card to generate sequential numbers
{{if .postgres}}
INSERT INTO {{.prefix}}card_sequence (id)
SELECT generate_series(1, (SELECT COUNT(*) FROM {{.prefix}}blocks WHERE type = 'card' AND delete_at = 0));

-- Update the sequence to continue from the last inserted value
SELECT setval('{{.prefix}}card_sequence_id_seq', (SELECT MAX(id) FROM {{.prefix}}card_sequence));

WITH numbered_cards AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY create_at, id) as row_num
    FROM {{.prefix}}blocks
    WHERE type = 'card' AND delete_at = 0
)
UPDATE {{.prefix}}blocks b
SET number = nc.row_num
FROM numbered_cards nc
WHERE b.id = nc.id;
{{end}}

{{if .mysql}}
SET @row_number = 0;
UPDATE {{.prefix}}blocks
SET number = (@row_number:=@row_number + 1)
WHERE type = 'card' AND delete_at = 0
ORDER BY create_at, id;

INSERT INTO {{.prefix}}card_sequence (id) VALUES (@row_number);
{{end}}

{{if .sqlite}}
INSERT INTO {{.prefix}}card_sequence (id)
SELECT NULL FROM {{.prefix}}blocks WHERE type = 'card' AND delete_at = 0;

UPDATE {{.prefix}}blocks
SET number = (
    SELECT COUNT(*)
    FROM {{.prefix}}blocks b2
    WHERE b2.type = 'card' AND b2.delete_at = 0
    AND (b2.create_at < {{.prefix}}blocks.create_at OR (b2.create_at = {{.prefix}}blocks.create_at AND b2.id <= {{.prefix}}blocks.id))
)
WHERE type = 'card' AND delete_at = 0;
{{end}}

-- Create index for performance
{{if .mysql}}
CREATE INDEX idx_blocks_board_number ON {{.prefix}}blocks(board_id, number);
{{else}}
CREATE INDEX IF NOT EXISTS idx_blocks_board_number ON {{.prefix}}blocks(board_id, number);
{{end}}

