package app

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
)

func TestApp_initializeTemplates(t *testing.T) {
	board := &model.Board{
		ID:              utils.NewID(utils.IDTypeBoard),
		TeamID:          model.GlobalTeamID,
		Type:            model.BoardTypeOpen,
		Title:           "test board",
		IsTemplate:      true,
		TemplateVersion: defaultTemplateVersion,
	}

	block := &model.Block{
		ID:       utils.NewID(utils.IDTypeBlock),
		ParentID: board.ID,
		BoardID:  board.ID,
		Type:     model.TypeText,
		Title:    "test text",
	}

	boardsAndBlocks := &model.BoardsAndBlocks{
		Boards: []*model.Board{board},
		Blocks: []*model.Block{block},
	}

	boardMember := &model.BoardMember{
		BoardID: board.ID,
		UserID:  "test-user",
	}

	t.Run("Needs template init", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.Store.EXPECT().GetTemplateBoards(model.GlobalTeamID, "").Return([]*model.Board{}, nil)
		th.Store.EXPECT().RemoveDefaultTemplates([]*model.Board{}).Return(nil)
		th.Store.EXPECT().CreateBoardsAndBlocks(gomock.Any(), gomock.Any()).AnyTimes().Return(boardsAndBlocks, nil)
		th.Store.EXPECT().GetMembersForBoard(board.ID).AnyTimes().Return([]*model.BoardMember{}, nil)
		th.Store.EXPECT().GetBoard(board.ID).AnyTimes().Return(board, nil)
		th.Store.EXPECT().GetMemberForBoard(gomock.Any(), gomock.Any()).AnyTimes().Return(boardMember, nil)
		th.Store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil).AnyTimes()

		th.FilesBackend.On("WriteFile", mock.Anything, mock.Anything).Return(int64(1), nil)

		done, err := th.App.initializeTemplates()
		require.NoError(t, err, "initializeTemplates should not error")
		require.True(t, done, "initialization was needed")
	})

	t.Run("Skip template init", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.Store.EXPECT().GetTemplateBoards(model.GlobalTeamID, "").Return([]*model.Board{board}, nil)

		done, err := th.App.initializeTemplates()
		require.NoError(t, err, "initializeTemplates should not error")
		require.False(t, done, "initialization was not needed")
	})
}
