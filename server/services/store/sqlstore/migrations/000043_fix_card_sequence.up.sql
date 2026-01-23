-- Fix card_sequence for PostgreSQL - update sequence to continue from last value
{{if .postgres}}
SELECT setval('{{.prefix}}card_sequence_id_seq', (SELECT COALESCE(MAX(id), 0) FROM {{.prefix}}card_sequence));
{{end}}

-- For MySQL and SQLite, the AUTO_INCREMENT is already correct
{{if .mysql}}
SELECT 1;
{{end}}

{{if .sqlite}}
SELECT 1;
{{end}}

