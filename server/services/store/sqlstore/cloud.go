// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"errors"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
)

var ErrInvalidCardLimitValue = errors.New("card limit value is invalid")

// activeCardsQuery applies the necessary filters to the query for it
// to fetch an active cards window if the cardLimit is set, or all the
// active cards if it's 0.
// If includeDeleted is true, the query wiil include cards from deleted boards.
func (s *SQLStore) activeCardsQuery(builder sq.StatementBuilderType, selectStr string, cardLimit int, includeDeleted bool) sq.SelectBuilder {
	query := builder.
		Select(selectStr).
		From(s.tablePrefix + "blocks b").
		Join(s.tablePrefix + "boards bd on b.board_id=bd.id").
		Where(sq.Eq{
			"b.type":         model.TypeCard,
			"bd.is_template": false,
		})
	if !includeDeleted {
		query = query.Where(sq.Eq{"bd.delete_at": 0})
	}

	if cardLimit != 0 {
		var cardLimitUint uint64
		if cardLimit > 0 {
			cardLimitUint = uint64(cardLimit)
		}

		query = query.
			Limit(1).
			Offset(cardLimitUint - 1)
	}

	return query
}

// getCardsCount returns the amount of cards in the server.
func (s *SQLStore) getCardsCount(db sq.BaseRunner) (int64, error) {
	row := s.activeCardsQuery(s.getQueryBuilder(db), "count(b.id)", 0, true).
		QueryRow()

	var usedCards int64
	err := row.Scan(&usedCards)
	if err != nil {
		return 0, err
	}

	return usedCards, nil
}

// getUsedCardsCount returns the amount of active cards in the server.
func (s *SQLStore) getUsedCardsCount(db sq.BaseRunner) (int64, error) {
	row := s.activeCardsQuery(s.getQueryBuilder(db), "count(b.id)", 0, false).
		QueryRow()

	var usedCards int64
	err := row.Scan(&usedCards)
	if err != nil {
		return 0, err
	}

	return usedCards, nil
}

// getCardLimitTimestamp returns the timestamp value from the
// system_settings table or zero if it doesn't exist.
func (s *SQLStore) getCardLimitTimestamp(db sq.BaseRunner) (int64, error) {
	scanner := s.getQueryBuilder(db).
		Select("value").
		From(s.tablePrefix + "system_settings").
		Where(sq.Eq{"id": store.CardLimitTimestampSystemKey}).
		QueryRow()

	var result string
	err := scanner.Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	cardLimitTimestamp, err := strconv.Atoi(result)
	if err != nil {
		return 0, ErrInvalidCardLimitValue
	}

	return int64(cardLimitTimestamp), nil
}

// updateCardLimitTimestamp updates the card limit value in the
// system_settings table with the timestamp of the nth last updated
// card, being nth the value of the cardLimit parameter. If cardLimit
// is zero, the timestamp will be set to zero.
func (s *SQLStore) updateCardLimitTimestamp(db sq.BaseRunner, cardLimit int) (int64, error) {
	query := s.getQueryBuilder(db).
		Insert(s.tablePrefix+"system_settings").
		Columns("id", "value")

	var value interface{} = 0
	if cardLimit != 0 {
		value = s.activeCardsQuery(sq.StatementBuilder, "b.update_at", cardLimit, false).
			OrderBy("b.update_at DESC").
			Prefix("COALESCE((").Suffix("), 0)")
	}
	query = query.Values(store.CardLimitTimestampSystemKey, value)

	if s.dbType == model.MysqlDBType {
		query = query.Suffix("ON DUPLICATE KEY UPDATE value = ?", value)
	} else {
		query = query.Suffix(
			`ON CONFLICT (id)
			 DO UPDATE SET value = EXCLUDED.value`,
		)
	}

	result, err := query.Exec()
	if err != nil {
		return 0, err
	}

	if _, err := result.RowsAffected(); err != nil {
		return 0, err
	}

	return s.getCardLimitTimestamp(db)
}
