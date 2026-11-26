// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

func TestGetBoardAndCardStaysOnBlockBoard(t *testing.T) {
	boardA := &model.Board{ID: "board-a", TeamID: "team-id"}
	boardB := &model.Board{ID: "board-b", TeamID: "team-id"}

	cardA := &model.Block{ID: "card-a", BoardID: boardA.ID, ParentID: boardA.ID, Type: model.TypeCard, Title: "card A"}
	cardB := &model.Block{ID: "card-b", BoardID: boardB.ID, ParentID: boardB.ID, Type: model.TypeCard, Title: "card B"}
	textB := &model.Block{ID: "text-b", BoardID: boardB.ID, ParentID: cardB.ID, Type: model.TypeText}

	storedBlocks := map[string]*model.Block{
		cardA.ID: cardA,
		cardB.ID: cardB,
		textB.ID: textB,
	}
	storedBoards := map[string]*model.Board{
		boardA.ID: boardA,
		boardB.ID: boardB,
	}

	testCases := []struct {
		name         string
		block        *model.Block
		expectedCard *model.Block
	}{
		{
			name:         "comment whose parent is a card on another board",
			block:        &model.Block{ID: "comment-1", BoardID: boardA.ID, ParentID: cardB.ID, Type: model.TypeComment},
			expectedCard: nil,
		},
		{
			name:         "comment whose grandparent is a card on another board",
			block:        &model.Block{ID: "comment-2", BoardID: boardA.ID, ParentID: textB.ID, Type: model.TypeComment},
			expectedCard: nil,
		},
		{
			name:         "comment whose parent is a card on the same board",
			block:        &model.Block{ID: "comment-3", BoardID: boardA.ID, ParentID: cardA.ID, Type: model.TypeComment},
			expectedCard: cardA,
		},
		{
			name:         "card whose parent is its own board",
			block:        cardA,
			expectedCard: cardA,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			th, tearDown := SetupTestHelper(t)
			defer tearDown()

			th.Store.EXPECT().GetBoard(gomock.Any()).DoAndReturn(func(id string) (*model.Board, error) {
				if b, ok := storedBoards[id]; ok {
					return b, nil
				}
				return nil, model.NewErrNotFound("board ID=" + id)
			}).AnyTimes()
			th.Store.EXPECT().GetBlock(gomock.Any()).DoAndReturn(func(id string) (*model.Block, error) {
				if b, ok := storedBlocks[id]; ok {
					return b, nil
				}
				return nil, model.NewErrNotFound("block ID=" + id)
			}).AnyTimes()

			board, card, err := th.App.getBoardAndCard(tc.block)
			if tc.expectedCard == nil {
				// Either outcome stops a notification from pairing board A with card B:
				// an error, or board A with no card.
				if err != nil {
					return
				}
				require.NotNil(t, board)
				require.Equal(t, boardA.ID, board.ID)
				require.Nil(t, card, "card resolved from a block on another board")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, board)
			require.Equal(t, boardA.ID, board.ID)
			require.NotNil(t, card)
			require.Equal(t, tc.expectedCard.ID, card.ID)
			require.Equal(t, board.ID, card.BoardID)
		})
	}
}
