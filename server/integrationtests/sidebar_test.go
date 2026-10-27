// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestSidebar(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	teamID := mmModel.NewId()

	// we'll create a new board.
	// The board should end up in a default "Boards" category
	board := th.CreateBoard(teamID, "O")

	categoryBoards := th.GetUserCategoryBoards(teamID)
	require.Equal(t, 1, len(categoryBoards))
	require.Equal(t, "Boards", categoryBoards[0].Name)
	require.Equal(t, 1, len(categoryBoards[0].BoardMetadata))
	require.Equal(t, board.ID, categoryBoards[0].BoardMetadata[0].BoardID)

	// create a new category, a new board
	// and move that board into the new category
	board2 := th.CreateBoard(teamID, "O")
	userID := th.GetUser1().ID
	category := th.CreateCategory(model.Category{
		Name:   "Category 2",
		TeamID: teamID,
		UserID: userID,
	})
	th.UpdateCategoryBoard(teamID, category.ID, board2.ID)

	categoryBoards = th.GetUserCategoryBoards(teamID)
	// now there should be two categories - boards and the one
	// we created just now
	require.Equal(t, 2, len(categoryBoards))

	// the newly created category should be the first one array
	// as new categories end up on top in LHS
	require.Equal(t, "Category 2", categoryBoards[0].Name)
	require.Equal(t, 1, len(categoryBoards[0].BoardMetadata))
	require.Equal(t, board2.ID, categoryBoards[0].BoardMetadata[0].BoardID)

	// now we'll delete the custom category we created, "Category 2"
	// and all it's boards should get moved to the Boards category
	th.DeleteCategory(teamID, category.ID)
	categoryBoards = th.GetUserCategoryBoards(teamID)
	require.Equal(t, 1, len(categoryBoards))
	require.Equal(t, "Boards", categoryBoards[0].Name)
	require.Equal(t, 2, len(categoryBoards[0].BoardMetadata))
	require.Contains(t, categoryBoards[0].BoardMetadata, model.CategoryBoardMetadata{BoardID: board.ID, Hidden: false})
	require.Contains(t, categoryBoards[0].BoardMetadata, model.CategoryBoardMetadata{BoardID: board2.ID, Hidden: false})
}

func TestHideUnhideBoard(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	teamID := mmModel.NewId()

	// we'll create a new board.
	// The board should end up in a default "Boards" category
	th.CreateBoard(teamID, "O")

	// the created board should not be hidden
	categoryBoards := th.GetUserCategoryBoards(teamID)
	require.Equal(t, 1, len(categoryBoards))
	require.Equal(t, "Boards", categoryBoards[0].Name)
	require.Equal(t, 1, len(categoryBoards[0].BoardMetadata))
	require.False(t, categoryBoards[0].BoardMetadata[0].Hidden)

	// now we'll hide the board
	response := th.Client.HideBoard(teamID, categoryBoards[0].ID, categoryBoards[0].BoardMetadata[0].BoardID)
	th.CheckOK(response)

	// verifying if the board has been marked as hidden
	categoryBoards = th.GetUserCategoryBoards(teamID)
	require.True(t, categoryBoards[0].BoardMetadata[0].Hidden)

	// trying to hide the already hidden board.This should have no effect
	response = th.Client.HideBoard(teamID, categoryBoards[0].ID, categoryBoards[0].BoardMetadata[0].BoardID)
	th.CheckOK(response)
	categoryBoards = th.GetUserCategoryBoards(teamID)
	require.True(t, categoryBoards[0].BoardMetadata[0].Hidden)

	// now we'll unhide the board
	response = th.Client.UnhideBoard(teamID, categoryBoards[0].ID, categoryBoards[0].BoardMetadata[0].BoardID)
	th.CheckOK(response)

	// verifying
	categoryBoards = th.GetUserCategoryBoards(teamID)
	require.False(t, categoryBoards[0].BoardMetadata[0].Hidden)

	// trying to unhide the already visible board.This should have no effect
	response = th.Client.UnhideBoard(teamID, categoryBoards[0].ID, categoryBoards[0].BoardMetadata[0].BoardID)
	th.CheckOK(response)
	categoryBoards = th.GetUserCategoryBoards(teamID)
	require.False(t, categoryBoards[0].BoardMetadata[0].Hidden)
}
