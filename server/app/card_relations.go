// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// CreateCardRelation creates a new card relation.
func (a *App) CreateCardRelation(relation *model.CardRelation, boardID string) (*model.CardRelation, error) {
	createdRelation, err := a.store.CreateCardRelation(relation)
	if err != nil {
		return nil, err
	}

	// Get the board for the source card to get team ID
	board, _, err := a.store.GetBoardAndCardByID(relation.SourceCardID)
	if err != nil {
		a.logger.Warn("CreateCardRelation: could not get board for source card",
			mlog.String("cardID", relation.SourceCardID),
			mlog.Err(err))
	} else if board != nil {
		createdRelation.BoardID = board.ID
		a.blockChangeNotifier.Enqueue(func() error {
			a.wsAdapter.BroadcastCardRelationChange(board.TeamID, createdRelation)
			return nil
		})
	}

	return createdRelation, nil
}

// GetCardRelations returns all relations for a card.
func (a *App) GetCardRelations(cardID string) ([]*model.CardRelationWithCard, error) {
	return a.store.GetCardRelations(cardID)
}

// GetCardRelation returns a specific card relation.
func (a *App) GetCardRelation(relationID string) (*model.CardRelation, error) {
	return a.store.GetCardRelation(relationID)
}

// UpdateCardRelation updates an existing card relation
func (a *App) UpdateCardRelation(relation *model.CardRelation) (*model.CardRelation, error) {
	updatedRelation, err := a.store.UpdateCardRelation(relation)
	if err != nil {
		return nil, err
	}

	// Get the board for the source card to get team ID
	board, _, err := a.store.GetBoardAndCardByID(relation.SourceCardID)
	if err != nil {
		a.logger.Warn("UpdateCardRelation: could not get board for source card",
			mlog.String("cardID", relation.SourceCardID),
			mlog.Err(err))
	} else if board != nil {
		updatedRelation.BoardID = board.ID
		a.blockChangeNotifier.Enqueue(func() error {
			a.wsAdapter.BroadcastCardRelationChange(board.TeamID, updatedRelation)
			return nil
		})
	}

	return updatedRelation, nil
}

// DeleteCardRelation deletes a card relation
func (a *App) DeleteCardRelation(relationID string) error {
	// Get relation first to broadcast deletion
	relation, err := a.store.GetCardRelation(relationID)
	if err != nil {
		return err
	}

	if deleteErr := a.store.DeleteCardRelation(relationID); deleteErr != nil {
		return deleteErr
	}

	// Get the board for the source card to get team ID
	board, _, err := a.store.GetBoardAndCardByID(relation.SourceCardID)
	if err != nil {
		a.logger.Warn("DeleteCardRelation: could not get board for source card",
			mlog.String("cardID", relation.SourceCardID),
			mlog.Err(err))
	} else if board != nil {
		a.blockChangeNotifier.Enqueue(func() error {
			a.wsAdapter.BroadcastCardRelationDelete(board.TeamID, relationID, board.ID)
			return nil
		})
	}

	return nil
}

// DeleteCardRelationsByCard deletes all relations involving a card
func (a *App) DeleteCardRelationsByCard(cardID string) error {
	relations, err := a.store.GetCardRelations(cardID)
	if err != nil {
		return err
	}

	for _, rel := range relations {
		if err := a.store.DeleteCardRelation(rel.ID); err != nil {
			a.logger.Warn("DeleteCardRelationsByCard: failed to delete relation",
				mlog.String("relationID", rel.ID),
				mlog.Err(err))
			// Continue deleting other relations
		}
	}

	return nil
}
