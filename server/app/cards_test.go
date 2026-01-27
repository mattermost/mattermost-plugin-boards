// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCard(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	board := &model.Board{
		ID: utils.NewID(utils.IDTypeBoard),
	}
	userID := utils.NewID(utils.IDTypeUser)

	props := makeProps(3)

	card := &model.Card{
		BoardID:      board.ID,
		CreatedBy:    userID,
		ModifiedBy:   userID,
		Title:        "test card",
		ContentOrder: []string{utils.NewID(utils.IDTypeBlock), utils.NewID(utils.IDTypeBlock)},
		Properties:   props,
	}
	block := model.Card2Block(card)

	t.Run("success scenario", func(t *testing.T) {
		th.Store.EXPECT().GetBoard(board.ID).Return(board, nil)
		th.Store.EXPECT().InsertBlock(gomock.AssignableToTypeOf(reflect.TypeOf(block)), userID).Return(nil)
		th.Store.EXPECT().GetMembersForBoard(board.ID).Return([]*model.BoardMember{}, nil)

		newCard, err := th.App.CreateCard(card, board.ID, userID, false)

		require.NoError(t, err)
		require.Equal(t, card.BoardID, newCard.BoardID)
		require.Equal(t, card.Title, newCard.Title)
		require.Equal(t, card.ContentOrder, newCard.ContentOrder)
		require.EqualValues(t, card.Properties, newCard.Properties)
	})

	t.Run("error scenario", func(t *testing.T) {
		th.Store.EXPECT().GetBoard(board.ID).Return(board, nil)
		th.Store.EXPECT().InsertBlock(gomock.AssignableToTypeOf(reflect.TypeOf(block)), userID).Return(blockError{"error"})

		newCard, err := th.App.CreateCard(card, board.ID, userID, false)

		require.Error(t, err, "error")
		require.Nil(t, newCard)
	})
}

func TestGetCards(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	board := &model.Board{
		ID: utils.NewID(utils.IDTypeBoard),
	}

	const cardCount = 25

	// make some cards
	blocks := make([]*model.Block, 0, cardCount)
	for i := 0; i < cardCount; i++ {
		card := &model.Block{
			ID:       utils.NewID(utils.IDTypeBlock),
			ParentID: board.ID,
			Schema:   1,
			Type:     model.TypeCard,
			Title:    fmt.Sprintf("card %d", i),
			BoardID:  board.ID,
		}
		blocks = append(blocks, card)
	}

	t.Run("success scenario", func(t *testing.T) {
		opts := model.QueryBlocksOptions{
			BoardID:   board.ID,
			BlockType: model.TypeCard,
		}

		th.Store.EXPECT().GetBlocks(opts).Return(blocks, nil)

		cards, err := th.App.GetCardsForBoard(board.ID, 0, 0)
		require.NoError(t, err)
		assert.Len(t, cards, cardCount)
	})

	t.Run("error scenario", func(t *testing.T) {
		opts := model.QueryBlocksOptions{
			BoardID:   board.ID,
			BlockType: model.TypeCard,
		}

		th.Store.EXPECT().GetBlocks(opts).Return(nil, blockError{"error"})

		cards, err := th.App.GetCardsForBoard(board.ID, 0, 0)
		require.Error(t, err)
		require.Nil(t, cards)
	})
}

