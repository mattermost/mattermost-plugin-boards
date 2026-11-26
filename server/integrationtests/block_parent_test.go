// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/stretchr/testify/require"
)

type blockParentTestData struct {
	boardA     *model.Board
	cardA      *model.Block
	boardB     *model.Board
	cardB      *model.Block
	commentOnA *model.Block
}

// setupBlockParentTestData creates board A (owned by the team member client) with
// a card and a comment on it, and board B (private, owned by another user) with a
// card. The team member is not a member of board B.
func setupBlockParentTestData(t *testing.T, th *TestHelper) blockParentTestData {
	testTeamID, _, _ := th.GetTestTeamIDs()
	require.NotEmpty(t, testTeamID)

	boardA := th.CreateBoard(testTeamID, model.BoardTypePrivate)

	cards, resp := th.Client.InsertBlocks(boardA.ID, []*model.Block{{
		ID:       utils.NewID(utils.IDTypeBlock),
		BoardID:  boardA.ID,
		ParentID: boardA.ID,
		CreateAt: 1,
		UpdateAt: 1,
		Type:     model.TypeCard,
		Title:    "card on board A",
	}}, true)
	th.CheckOK(resp)
	require.Len(t, cards, 1)
	cardA := cards[0]

	comments, resp := th.Client.InsertBlocks(boardA.ID, []*model.Block{{
		ID:       utils.NewID(utils.IDTypeBlock),
		BoardID:  boardA.ID,
		ParentID: cardA.ID,
		CreateAt: 1,
		UpdateAt: 1,
		Type:     model.TypeComment,
		Title:    "comment on card A",
	}}, true)
	th.CheckOK(resp)
	require.Len(t, comments, 1)
	commentOnA := comments[0]

	boardB, err := th.Server.App().CreateBoard(
		&model.Board{Title: "Board B", TeamID: testTeamID, Type: model.BoardTypePrivate, MinimumRole: "viewer"},
		userAdminID,
		true,
	)
	require.NoError(t, err)

	cardB := &model.Block{
		ID:       utils.NewID(utils.IDTypeBlock),
		BoardID:  boardB.ID,
		ParentID: boardB.ID,
		Type:     model.TypeCard,
		Title:    "card on board B",
		Fields:   map[string]interface{}{},
	}
	err = th.Server.App().InsertBlock(cardB, userAdminID)
	require.NoError(t, err)

	// precondition: the team member cannot read board B
	_, resp = th.Client.GetBlocksForBoard(boardB.ID)
	require.Error(t, resp.Error)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	return blockParentTestData{
		boardA:     boardA,
		cardA:      cardA,
		boardB:     boardB,
		cardB:      cardB,
		commentOnA: commentOnA,
	}
}

func newComment(boardID, parentID, title string) *model.Block {
	return &model.Block{
		ID:       utils.NewID(utils.IDTypeBlock),
		BoardID:  boardID,
		ParentID: parentID,
		CreateAt: 1,
		UpdateAt: 1,
		Type:     model.TypeComment,
		Title:    title,
	}
}

func findBlockByTitle(blocks []*model.Block, title string) *model.Block {
	for _, b := range blocks {
		if b.Title == title {
			return b
		}
	}
	return nil
}

