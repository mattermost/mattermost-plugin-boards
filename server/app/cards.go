// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"fmt"

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

	// Get board to populate card code
	board, err := a.store.GetBoard(newCard.BoardID)
	if err != nil {
		return nil, fmt.Errorf("cannot get board: %w", err)
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
