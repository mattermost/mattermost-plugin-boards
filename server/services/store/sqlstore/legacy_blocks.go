package sqlstore

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

func legacyBoardFields(prefix string) []string {
	// substitute new columns with `"\"\""` (empty string) so as to allow
	// row scan to continue to work with new models.

	fields := []string{
		"id",
		"team_id",
		"COALESCE(channel_id, '')",
		"COALESCE(created_by, '')",
		"modified_by",
		"type",
		"''", // substitute for minimum_role column.
		"title",
		"description",
		"icon",
		"show_description",
		"is_template",
		"template_version",
		"COALESCE(properties, '{}')",
		"COALESCE(card_properties, '[]')",
		"create_at",
		"update_at",
		"delete_at",
	}

	if prefix == "" {
		return fields
	}

	prefixedFields := make([]string, len(fields))
	for i, field := range fields {
		switch {
		case strings.HasPrefix(field, "COALESCE("):
			prefixedFields[i] = strings.Replace(field, "COALESCE(", "COALESCE("+prefix, 1)
		case field == "''":
			prefixedFields[i] = field
		default:
			prefixedFields[i] = prefix + field
		}
	}
	return prefixedFields
}

func (s *SQLStore) getLegacyBoardsByCondition(db sq.BaseRunner, conditions ...interface{}) ([]*model.Board, error) {
	return s.getBoardsFieldsByCondition(db, legacyBoardFields(""), conditions...)
}