func TestPatchCard(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	board := &model.Board{
		ID: utils.NewID(utils.IDTypeBoard),
	}
	userID := utils.NewID(utils.IDTypeUser)

	props := makeProps(3)

	card := &model.Card{
		BoardID:      board.ID,
		CreatedBy:    userID,
		ModifiedBy:   userID,
		Title:        "test card for patch",
		ContentOrder: []string{utils.NewID(utils.IDTypeBlock), utils.NewID(utils.IDTypeBlock)},
		Properties:   copyProps(props),
	}

	newTitle := "patched"
	newIcon := "😀"
	newContentOrder := reverse(card.ContentOrder)

	cardPatch := &model.CardPatch{
		Title:             &newTitle,
		ContentOrder:      &newContentOrder,
		Icon:              &newIcon,
		UpdatedProperties: modifyProps(props),
	}

	t.Run("success scenario", func(t *testing.T) {
		expectedPatchedCard := cardPatch.Patch(card)
		expectedPatchedBlock := model.Card2Block(expectedPatchedCard)

		var blockPatch *model.BlockPatch
		th.Store.EXPECT().GetBoard(board.ID).Return(board, nil)
		th.Store.EXPECT().PatchBlock(card.ID, gomock.AssignableToTypeOf(reflect.TypeOf(blockPatch)), userID).Return(nil)
		th.Store.EXPECT().GetMembersForBoard(board.ID).Return([]*model.BoardMember{}, nil)
		th.Store.EXPECT().GetBlock(card.ID).Return(expectedPatchedBlock, nil).AnyTimes()

		patchedCard, err := th.App.PatchCard(cardPatch, card.ID, userID, false)

		require.NoError(t, err)
		require.Equal(t, board.ID, patchedCard.BoardID)
		require.Equal(t, newTitle, patchedCard.Title)
		require.Equal(t, newIcon, patchedCard.Icon)
		require.Equal(t, newContentOrder, patchedCard.ContentOrder)
		require.EqualValues(t, expectedPatchedCard.Properties, patchedCard.Properties)
	})

	t.Run("error scenario", func(t *testing.T) {
		var blockPatch *model.BlockPatch
		th.Store.EXPECT().GetBoard(board.ID).Return(board, nil)
		th.Store.EXPECT().PatchBlock(card.ID, gomock.AssignableToTypeOf(reflect.TypeOf(blockPatch)), userID).Return(blockError{"error"})

		patchedCard, err := th.App.PatchCard(cardPatch, card.ID, userID, false)

		require.Error(t, err, "error")
		require.Nil(t, patchedCard)
	})
}

func TestGetCard(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	boardID := utils.NewID(utils.IDTypeBoard)
	userID := utils.NewID(utils.IDTypeUser)
	props := makeProps(5)
	contentOrder := []string{utils.NewID(utils.IDTypeUser), utils.NewID(utils.IDTypeUser)}
	fields := make(map[string]any)
	fields["contentOrder"] = contentOrder
	fields["properties"] = props
	fields["icon"] = "😀"
	fields["isTemplate"] = true

	block := &model.Block{
		ID:         utils.NewID(utils.IDTypeBlock),
		ParentID:   boardID,
		Type:       model.TypeCard,
		Title:      "test card",
		BoardID:    boardID,
		Fields:     fields,
		CreatedBy:  userID,
		ModifiedBy: userID,
	}

	t.Run("success scenario", func(t *testing.T) {
		th.Store.EXPECT().GetBlock(block.ID).Return(block, nil)

		card, err := th.App.GetCardByID(block.ID)

		require.NoError(t, err)
		require.Equal(t, boardID, card.BoardID)
		require.Equal(t, block.Title, card.Title)
		require.Equal(t, "😀", card.Icon)
		require.Equal(t, true, card.IsTemplate)
		require.Equal(t, contentOrder, card.ContentOrder)
		require.EqualValues(t, props, card.Properties)
	})

	t.Run("not found", func(t *testing.T) {
		bogusID := utils.NewID(utils.IDTypeBlock)
		th.Store.EXPECT().GetBlock(bogusID).Return(nil, model.NewErrNotFound(bogusID))

		card, err := th.App.GetCardByID(bogusID)

		require.Error(t, err, "error")
		require.True(t, model.IsErrNotFound(err))
		require.Nil(t, card)
	})

	t.Run("error scenario", func(t *testing.T) {
		th.Store.EXPECT().GetBlock(block.ID).Return(nil, blockError{"error"})

		card, err := th.App.GetCardByID(block.ID)

		require.Error(t, err, "error")
		require.Nil(t, card)
	})
}

// reverse is a helper function to copy and reverse a slice of strings.
func reverse(src []string) []string {
	out := make([]string, 0, len(src))
	for i := len(src) - 1; i >= 0; i-- {
		out = append(out, src[i])
	}
	return out
}

func makeProps(count int) map[string]any {
	props := make(map[string]any)
	for i := 0; i < count; i++ {
		props[utils.NewID(utils.IDTypeBlock)] = utils.NewID(utils.IDTypeBlock)
	}
	return props
}

