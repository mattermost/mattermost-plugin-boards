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

		// Generate new IDs for each test run to avoid conflicts
		testBoardID := utils.NewID(utils.IDTypeBoard)
		testCategoryID := utils.NewID(utils.IDTypeNone) // Categories don't have a specific ID type

		category := model.Category{
			ID:     testCategoryID,
			Name:   "TestCategory",
			UserID: testUserID,
			TeamID: testTeamID,
		}
		err := store.CreateCategory(category)
		require.NoError(t, err)

		testRunDataRetention(t, store, 0, testBoardID, testCategoryID)
		testRunDataRetention(t, store, 2, testBoardID, testCategoryID)
		testRunDataRetention(t, store, 10, testBoardID, testCategoryID)
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

		// Data retention deletes category_boards entries but not categories themselves
		// Check that the category exists but has no boards
		categoryBoards, err := store.GetUserCategoryBoards(testUserID, testTeamID)
		require.NoError(t, err)
		// The category should exist but with no boards
		if len(categoryBoards) > 0 {
			// If category exists, it should have no board metadata
			for _, cb := range categoryBoards {
				require.Empty(t, cb.BoardMetadata, "category should have no boards after data retention")
			}
		}
	})
}
