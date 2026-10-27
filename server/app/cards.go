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

	// InsertBlocksAndNotify assigns the card number
	block := model.Card2Block(card)

	newBlocks, err := a.InsertBlocksAndNotify([]*model.Block{block}, userID, disableNotify)
	if err != nil {
		return nil, fmt.Errorf("cannot create card: %w", err)
	}

	newCard, err := model.Block2Card(newBlocks[0])
	if err != nil {
		return nil, err
	}

	a.populateTicketCode(newCard)

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

	// Fetch board prefix for ticket code computation
	board, _ := a.store.GetBoard(boardID)

	cards := make([]*model.Card, 0, len(blocks))
	for _, blk := range blocks {
		b := blk
		if card, err := model.Block2Card(b); err != nil {
			return nil, fmt.Errorf("Block2Card fail: %w", err)
		} else {
			if board != nil {
				card.TicketCode = model.FormatTicketCode(board.CardPrefix, card.CardNumber)
			}
			cards = append(cards, card)
		}
	}
	return cards, nil
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

	a.populateTicketCode(card)

	return card, nil
}

// populateTicketCode sets card.TicketCode from its board's card prefix, if any.
func (a *App) populateTicketCode(card *model.Card) {
	if card.CardNumber <= 0 {
		return
	}
	board, err := a.store.GetBoard(card.BoardID)
	if err != nil {
		return
	}
	card.TicketCode = model.FormatTicketCode(board.CardPrefix, card.CardNumber)
}

func (a *App) GetCardByTicketCode(ticketCode string, teamID string) (*model.Card, error) {
	prefix, number, err := model.ParseTicketCode(ticketCode)
	if err != nil {
		return nil, err
	}

	block, err := a.store.GetCardByTicketCode(prefix, number, teamID)
	if err != nil {
		return nil, err
	}

	card, err := model.Block2Card(block)
	if err != nil {
		return nil, err
	}

	card.TicketCode = model.FormatTicketCode(prefix, number)
	return card, nil
}