func TestBlockParentMustBeOnSameBoard(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	data := setupBlockParentTestData(t, th)

	t.Run("create comment with parent on another board is rejected", func(t *testing.T) {
		title := "comment with parent on another board " + utils.NewID(utils.IDTypeNone)
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{newComment(data.boardA.ID, data.cardB.ID, title)}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Empty(t, newBlocks)

		blocks, resp := th.Client.GetBlocksForBoard(data.boardA.ID)
		th.CheckOK(resp)
		require.Nil(t, findBlockByTitle(blocks, title), "block with a parent on another board must not be stored")
	})

	t.Run("create card with parent on another board is rejected", func(t *testing.T) {
		title := "card with parent on another board " + utils.NewID(utils.IDTypeNone)
		card := &model.Block{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  data.boardA.ID,
			ParentID: data.cardB.ID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeCard,
			Title:    title,
		}
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{card}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Empty(t, newBlocks)

		blocks, resp := th.Client.GetBlocksForBoard(data.boardA.ID)
		th.CheckOK(resp)
		require.Nil(t, findBlockByTitle(blocks, title))
	})

	t.Run("create comment with parent set to another board ID is rejected", func(t *testing.T) {
		title := "comment with parent set to another board ID " + utils.NewID(utils.IDTypeNone)
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{newComment(data.boardA.ID, data.boardB.ID, title)}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Empty(t, newBlocks)

		blocks, resp := th.Client.GetBlocksForBoard(data.boardA.ID)
		th.CheckOK(resp)
		require.Nil(t, findBlockByTitle(blocks, title))
	})

	t.Run("batch with one block whose parent is on another board is rejected as a whole", func(t *testing.T) {
		okTitle := "comment with parent on same board " + utils.NewID(utils.IDTypeNone)
		badTitle := "comment with parent on another board " + utils.NewID(utils.IDTypeNone)
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{
			newComment(data.boardA.ID, data.cardA.ID, okTitle),
			newComment(data.boardA.ID, data.cardB.ID, badTitle),
		}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Empty(t, newBlocks)

		blocks, resp := th.Client.GetBlocksForBoard(data.boardA.ID)
		th.CheckOK(resp)
		require.Nil(t, findBlockByTitle(blocks, okTitle))
		require.Nil(t, findBlockByTitle(blocks, badTitle))
	})

	t.Run("commenter creating comment with parent on another board is rejected", func(t *testing.T) {
		_, err := th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: data.boardA.ID, UserID: userCommenterID, SchemeCommenter: true})
		require.NoError(t, err)

		title := "commenter comment with parent on another board " + utils.NewID(utils.IDTypeNone)
		newBlocks, resp := clients.Commenter.InsertBlocks(data.boardA.ID, []*model.Block{newComment(data.boardA.ID, data.cardB.ID, title)}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Empty(t, newBlocks)

		blocks, resp := th.Client.GetBlocksForBoard(data.boardA.ID)
		th.CheckOK(resp)
		require.Nil(t, findBlockByTitle(blocks, title))
	})

	t.Run("patch comment parent to a card on another board is rejected", func(t *testing.T) {
		newParent := data.cardB.ID
		_, resp := th.Client.PatchBlock(data.boardA.ID, data.commentOnA.ID, &model.BlockPatch{ParentID: &newParent}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		block, err := th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, data.cardA.ID, block.ParentID)
	})

	t.Run("patch comment parent to another board ID is rejected", func(t *testing.T) {
		newParent := data.boardB.ID
		_, resp := th.Client.PatchBlock(data.boardA.ID, data.commentOnA.ID, &model.BlockPatch{ParentID: &newParent}, false)
		require.Error(t, resp.Error)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		block, err := th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, data.cardA.ID, block.ParentID)
	})

	t.Run("batch patch comment parent to a card on another board is rejected", func(t *testing.T) {
		newParent := data.cardB.ID
		batch := model.BlockPatchBatch{
			BlockIDs:     []string{data.commentOnA.ID},
			BlockPatches: []model.BlockPatch{{ParentID: &newParent}},
		}
		body, err := json.Marshal(batch)
		require.NoError(t, err)

		res, err := th.Client.DoAPIPatch("/boards/"+data.boardA.ID+"/blocks", string(body))
		if res != nil {
			defer res.Body.Close()
		}
		require.Error(t, err)
		require.NotNil(t, res)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)

		block, err := th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, data.cardA.ID, block.ParentID)
	})

	t.Run("batch patch with one foreign parent leaves every block unchanged", func(t *testing.T) {
		newTitle := "retitled " + utils.NewID(utils.IDTypeNone)
		newParent := data.cardB.ID
		batch := model.BlockPatchBatch{
			BlockIDs:     []string{data.cardA.ID, data.commentOnA.ID},
			BlockPatches: []model.BlockPatch{{Title: &newTitle}, {ParentID: &newParent}},
		}
		body, err := json.Marshal(batch)
		require.NoError(t, err)

		res, err := th.Client.DoAPIPatch("/boards/"+data.boardA.ID+"/blocks", string(body))
		if res != nil {
			defer res.Body.Close()
		}
		require.Error(t, err)
		require.NotNil(t, res)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)

		comment, err := th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, data.cardA.ID, comment.ParentID)

		card, err := th.Server.App().GetBlockByID(data.cardA.ID)
		require.NoError(t, err)
		require.NotEqual(t, newTitle, card.Title)
	})
}

