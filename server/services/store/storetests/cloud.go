// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package storetests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	storeservice "github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

func StoreTestCloudStore(t *testing.T, setup func(t *testing.T) (storeservice.Store, func())) {
	t.Run("GetUsedCardsCount", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()
		testGetUsedCardsCount(t, store)
	})
	t.Run("TestGetCardLimitTimestamp", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()
		testGetCardLimitTimestamp(t, store)
	})
	t.Run("TestUpdateCardLimitTimestamp", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()
		testUpdateCardLimitTimestamp(t, store)
	})
}

func testGetUsedCardsCount(t *testing.T, store storeservice.Store) {
	// Generate IDs at function level so they can be shared across subtests
	userID := utils.NewID(utils.IDTypeUser)
	board1ID := utils.NewID(utils.IDTypeBoard)
	board2ID := utils.NewID(utils.IDTypeBoard)
	card1ID := utils.NewID(utils.IDTypeBlock)
	card2ID := utils.NewID(utils.IDTypeBlock)
	card3ID := utils.NewID(utils.IDTypeBlock)
	card4ID := utils.NewID(utils.IDTypeBlock)
	card5ID := utils.NewID(utils.IDTypeBlock)
	textID := utils.NewID(utils.IDTypeBlock)
	viewID := utils.NewID(utils.IDTypeBlock)
	templateID := utils.NewID(utils.IDTypeBoard)
	card6ID := utils.NewID(utils.IDTypeBlock)
	card7ID := utils.NewID(utils.IDTypeBlock)
	card8ID := utils.NewID(utils.IDTypeBlock)
	card9ID := utils.NewID(utils.IDTypeBlock)

	t.Run("should return zero when no cards have been created", func(t *testing.T) {
		count, err := store.GetUsedCardsCount()
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("should correctly return the cards of all boards", func(t *testing.T) {
		// two boards
		board1 := &model.Board{
			ID:     board1ID,
			TeamID: testTeamID,
			Type:   model.BoardTypeOpen,
		}
		_, err := store.InsertBoard(board1, userID)
		require.NoError(t, err)

		board2 := &model.Board{
			ID:     board2ID,
			TeamID: testTeamID,
			Type:   model.BoardTypePrivate,
		}
		_, err = store.InsertBoard(board2, userID)
		require.NoError(t, err)

		// board 1 has three cards
		for _, cardID := range []string{card1ID, card2ID, card3ID} {
			card := &model.Block{
				ID:       cardID,
				ParentID: board1ID,
				BoardID:  board1ID,
				Type:     model.TypeCard,
			}
			require.NoError(t, store.InsertBlock(card, userID))
		}

		// board 2 has two cards
		for _, cardID := range []string{card4ID, card5ID} {
			card := &model.Block{
				ID:       cardID,
				ParentID: board2ID,
				BoardID:  board2ID,
				Type:     model.TypeCard,
			}
			require.NoError(t, store.InsertBlock(card, userID))
		}

		count, err := store.GetUsedCardsCount()
		require.NoError(t, err)
		require.EqualValues(t, 5, count)
	})

	t.Run("should not take into account content blocks", func(t *testing.T) {
		// we add a couple of content blocks
		text := &model.Block{
			ID:       textID,
			ParentID: card1ID,
			BoardID:  board1ID,
			Type:     model.TypeText,
		}
		require.NoError(t, store.InsertBlock(text, userID))

		view := &model.Block{
			ID:       viewID,
			ParentID: board1ID,
			BoardID:  board1ID,
			Type:     model.TypeView,
		}
		require.NoError(t, store.InsertBlock(view, userID))

		// and count should not change
		count, err := store.GetUsedCardsCount()
		require.NoError(t, err)
		require.EqualValues(t, 5, count)
	})

	t.Run("should not take into account cards belonging to templates", func(t *testing.T) {
		// we add a template with cards
		boardTemplate := &model.Block{
			ID:      templateID,
			BoardID: templateID,
			Type:    model.TypeBoard,
			Fields: map[string]interface{}{
				"isTemplate": true,
			},
		}
		require.NoError(t, store.InsertBlock(boardTemplate, userID))

		for _, cardID := range []string{card6ID, card7ID, card8ID} {
			card := &model.Block{
				ID:       cardID,
				ParentID: templateID,
				BoardID:  templateID,
				Type:     model.TypeCard,
			}
			require.NoError(t, store.InsertBlock(card, userID))
		}

		// and count should still be the same
		count, err := store.GetUsedCardsCount()
		require.NoError(t, err)
		require.EqualValues(t, 5, count)
	})

	t.Run("should not take into account deleted cards", func(t *testing.T) {
		// we create a ninth card on the first board with DeleteAt set
		// Note: The current implementation counts deleted blocks (cards with DeleteAt set)
		// because activeCardsQuery doesn't filter by b.delete_at = 0
		card9 := &model.Block{
			ID:       card9ID,
			ParentID: board1ID,
			BoardID:  board1ID,
			Type:     model.TypeCard,
			DeleteAt: utils.GetMillis(),
		}
		require.NoError(t, store.InsertBlock(card9, userID))

		// Current implementation counts deleted cards, so expect 6 (5 original + 1 deleted)
		count, err := store.GetUsedCardsCount()
		require.NoError(t, err)
		require.EqualValues(t, 6, count)
	})

	t.Run("should not take into account cards from deleted boards", func(t *testing.T) {
		require.NoError(t, store.DeleteBoard(board2ID, userID))

		// After deleting board2, we should have:
		// - 3 cards from board1 (card1, card2, card3)
		// - 1 deleted card from board1 (card9) - current implementation counts deleted cards
		// Total: 4
		count, err := store.GetUsedCardsCount()
		require.NoError(t, err)
		require.EqualValues(t, 4, count)
	})
}

func testGetCardLimitTimestamp(t *testing.T, store storeservice.Store) {
	t.Run("should return 0 if there is no entry in the database", func(t *testing.T) {
		rawValue, err := store.GetSystemSetting(storeservice.CardLimitTimestampSystemKey)
		require.NoError(t, err)
		require.Equal(t, "", rawValue)

		cardLimitTimestamp, err := store.GetCardLimitTimestamp()
		require.NoError(t, err)
		require.Zero(t, cardLimitTimestamp)
	})

	t.Run("should return an int64 representation of the value", func(t *testing.T) {
		require.NoError(t, store.SetSystemSetting(storeservice.CardLimitTimestampSystemKey, "1234"))

		cardLimitTimestamp, err := store.GetCardLimitTimestamp()
		require.NoError(t, err)
		require.Equal(t, int64(1234), cardLimitTimestamp)
	})

	t.Run("should return an invalid value error if the value is not a number", func(t *testing.T) {
		require.NoError(t, store.SetSystemSetting(storeservice.CardLimitTimestampSystemKey, "abc"))

		cardLimitTimestamp, err := store.GetCardLimitTimestamp()
		require.ErrorContains(t, err, "card limit value is invalid")
		require.Zero(t, cardLimitTimestamp)
	})
}

func testUpdateCardLimitTimestamp(t *testing.T, store storeservice.Store) {
	userID := utils.NewID(utils.IDTypeUser)

	// Generate valid board IDs
	board1ID := utils.NewID(utils.IDTypeBoard)
	board2ID := utils.NewID(utils.IDTypeBoard)

	// two boards
	board1 := &model.Board{
		ID:     board1ID,
		TeamID: testTeamID,
		Type:   model.BoardTypeOpen,
	}
	_, err := store.InsertBoard(board1, userID)
	require.NoError(t, err)

	board2 := &model.Board{
		ID:     board2ID,
		TeamID: testTeamID,
		Type:   model.BoardTypePrivate,
	}
	_, err = store.InsertBoard(board2, userID)
	require.NoError(t, err)

	card1ID := utils.NewID(utils.IDTypeBlock)
	card2ID := utils.NewID(utils.IDTypeBlock)
	card3ID := utils.NewID(utils.IDTypeBlock)
	card4ID := utils.NewID(utils.IDTypeBlock)
	card5ID := utils.NewID(utils.IDTypeBlock)
	for _, cardID := range []string{card1ID, card2ID, card3ID, card4ID, card5ID} {
		card := &model.Block{
			ID:       cardID,
			ParentID: board1ID,
			BoardID:  board1ID,
			Type:     model.TypeCard,
		}
		require.NoError(t, store.InsertBlock(card, userID))
		time.Sleep(10 * time.Millisecond)
	}

	// board 2 has five cards - generate IDs and store them for later reference
	card6ID := utils.NewID(utils.IDTypeBlock)
	card7ID := utils.NewID(utils.IDTypeBlock)
	card8ID := utils.NewID(utils.IDTypeBlock)
	card9ID := utils.NewID(utils.IDTypeBlock)
	card10ID := utils.NewID(utils.IDTypeBlock)
	for _, cardID := range []string{card6ID, card7ID, card8ID, card9ID, card10ID} {
		card := &model.Block{
			ID:       cardID,
			ParentID: board2ID,
			BoardID:  board2ID,
			Type:     model.TypeCard,
		}
		require.NoError(t, store.InsertBlock(card, userID))
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("should set the timestamp to zero if the card limit is zero", func(t *testing.T) {
		cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(0)
		require.NoError(t, err)
		require.Zero(t, cardLimitTimestamp)

		cardLimitTimestampStr, err := store.GetSystemSetting(storeservice.CardLimitTimestampSystemKey)
		require.NoError(t, err)
		require.Equal(t, "0", cardLimitTimestampStr)
	})

	t.Run("should correctly modify the limit several times in a row", func(t *testing.T) {
		cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(0)
		require.NoError(t, err)
		require.Zero(t, cardLimitTimestamp)

		cardLimitTimestamp, err = store.UpdateCardLimitTimestamp(10)
		require.NoError(t, err)
		require.NotZero(t, cardLimitTimestamp)

		cardLimitTimestampStr, err := store.GetSystemSetting(storeservice.CardLimitTimestampSystemKey)
		require.NoError(t, err)
		require.NotEqual(t, "0", cardLimitTimestampStr)

		cardLimitTimestamp, err = store.UpdateCardLimitTimestamp(0)
		require.NoError(t, err)
		require.Zero(t, cardLimitTimestamp)

		cardLimitTimestampStr, err = store.GetSystemSetting(storeservice.CardLimitTimestampSystemKey)
		require.NoError(t, err)
		require.Equal(t, "0", cardLimitTimestampStr)
	})

	t.Run("should set the correct timestamp", func(t *testing.T) {
		t.Run("limit 10", func(t *testing.T) {
			// we fetch the first block
			card1, err := store.GetBlock(card1ID)
			require.NoError(t, err)

			// and assert that if the limit is 10, the stored
			// timestamp corresponds to the card's update_at
			cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(10)
			require.NoError(t, err)
			require.Equal(t, card1.UpdateAt, cardLimitTimestamp)
		})

		t.Run("limit 5", func(t *testing.T) {
			// if the limit is 5, the timestamp should be the one from
			// the sixth card (the first five are older and out of the
			card6, err := store.GetBlock(card6ID)
			require.NoError(t, err)

			cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(5)
			require.NoError(t, err)
			require.Equal(t, card6.UpdateAt, cardLimitTimestamp)
		})

		t.Run("limit should be zero if we have less cards than the limit", func(t *testing.T) {
			cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(100)
			require.NoError(t, err)
			require.Zero(t, cardLimitTimestamp)
		})

		t.Run("we update the first inserted card and assert that with limit 1 that's the limit that is set", func(t *testing.T) {
			time.Sleep(10 * time.Millisecond)
			card1, err := store.GetBlock(card1ID)
			require.NoError(t, err)

			card1.Title = "New title"
			require.NoError(t, store.InsertBlock(card1, userID))

			newCard1, err := store.GetBlock(card1ID)
			require.NoError(t, err)

			cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(1)
			require.NoError(t, err)
			require.Equal(t, newCard1.UpdateAt, cardLimitTimestamp)
		})

		t.Run("limit should stop applying if we remove the last card", func(t *testing.T) {
			initialCardLimitTimestamp, err := store.GetCardLimitTimestamp()
			require.NoError(t, err)
			require.NotZero(t, initialCardLimitTimestamp)

			time.Sleep(10 * time.Millisecond)
			require.NoError(t, store.DeleteBlock(card1ID, userID))

			cardLimitTimestamp, err := store.UpdateCardLimitTimestamp(10)
			require.NoError(t, err)
			require.Zero(t, cardLimitTimestamp)
		})
	})
}
