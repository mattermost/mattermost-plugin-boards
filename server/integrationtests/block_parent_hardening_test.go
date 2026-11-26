// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/stretchr/testify/require"
)

// TestBlockParentErrorIsSameForMissingAndOtherBoardParent checks that a rejected
// parent gives the same response whether the parent does not exist or exists on
// a board the caller cannot read.
func TestBlockParentErrorIsSameForMissingAndOtherBoardParent(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	data := setupBlockParentTestData(t, th)
	missingID := utils.NewID(utils.IDTypeBlock)

	normalize := func(err error, parentID string) string {
		require.Error(t, err)
		return strings.ReplaceAll(err.Error(), parentID, "<parent>")
	}

	t.Run("create", func(t *testing.T) {
		_, respOther := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{newComment(data.boardA.ID, data.cardB.ID, "c1")}, false)
		_, respMissing := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{newComment(data.boardA.ID, missingID, "c2")}, false)

		require.Equal(t, http.StatusBadRequest, respOther.StatusCode)
		require.Equal(t, respOther.StatusCode, respMissing.StatusCode)

		// the server regenerates block IDs on create, so mask them too
		require.Equal(t,
			maskBlockIDs(normalize(respMissing.Error, missingID)),
			maskBlockIDs(normalize(respOther.Error, data.cardB.ID)),
			"response reveals whether the parent block exists",
		)
	})

	t.Run("patch", func(t *testing.T) {
		otherParent := data.cardB.ID
		_, respOther := th.Client.PatchBlock(data.boardA.ID, data.commentOnA.ID, &model.BlockPatch{ParentID: &otherParent}, false)
		_, respMissing := th.Client.PatchBlock(data.boardA.ID, data.commentOnA.ID, &model.BlockPatch{ParentID: &missingID}, false)

		require.Equal(t, http.StatusBadRequest, respOther.StatusCode)
		require.Equal(t, respOther.StatusCode, respMissing.StatusCode)
		require.Equal(t,
			normalize(respMissing.Error, missingID),
			normalize(respOther.Error, data.cardB.ID),
			"response reveals whether the parent block exists",
		)
	})
}

// maskBlockIDs replaces every 27 character server-generated ID with a placeholder.
func maskBlockIDs(s string) string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	for _, f := range fields {
		if len(f) == 27 {
			s = strings.ReplaceAll(s, f, "<id>")
		}
	}
	return s
}

// TestCreateBoardsAndBlocksDoesNotKeepDeletedParentFromOtherBoard checks that a
// block created through POST /boards-and-blocks is not stored with a parent that
// is a deleted card on another board.
func TestCreateBoardsAndBlocksDoesNotKeepDeletedParentFromOtherBoard(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	data := setupBlockParentTestData(t, th)
	testTeamID, _, _ := th.GetTestTeamIDs()

	require.NoError(t, th.Server.App().DeleteBlock(data.cardB.ID, userAdminID))
	_, err := th.Server.App().GetBlockByID(data.cardB.ID)
	require.True(t, model.IsErrNotFound(err), "precondition: card B is deleted")

	newBoardID := utils.NewID(utils.IDTypeBoard)
	title := "comment under deleted card on another board " + utils.NewID(utils.IDTypeNone)
	bab := &model.BoardsAndBlocks{
		Boards: []*model.Board{{ID: newBoardID, Title: "new board", TeamID: testTeamID, Type: model.BoardTypePrivate}},
		Blocks: []*model.Block{{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  newBoardID,
			ParentID: data.cardB.ID,
			Type:     model.TypeComment,
			Title:    title,
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		}},
	}

	newBab, resp := th.Client.CreateBoardsAndBlocks(bab)
	if resp.Error != nil {
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		return
	}
	require.NotNil(t, newBab)
	require.Len(t, newBab.Boards, 1)

	stored, err := th.Server.App().GetBlocksForBoard(newBab.Boards[0].ID)
	require.NoError(t, err)
	for _, b := range stored {
		require.NotEqual(t, data.cardB.ID, b.ParentID, "block stored with a parent on another board")
	}
}
