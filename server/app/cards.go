// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

func (a *App) CreateCard(card *model.Card, boardID string, userID string, disableNotify bool) (*model.Card, error) {
	// Convert the card struct to a block and insert the block.
	now := utils.GetMillis()

	card.ID = utils.NewID(utils.IDTypeCard)
	card.BoardID = boardID
	card.CreatedBy = userID
	card.ModifiedBy = userID
	card.CreateAt = now
	card.UpdateAt = now
	card.DeleteAt = 0

	// Get the next card number for this board
	nextNumber, err := a.store.GetNextCardNumber(boardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get next card number: %w", err)
	}
	card.Number = nextNumber

	// Get board to populate card code
	board, err := a.store.GetBoard(boardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get board: %w", err)
	}

	block := model.Card2Block(card)

	newBlocks, err := a.InsertBlocksAndNotify([]*model.Block{block}, userID, disableNotify)
	if err != nil {
		return nil, fmt.Errorf("cannot create card: %w", err)
	}

	newCard, err := model.Block2Card(newBlocks[0])
	if err != nil {
		return nil, err
	}

	a.populateCardCode(newCard, board)

	return newCard, nil
}

func (a *App) GetCardsForBoard(boardID string, page int, perPage int) ([]*model.Card, error) {
	opts := model.QueryBlocksOptions{
		BoardID:   boardID,
		BlockType: model.TypeCard,
		Page:      page,
		PerPage:   perPage,
	}

	blocks, err := a.store.GetBlocks(opts)
	if err != nil {
		return nil, err
	}

	// Get board to populate card codes
	board, err := a.store.GetBoard(boardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get board: %w", err)
	}

	cards := make([]*model.Card, 0, len(blocks))
	for _, blk := range blocks {
		b := blk
		if card, err := model.Block2Card(b); err != nil {
			return nil, fmt.Errorf("Block2Card fail: %w", err)
		} else {
			a.populateCardCode(card, board)
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func (a *App) populateCardCode(card *model.Card, board *model.Board) {
	if card.Number > 0 && board.Code != "" {
		card.Code = fmt.Sprintf("%s-%d", board.Code, card.Number)
	}
}

func (a *App) PatchCard(cardPatch *model.CardPatch, cardID string, userID string, disableNotify bool) (*model.Card, error) {
	// Get the current card to check for status transitions
	currentBlock, err := a.store.GetBlock(cardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get current card: %w", err)
	}

	currentCard, err := model.Block2Card(currentBlock)
	if err != nil {
		return nil, err
	}

	// Get board to check status transition rules
	board, err := a.store.GetBoard(currentCard.BoardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get board: %w", err)
	}

	// Validate status transitions if properties are being updated
	if len(cardPatch.UpdatedProperties) > 0 {
		if validationErr := a.validateStatusTransitions(board, currentCard, cardPatch); validationErr != nil {
			return nil, validationErr
		}
	}

	blockPatch, err := model.CardPatch2BlockPatch(cardPatch)
	if err != nil {
		return nil, err
	}

	newBlock, err := a.PatchBlockAndNotify(cardID, blockPatch, userID, disableNotify)
	if err != nil {
		return nil, fmt.Errorf("cannot patch card %s: %w", cardID, err)
	}

	newCard, err := model.Block2Card(newBlock)
	if err != nil {
		return nil, err
	}

	a.populateCardCode(newCard, board)

	return newCard, nil
}

func (a *App) GetCardByID(cardID string) (*model.Card, error) {
	cardBlock, err := a.GetBlockByID(cardID)
	if err != nil {
		return nil, err
	}

	card, err := model.Block2Card(cardBlock)
	if err != nil {
		return nil, err
	}

	// Get board to populate card code
	board, err := a.store.GetBoard(card.BoardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get board: %w", err)
	}
	a.populateCardCode(card, board)

	return card, nil
}

func (a *App) GetCardByCode(code string) (*model.Card, *model.Board, string, error) {
	block, board, err := a.store.GetCardByCode(code)
	if err != nil {
		return nil, nil, "", err
	}

	card, err := model.Block2Card(block)
	if err != nil {
		return nil, nil, "", fmt.Errorf("Block2Card fail: %w", err)
	}

	a.populateCardCode(card, board)

	// Get first view for the board
	views, err := a.store.GetBlocksWithType(board.ID, string(model.TypeView))
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot get views: %w", err)
	}

	viewID := ""
	if len(views) > 0 {
		viewID = views[0].ID
	}

	return card, board, viewID, nil
}

// validateStatusTransitions checks if status property changes are allowed based on transition rules.
func (a *App) validateStatusTransitions(board *model.Board, currentCard *model.Card, cardPatch *model.CardPatch) error {
	// Find status properties in the board's card properties
	for _, cardProp := range board.CardProperties {
		propID, ok := cardProp["id"].(string)
		if !ok || propID == "" {
			continue
		}

		propType, _ := cardProp["type"].(string)
		propName, _ := cardProp["name"].(string)

		// Check if this is a select-type property named "Status" (case-insensitive)
		if propType != "select" {
			continue
		}

		// Only validate properties named "Status" (case-insensitive)
		if strings.ToLower(propName) != "status" {
			continue
		}

		// Check if this property is being updated in the patch
		newValue, isBeingUpdated := cardPatch.UpdatedProperties[propID]
		if !isBeingUpdated {
			continue
		}

		// Get the current value of this property
		currentValue, hasCurrentValue := currentCard.Properties[propID]

		// If there's no current value, this is setting an initial value, which is always allowed
		if !hasCurrentValue {
			continue
		}

		// Convert values to strings (they should be option IDs)
		fromStatus, ok1 := currentValue.(string)
		toStatus, ok2 := newValue.(string)

		if !ok1 || !ok2 {
			continue
		}

		// Skip if the value isn't actually changing
		if fromStatus == toStatus {
			continue
		}

		// Check if this transition is allowed
		allowed, err := a.store.IsStatusTransitionAllowed(board.ID, fromStatus, toStatus)
		if err != nil {
			return fmt.Errorf("error checking status transition: %w", err)
		}

		if !allowed {
			// Get the option values for better error message
			fromStatusValue := a.getStatusOptionValue(cardProp, fromStatus)
			toStatusValue := a.getStatusOptionValue(cardProp, toStatus)

			return model.NewErrBadRequest(fmt.Sprintf("The %s card can't move to the %s status.", fromStatusValue, toStatusValue))
		}
	}

	return nil
}

// getStatusOptionValue retrieves the display value for a status option ID.
func (a *App) getStatusOptionValue(cardProp map[string]interface{}, optionID string) string {
	options, ok := cardProp["options"].([]interface{})
	if !ok {
		return optionID
	}

	for _, opt := range options {
		optMap, ok := opt.(map[string]interface{})
		if !ok {
			continue
		}

		if id, ok := optMap["id"].(string); ok && id == optionID {
			if value, ok := optMap["value"].(string); ok {
				return value
			}
		}
	}

	return optionID
}
