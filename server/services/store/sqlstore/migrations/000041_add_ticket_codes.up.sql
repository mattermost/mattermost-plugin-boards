{{- /* Add ticket code support: card_prefix and card_count columns to boards and boards_history tables */ -}}
{{ addColumnIfNeeded "boards" "card_prefix" "varchar(10)" "" }}
{{ addColumnIfNeeded "boards" "card_count" "bigint" "" }}
{{ addColumnIfNeeded "boards_history" "card_prefix" "varchar(10)" "" }}
{{ addColumnIfNeeded "boards_history" "card_count" "bigint" "" }}
