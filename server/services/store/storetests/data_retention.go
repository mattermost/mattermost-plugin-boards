// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
package storetests

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/stretchr/testify/require"
)

func StoreTestDataRetention(t *testing.T, setup func(t *testing.T) (store.Store, func())) {
	t.Run("RunDataRetention", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()

		for _, batchSize := range []int{0, 2, 10} {
			// Generate fresh IDs for each invocation so that each call
			// starts with a clean board and category, independent of any
			// hard-deletes performed by the previous run.
			boardID := utils.NewID(utils.IDTypeBoard)
			categoryID := utils.NewID(utils.IDTypeNone)

			category := model.Category{
				ID:     categoryID,
				Name:   "TestCategory",
				UserID: testUserID,
				TeamID: testTeamID,
			}
			err := store.CreateCategory(category)
			require.NoError(t, err)

			testRunDataRetention(t, store, batchSize, boardID, categoryID)
		}
	})
}

func LoadData(t *testing.T, store store.Store, testBoardID, testCategoryID string) {
	validBoard := model.Board{
		ID:         testBoardID,
		IsTemplate: false,
		ModifiedBy: testUserID,
		TeamID:     testTeamID,
	}
	board, err := store.InsertBoard(&validBoard, testUserID)
	require.NoError(t, err)

	validBlock := &model.Block{
		ID:         utils.NewID(utils.IDTypeBlock),
		BoardID:    board.ID,
		ModifiedBy: testUserID,
	}

	validBlock2 := &model.Block{
		ID:         utils.NewID(utils.IDTypeBlock),
		BoardID:    board.ID,
		ModifiedBy: testUserID,
	}
	validBlock3 := &model.Block{
		ID:         utils.NewID(utils.IDTypeBlock),
		BoardID:    board.ID,
		ModifiedBy: testUserID,
	}

	validBlock4 := &model.Block{
		ID:         utils.NewID(utils.IDTypeBlock),
		BoardID:    board.ID,
		ModifiedBy: testUserID,
	}

	newBlocks := []*model.Block{validBlock, validBlock2, validBlock3, validBlock4}

	err = store.InsertBlocks(newBlocks, testUserID)
	require.NoError(t, err)

	member := &model.BoardMember{
		UserID:      testUserID,
		BoardID:     testBoardID,
		SchemeAdmin: true,
	}
	_, err = store.SaveMember(member)
	require.NoError(t, err)

	sharing := model.Sharing{
		ID:      testBoardID,
		Enabled: true,
		Token:   "testToken",
	}
	err = store.UpsertSharing(sharing)
	require.NoError(t, err)

	err = store.AddUpdateCategoryBoard(testUserID, testCategoryID, []string{testBoardID})
	require.NoError(t, err)
}

func testRunDataRetention(t *testing.T, store store.Store, batchSize int, testBoardID, testCategoryID string) {
	LoadData(t, store, testBoardID, testCategoryID)

	blocks, err := store.GetBlocksForBoard(testBoardID)
	require.NoError(t, err)
	require.Len(t, blocks, 4)
	initialCount := len(blocks)

	t.Run("test no deletions", func(t *testing.T) {
		deletions, err := store.RunDataRetention(utils.GetMillisForTime(time.Now().Add(-time.Hour*1)), int64(batchSize))
		require.NoError(t, err)
		require.Equal(t, int64(0), deletions)
	})

	t.Run("test all deletions", func(t *testing.T) {
		deletions, err := store.RunDataRetention(utils.GetMillisForTime(time.Now().Add(time.Hour*1)), int64(batchSize))
		require.NoError(t, err)
		require.True(t, deletions > int64(initialCount))

		// expect all blocks to be deleted.
		blocks, errBlocks := store.GetBlocksForBoard(testBoardID)
		require.NoError(t, errBlocks)
		require.Equal(t, 0, len(blocks))

		// GetMemberForBoard throws error on now rows found
		member, err := store.GetMemberForBoard(testBoardID, testUserID)
		require.Error(t, err)
		require.True(t, model.IsErrNotFound(err), err)
		require.Nil(t, member)

		// GetSharing throws error on now rows found
		sharing, err := store.GetSharing(testBoardID)
		require.Error(t, err)
		require.True(t, model.IsErrNotFound(err), err)
		require.Nil(t, sharing)

		// Data retention deletes category_boards entries but not categories themselves.
		// Assert the category was preserved and has no boards remaining.
		categoryBoards, err := store.GetUserCategoryBoards(testUserID, testTeamID)
		require.NoError(t, err)
		require.NotEmpty(t, categoryBoards, "category should still exist after data retention")
		for _, cb := range categoryBoards {
			require.Empty(t, cb.BoardMetadata, "category should have no boards after data retention")
		}
	})
}
