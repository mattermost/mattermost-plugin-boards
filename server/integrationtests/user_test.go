// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/client"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestGetMe(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	t.Run("not login yet", func(t *testing.T) {
		// Create a client without authentication
		client := client.NewClient(th.Server.Config().ServerRoot, "")
		me, resp := client.GetMe()
		require.Error(t, resp.Error)
		require.Nil(t, me)
	})
}

func TestGetUser(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	th.Client = clients.TeamMember

	me, resp := th.Client.GetMe()
	require.NoError(t, resp.Error)
	require.NotNil(t, me)

	t.Run("me's id", func(t *testing.T) {
		user, resp := th.Client.GetUser(me.ID)
		require.NoError(t, resp.Error)
		require.NotNil(t, user)
		require.Equal(t, me.ID, user.ID)
		require.Equal(t, me.Username, user.Username)
	})

	t.Run("nonexist user", func(t *testing.T) {
		user, resp := th.Client.GetUser("nonexistid")
		require.Error(t, resp.Error)
		require.Nil(t, user)
	})
}

func TestGetUserList(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	th.Client = clients.TeamMember
	th.Client2 = clients.Editor

	me, resp := th.Client.GetMe()
	require.NoError(t, resp.Error)
	require.NotNil(t, me)

	userID1 := me.ID
	userID2 := th.GetUser2().ID

	// Admin user should return both
	returnUsers, resp := clients.Admin.GetUserList([]string{userID1, userID2})
	require.NoError(t, resp.Error)
	require.NotNil(t, returnUsers)
	require.Equal(t, 2, len(returnUsers))

	// Guest user should return none
	returnUsers2, resp := clients.Guest.GetUserList([]string{userID1, userID2})
	require.NoError(t, resp.Error)
	require.NotNil(t, returnUsers2)
	require.Equal(t, 0, len(returnUsers2))

	// Get dynamically generated team IDs from the test store
	testTeamID, _, _ := th.GetTestTeamIDs()
	newBoard := &model.Board{
		Title:  "title",
		Type:   model.BoardTypeOpen,
		TeamID: testTeamID,
	}
	board, err := th.Server.App().CreateBoard(newBoard, userID1, true)
	require.NoError(t, err)

	// add Guest as board member
	newGuestMember := &model.BoardMember{
		UserID:          userGuestID,
		BoardID:         board.ID,
		SchemeViewer:    true,
		SchemeCommenter: true,
		SchemeEditor:    true,
		SchemeAdmin:     false,
	}
	guestMember, err := th.Server.App().AddMemberToBoard(newGuestMember)
	require.NoError(t, err)
	require.NotNil(t, guestMember)

	// Guest user should now return one of members
	guestUsers, resp := clients.Guest.GetUserList([]string{th.GetUser1().ID, th.GetUser2().ID})
	require.NoError(t, resp.Error)
	require.NotNil(t, guestUsers)
	require.Equal(t, 1, len(guestUsers))

	// add other user as board member
	newBoardMember := &model.BoardMember{
		UserID:          userID2,
		BoardID:         board.ID,
		SchemeViewer:    true,
		SchemeCommenter: true,
		SchemeEditor:    true,
		SchemeAdmin:     false,
	}
	newMember, err := th.Server.App().AddMemberToBoard(newBoardMember)
	require.NoError(t, err)
	require.NotNil(t, newMember)

	// Guest user should now return both
	guestUsers, resp = clients.Guest.GetUserList([]string{th.GetUser1().ID, th.GetUser2().ID})
	require.NoError(t, resp.Error)
	require.NotNil(t, guestUsers)
	require.Equal(t, 2, len(guestUsers))
}

func randomBytes(t *testing.T, n int) []byte {
	bb := make([]byte, n)
	_, err := rand.Read(bb)
	require.NoError(t, err)
	return bb
}

func TestTeamUploadFile(t *testing.T) {
	t.Run("no permission", func(t *testing.T) { // native auth, but not login
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		// Use unauthenticated client
		th.Client = client.NewClient(th.Server.Config().ServerRoot, "")
		// Generate a valid Mattermost team ID (26 characters)
		teamID := mmModel.NewId()
		boardID := utils.NewID(utils.IDTypeBoard)
		data := randomBytes(t, 1024)
		result, resp := th.Client.TeamUploadFile(teamID, boardID, bytes.NewReader(data))
		require.Error(t, resp.Error)
		require.Nil(t, result)
	})

	t.Run("a board admin should be able to update a file", func(t *testing.T) { // single token auth
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember
		// Generate a valid Mattermost team ID (26 characters) for file operations
		teamID := mmModel.NewId()
		newBoard := &model.Board{
			Type:   model.BoardTypeOpen,
			TeamID: teamID,
		}
		board, resp := th.Client.CreateBoard(newBoard)
		th.CheckOK(resp)
		require.NotNil(t, board)

		data := randomBytes(t, 1024)
		result, resp := th.Client.TeamUploadFile(teamID, board.ID, bytes.NewReader(data))
		th.CheckOK(resp)
		require.NotNil(t, result)
		require.NotEmpty(t, result.FileID)
		// TODO get the uploaded file
	})

	t.Run("user that doesn't belong to the board should not be able to upload a file", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember
		th.Client2 = clients.Viewer
		// Generate a valid Mattermost team ID (26 characters) for file operations
		teamID := mmModel.NewId()
		newBoard := &model.Board{
			Type:   model.BoardTypeOpen,
			TeamID: teamID,
		}
		board, resp := th.Client.CreateBoard(newBoard)
		th.CheckOK(resp)
		require.NotNil(t, board)

		data := randomBytes(t, 1024)

		// a user that doesn't belong to the board tries to upload the file
		result, resp := th.Client2.TeamUploadFile(teamID, board.ID, bytes.NewReader(data))
		th.CheckForbidden(resp)
		require.Nil(t, result)
	})
}
