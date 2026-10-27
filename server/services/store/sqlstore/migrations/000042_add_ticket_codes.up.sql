{{- /* Add ticket code support: card_prefix and card_count columns to boards and boards_history tables */ -}}
{{ addColumnIfNeeded "boards" "card_prefix" "varchar(10)" "" }}
{{ addColumnIfNeeded "boards" "card_count" "bigint" "" }}
{{ addColumnIfNeeded "boards_history" "card_prefix" "varchar(10)" "" }}
{{ addColumnIfNeeded "boards_history" "card_count" "bigint" "" }}

{{- /* Card prefixes are unique per team. Boards without one store NULL, which the unique constraint ignores. */ -}}
UPDATE {{.prefix}}boards SET card_prefix = NULL WHERE card_prefix = '';

{{if or .mysql .postgres}}
{{ addConstraintIfNeeded "boards" "unique_team_card_prefix" "UNIQUE" "UNIQUE(team_id, card_prefix)"}}
{{end}}

{{if .sqlite}}
CREATE UNIQUE INDEX IF NOT EXISTS {{.prefix}}unique_team_card_prefix ON {{.prefix}}boards (team_id, card_prefix);
{{end}}
