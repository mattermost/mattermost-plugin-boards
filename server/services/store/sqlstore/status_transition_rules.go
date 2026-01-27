// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (s *SQLStore) statusTransitionRuleFields() []string {
	return []string{
		"id",
		"board_id",
		"from_status",
		"to_status",
		"allowed",
		"create_at",
		"update_at",
	}
}

func (s *SQLStore) statusTransitionRuleFromRow(row sq.RowScanner) (*model.StatusTransitionRule, error) {
	var rule model.StatusTransitionRule
	err := row.Scan(
		&rule.ID,
		&rule.BoardID,
		&rule.FromStatus,
		&rule.ToStatus,
		&rule.Allowed,
		&rule.CreateAt,
		&rule.UpdateAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *SQLStore) getStatusTransitionRules(db sq.BaseRunner, boardID string) ([]*model.StatusTransitionRule, error) {
	query := s.getQueryBuilder(db).
		Select(s.statusTransitionRuleFields()...).
		From(s.tablePrefix + "status_transition_rules").
		Where(sq.Eq{"board_id": boardID})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error("getStatusTransitionRules ERROR", mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	rules := []*model.StatusTransitionRule{}
	for rows.Next() {
		rule, err := s.statusTransitionRuleFromRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (s *SQLStore) saveStatusTransitionRules(db sq.BaseRunner, rules []*model.StatusTransitionRule) error {
	if len(rules) == 0 {
		return nil
	}

	now := utils.GetMillis()

	for _, rule := range rules {
		rule.Populate()
		rule.UpdateAt = now

		if err := rule.IsValid(); err != nil {
			return err
		}

		query := s.getQueryBuilder(db).
			Insert(s.tablePrefix+"status_transition_rules").
			Columns(s.statusTransitionRuleFields()...).
			Values(
				rule.ID,
				rule.BoardID,
				rule.FromStatus,
				rule.ToStatus,
				rule.Allowed,
				rule.CreateAt,
				rule.UpdateAt,
			)

		if s.dbType == model.MysqlDBType {
			query = query.Suffix("ON DUPLICATE KEY UPDATE allowed = ?, update_at = ?", rule.Allowed, rule.UpdateAt)
		} else {
			query = query.Suffix(`
				ON CONFLICT (board_id, from_status, to_status)
				DO UPDATE SET allowed = EXCLUDED.allowed, update_at = EXCLUDED.update_at
			`)
		}

		if _, err := query.Exec(); err != nil {
			s.logger.Error("saveStatusTransitionRules ERROR", mlog.Err(err))
			return err
		}
	}

	return nil
}

func (s *SQLStore) deleteStatusTransitionRulesForBoard(db sq.BaseRunner, boardID string) error {
	query := s.getQueryBuilder(db).
		Delete(s.tablePrefix + "status_transition_rules").
		Where(sq.Eq{"board_id": boardID})

	if _, err := query.Exec(); err != nil {
		s.logger.Error("deleteStatusTransitionRulesForBoard ERROR", mlog.Err(err))
		return err
	}

	return nil
}

func (s *SQLStore) isStatusTransitionAllowed(db sq.BaseRunner, boardID, fromStatus, toStatus string) (bool, error) {
	// If no rules exist for this board, all transitions are allowed
	query := s.getQueryBuilder(db).
		Select("allowed").
		From(s.tablePrefix + "status_transition_rules").
		Where(sq.Eq{
			"board_id":    boardID,
			"from_status": fromStatus,
			"to_status":   toStatus,
		})

	var allowed bool
	err := query.QueryRow().Scan(&allowed)
	if err != nil {
		// If no rule found, transition is allowed by default
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		s.logger.Error("isStatusTransitionAllowed ERROR", mlog.Err(err))
		return false, err
	}

	return allowed, nil
}
