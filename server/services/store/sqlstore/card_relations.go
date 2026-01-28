// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (s *SQLStore) cardRelationFields(tableAlias string) []string {
	if tableAlias != "" && tableAlias[len(tableAlias)-1] != '.' {
		tableAlias += "."
	}

	return []string{
		tableAlias + "id",
		tableAlias + "source_card_id",
		tableAlias + "target_card_id",
		tableAlias + "relation_type",
		tableAlias + "created_by",
		tableAlias + "create_at_millis",
	}
}

func (s *SQLStore) cardRelationFromRow(row sq.RowScanner) (*model.CardRelation, error) {
	var relation model.CardRelation

	err := row.Scan(
		&relation.ID,
		&relation.SourceCardID,
		&relation.TargetCardID,
		&relation.RelationType,
		&relation.CreatedBy,
		&relation.CreateAt,
	)
	if err != nil {
		return nil, err
	}

	return &relation, nil
}

func (s *SQLStore) cardRelationsFromRows(rows *sql.Rows) ([]*model.CardRelation, error) {
	relations := []*model.CardRelation{}

	for rows.Next() {
		relation, err := s.cardRelationFromRow(rows)
		if err != nil {
			s.logger.Error("cardRelationsFromRows scan error", mlog.Err(err))
			return nil, err
		}
		relations = append(relations, relation)
	}

	return relations, nil
}

func (s *SQLStore) createCardRelation(db sq.BaseRunner, relation *model.CardRelation) (*model.CardRelation, error) {
	if err := relation.IsValid(); err != nil {
		return nil, err
	}

	relation.Populate()

	query := s.getQueryBuilder(db).
		Insert(s.tablePrefix+"card_relations").
		Columns(
			"id",
			"source_card_id",
			"target_card_id",
			"relation_type",
			"created_by",
			"create_at_millis",
		).
		Values(
			relation.ID,
			relation.SourceCardID,
			relation.TargetCardID,
			relation.RelationType,
			relation.CreatedBy,
			relation.CreateAt,
		)

	if _, err := query.Exec(); err != nil {
		s.logger.Error("createCardRelation error", mlog.Err(err))
		return nil, err
	}

	return relation, nil
}

func (s *SQLStore) getCardRelation(db sq.BaseRunner, relationID string) (*model.CardRelation, error) {
	query := s.getQueryBuilder(db).
		Select(s.cardRelationFields("")...).
		From(s.tablePrefix + "card_relations").
		Where(sq.Eq{"id": relationID})

	row := query.QueryRow()
	relation, err := s.cardRelationFromRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewErrNotFound("card relation ID=" + relationID)
		}
		s.logger.Error("getCardRelation error", mlog.Err(err))
		return nil, err
	}

	return relation, nil
}

func (s *SQLStore) getCardRelations(db sq.BaseRunner, cardID string) ([]*model.CardRelationWithCard, error) {
	// Get all relations where the card is either source or target
	query := s.getQueryBuilder(db).
		Select(s.cardRelationFields("cr.")...).
		From(s.tablePrefix + "card_relations AS cr").
		Where(sq.Or{
			sq.Eq{"cr.source_card_id": cardID},
			sq.Eq{"cr.target_card_id": cardID},
		})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error("getCardRelations error", mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	relations, err := s.cardRelationsFromRows(rows)
	if err != nil {
		return nil, err
	}

	// Fetch related cards
	result := make([]*model.CardRelationWithCard, 0, len(relations))
	for _, relation := range relations {
		// Determine which card to fetch (the other card in the relation)
		relatedCardID := relation.TargetCardID
		if relation.SourceCardID != cardID {
			relatedCardID = relation.SourceCardID
		}

		// Fetch the related card (block)
		block, err := s.getBlock(db, relatedCardID)
		if err != nil {
			s.logger.Warn("getCardRelations: related card not found",
				mlog.String("cardID", relatedCardID),
				mlog.Err(err))
			continue
		}

		// Convert block to card
		card, err := model.Block2Card(block)
		if err != nil {
			s.logger.Warn("getCardRelations: error converting block to card",
				mlog.String("blockID", block.ID),
				mlog.Err(err))
			continue
		}

		result = append(result, &model.CardRelationWithCard{
			CardRelation: *relation,
			Card:         card,
		})
	}

	return result, nil
}

func (s *SQLStore) updateCardRelation(db sq.BaseRunner, relation *model.CardRelation) (*model.CardRelation, error) {
	if err := relation.IsValid(); err != nil {
		return nil, err
	}

	query := s.getQueryBuilder(db).
		Update(s.tablePrefix+"card_relations").
		Set("relation_type", relation.RelationType).
		Where(sq.Eq{"id": relation.ID})

	if _, err := query.Exec(); err != nil {
		s.logger.Error("updateCardRelation error", mlog.Err(err))
		return nil, err
	}

	return relation, nil
}

func (s *SQLStore) deleteCardRelation(db sq.BaseRunner, relationID string) error {
	query := s.getQueryBuilder(db).
		Delete(s.tablePrefix + "card_relations").
		Where(sq.Eq{"id": relationID})

	if _, err := query.Exec(); err != nil {
		s.logger.Error("deleteCardRelation error", mlog.Err(err))
		return err
	}

	return nil
}