// TestBlockParentOnSameBoardIsAllowed covers parent values that must keep working.
func TestBlockParentOnSameBoardIsAllowed(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	data := setupBlockParentTestData(t, th)

	t.Run("create comment with parent card on the same board", func(t *testing.T) {
		title := "comment on same board card " + utils.NewID(utils.IDTypeNone)
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{newComment(data.boardA.ID, data.cardA.ID, title)}, false)
		th.CheckOK(resp)
		require.Len(t, newBlocks, 1)
		require.Equal(t, data.cardA.ID, newBlocks[0].ParentID)
	})

	t.Run("create card with parent set to its own board ID", func(t *testing.T) {
		card := &model.Block{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  data.boardA.ID,
			ParentID: data.boardA.ID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeCard,
			Title:    "card with board parent",
		}
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{card}, false)
		th.CheckOK(resp)
		require.Len(t, newBlocks, 1)
		require.Equal(t, data.boardA.ID, newBlocks[0].ParentID)
	})

	t.Run("create block with empty parent", func(t *testing.T) {
		card := &model.Block{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  data.boardA.ID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeCard,
			Title:    "card with empty parent",
		}
		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{card}, false)
		th.CheckOK(resp)
		require.Len(t, newBlocks, 1)
		require.Empty(t, newBlocks[0].ParentID)
	})

	t.Run("create batch where parent is another new block in the same batch", func(t *testing.T) {
		tempCardID := utils.NewID(utils.IDTypeBlock)
		card := &model.Block{
			ID:       tempCardID,
			BoardID:  data.boardA.ID,
			ParentID: data.boardA.ID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeCard,
			Title:    "batch card",
		}
		comment := newComment(data.boardA.ID, tempCardID, "batch comment")
		text := &model.Block{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  data.boardA.ID,
			ParentID: tempCardID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeText,
			Title:    "batch text",
		}

		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{card, comment, text}, false)
		th.CheckOK(resp)
		require.Len(t, newBlocks, 3)

		newCard := findBlockByTitle(newBlocks, "batch card")
		gotComment := findBlockByTitle(newBlocks, "batch comment")
		newText := findBlockByTitle(newBlocks, "batch text")
		require.NotNil(t, newCard)
		require.NotNil(t, gotComment)
		require.NotNil(t, newText)
		require.NotEqual(t, tempCardID, newCard.ID)
		require.Equal(t, newCard.ID, gotComment.ParentID)
		require.Equal(t, newCard.ID, newText.ParentID)
	})

	t.Run("batch reusing a foreign card ID as a temporary ID is remapped onto the same board", func(t *testing.T) {
		card := &model.Block{
			ID:       data.cardB.ID,
			BoardID:  data.boardA.ID,
			ParentID: data.boardA.ID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeCard,
			Title:    "batch card reusing id",
		}
		comment := newComment(data.boardA.ID, data.cardB.ID, "batch comment reusing id")

		newBlocks, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{card, comment}, false)
		th.CheckOK(resp)
		require.Len(t, newBlocks, 2)

		newCard := findBlockByTitle(newBlocks, "batch card reusing id")
		gotComment := findBlockByTitle(newBlocks, "batch comment reusing id")
		require.NotNil(t, newCard)
		require.NotNil(t, gotComment)
		require.NotEqual(t, data.cardB.ID, newCard.ID)
		require.Equal(t, newCard.ID, gotComment.ParentID)
	})

	t.Run("patch comment parent to another card on the same board", func(t *testing.T) {
		cards, resp := th.Client.InsertBlocks(data.boardA.ID, []*model.Block{{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  data.boardA.ID,
			ParentID: data.boardA.ID,
			CreateAt: 1,
			UpdateAt: 1,
			Type:     model.TypeCard,
			Title:    "second card on board A",
		}}, true)
		th.CheckOK(resp)
		require.Len(t, cards, 1)
		secondCard := cards[0]

		newParent := secondCard.ID
		_, resp = th.Client.PatchBlock(data.boardA.ID, data.commentOnA.ID, &model.BlockPatch{ParentID: &newParent}, false)
		th.CheckOK(resp)

		block, err := th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, secondCard.ID, block.ParentID)

		// batch patch back to the original card
		originalParent := data.cardA.ID
		body, err := json.Marshal(model.BlockPatchBatch{
			BlockIDs:     []string{data.commentOnA.ID},
			BlockPatches: []model.BlockPatch{{ParentID: &originalParent}},
		})
		require.NoError(t, err)
		res, err := th.Client.DoAPIPatch("/boards/"+data.boardA.ID+"/blocks", string(body))
		if res != nil {
			defer res.Body.Close()
		}
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)

		block, err = th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, data.cardA.ID, block.ParentID)
	})

	t.Run("patch card parent to its own board ID", func(t *testing.T) {
		newParent := data.boardA.ID
		_, resp := th.Client.PatchBlock(data.boardA.ID, data.cardA.ID, &model.BlockPatch{ParentID: &newParent}, false)
		th.CheckOK(resp)

		block, err := th.Server.App().GetBlockByID(data.cardA.ID)
		require.NoError(t, err)
		require.Equal(t, data.boardA.ID, block.ParentID)
	})

	t.Run("patch without parent keeps the existing parent", func(t *testing.T) {
		newTitle := "updated comment"
		_, resp := th.Client.PatchBlock(data.boardA.ID, data.commentOnA.ID, &model.BlockPatch{Title: &newTitle}, false)
		th.CheckOK(resp)

		block, err := th.Server.App().GetBlockByID(data.commentOnA.ID)
		require.NoError(t, err)
		require.Equal(t, data.cardA.ID, block.ParentID)
		require.Equal(t, newTitle, block.Title)
	})
}