func copyProps(m map[string]any) map[string]any {
	out := make(map[string]any)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func modifyProps(m map[string]any) map[string]any {
	out := make(map[string]any)
	for k := range m {
		out[k] = utils.NewID(utils.IDTypeBlock)
	}
	return out
}

func TestValidateStatusTransitions(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	boardID := utils.NewID(utils.IDTypeBoard)
	userID := utils.NewID(utils.IDTypeUser)
	statusPropID := utils.NewID(utils.IDTypeBlock)

	// Create status options
	statusOption1 := utils.NewID(utils.IDTypeBlock)
	statusOption2 := utils.NewID(utils.IDTypeBlock)
	statusOption3 := utils.NewID(utils.IDTypeBlock)

	board := &model.Board{
		ID: boardID,
		CardProperties: []map[string]interface{}{
			{
				"id":   statusPropID,
				"name": "Status",
				"type": "select",
				"options": []interface{}{
					map[string]interface{}{
						"id":    statusOption1,
						"value": "To Do",
						"color": "propColorDefault",
					},
					map[string]interface{}{
						"id":    statusOption2,
						"value": "In Progress",
						"color": "propColorBlue",
					},
					map[string]interface{}{
						"id":    statusOption3,
						"value": "Done",
						"color": "propColorGreen",
					},
				},
			},
		},
	}

	card := &model.Card{
		ID:         utils.NewID(utils.IDTypeBlock),
		BoardID:    boardID,
		CreatedBy:  userID,
		ModifiedBy: userID,
		Title:      "test card",
		Properties: map[string]interface{}{
			statusPropID: statusOption1, // Currently "To Do"
		},
	}

	t.Run("allowed transition", func(t *testing.T) {
		// Transition from "To Do" to "In Progress" is allowed
		cardPatch := &model.CardPatch{
			UpdatedProperties: map[string]interface{}{
				statusPropID: statusOption2,
			},
		}

		th.Store.EXPECT().IsStatusTransitionAllowed(boardID, statusOption1, statusOption2).Return(true, nil)

		err := th.App.validateStatusTransitions(board, card, cardPatch)
		require.NoError(t, err)
	})

	t.Run("disallowed transition", func(t *testing.T) {
		// Transition from "To Do" to "Done" is not allowed
		cardPatch := &model.CardPatch{
			UpdatedProperties: map[string]interface{}{
				statusPropID: statusOption3,
			},
		}

		th.Store.EXPECT().IsStatusTransitionAllowed(boardID, statusOption1, statusOption3).Return(false, nil)

		err := th.App.validateStatusTransitions(board, card, cardPatch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "can't move to")
	})

	t.Run("no current value - always allowed", func(t *testing.T) {
		// Setting initial status is always allowed
		cardWithoutStatus := &model.Card{
			ID:         utils.NewID(utils.IDTypeBlock),
			BoardID:    boardID,
			CreatedBy:  userID,
			ModifiedBy: userID,
			Title:      "test card",
			Properties: map[string]interface{}{},
		}

		cardPatch := &model.CardPatch{
			UpdatedProperties: map[string]interface{}{
				statusPropID: statusOption1,
			},
		}

		// Should not call IsStatusTransitionAllowed since there's no current value
		err := th.App.validateStatusTransitions(board, cardWithoutStatus, cardPatch)
		require.NoError(t, err)
	})

	t.Run("same value - no validation needed", func(t *testing.T) {
		// Keeping the same status doesn't require validation
		cardPatch := &model.CardPatch{
			UpdatedProperties: map[string]interface{}{
				statusPropID: statusOption1,
			},
		}

		// Should not call IsStatusTransitionAllowed since value isn't changing
		err := th.App.validateStatusTransitions(board, card, cardPatch)
		require.NoError(t, err)
	})

	t.Run("non-select property - no validation", func(t *testing.T) {
		// Non-select properties should not be validated
		textPropID := utils.NewID(utils.IDTypeBlock)
		boardWithTextProp := &model.Board{
			ID: boardID,
			CardProperties: []map[string]interface{}{
				{
					"id":   textPropID,
					"name": "Description",
					"type": "text",
				},
			},
		}

		cardPatch := &model.CardPatch{
			UpdatedProperties: map[string]interface{}{
				textPropID: "new description",
			},
		}

		err := th.App.validateStatusTransitions(boardWithTextProp, card, cardPatch)
		require.NoError(t, err)
	})
}
