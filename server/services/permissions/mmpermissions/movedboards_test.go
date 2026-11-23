// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmpermissions

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	mmpermissionsMocks "github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mmpermissions/mocks"
	permissionsMocks "github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mocks"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type fakeMovedBoards struct {
	moved map[string]bool
	calls int
}

func (f *fakeMovedBoards) IsMoved(boardID string) bool {
	f.calls++
	return f.moved[boardID]
}

func TestHasPermissionToBoardMovedBoardsReadOnly(t *testing.T) {
	writePermissions := []*mmModel.Permission{
		model.PermissionManageBoardType,
		model.PermissionDeleteBoard,
		model.PermissionManageBoardRoles,
		model.PermissionShareBoard,
		model.PermissionDeleteOthersComments,
		model.PermissionManageBoardCards,
		model.PermissionManageBoardProperties,
		model.PermissionCommentBoardCards,
	}

	setupWithEnabled := func(t *testing.T, enabled func() bool, moved bool, teamAdmin bool) (*Service, *fakeMovedBoards) {
		ctrl := gomock.NewController(t)
		store := permissionsMocks.NewMockStore(ctrl)
		api := mmpermissionsMocks.NewMockAPI(ctrl)
		checker := &fakeMovedBoards{moved: map[string]bool{testBoardID: moved}}

		store.EXPECT().GetBoard(testBoardID).Return(&model.Board{ID: testBoardID, TeamID: testTeamID}, nil).AnyTimes()
		store.EXPECT().GetMemberForBoard(testBoardID, testUserID).
			Return(&model.BoardMember{BoardID: testBoardID, UserID: testUserID, SchemeAdmin: true}, nil).AnyTimes()
		store.EXPECT().GetUserByID(testUserID).Return(&model.User{ID: testUserID}, nil).AnyTimes()
		api.EXPECT().HasPermissionToTeam(testUserID, testTeamID, model.PermissionViewTeam).Return(true).AnyTimes()
		api.EXPECT().HasPermissionToTeam(testUserID, testTeamID, model.PermissionManageTeam).Return(teamAdmin).AnyTimes()

		svc := New(store, api, mlog.CreateConsoleTestLogger(t),
			WithMovedBoardsReadOnly(enabled, checker))
		return svc, checker
	}

	setup := func(t *testing.T, enabled bool, moved bool, teamAdmin bool) (*Service, *fakeMovedBoards) {
		return setupWithEnabled(t, func() bool { return enabled }, moved, teamAdmin)
	}

	t.Run("denies every write on a moved board but still allows viewing", func(t *testing.T) {
		svc, _ := setup(t, true, true, false)

		assert.True(t, svc.HasPermissionToBoard(testUserID, testBoardID, model.PermissionViewBoard))
		for _, permission := range writePermissions {
			assert.Falsef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}
	})

	t.Run("also applies to team admins", func(t *testing.T) {
		svc, _ := setup(t, true, true, true)

		assert.True(t, svc.HasPermissionToBoard(testUserID, testBoardID, model.PermissionViewBoard))
		for _, permission := range writePermissions {
			assert.Falsef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}
	})

	t.Run("allows writes on a board that did not move", func(t *testing.T) {
		svc, _ := setup(t, true, false, false)

		for _, permission := range writePermissions {
			assert.Truef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}
	})

	t.Run("allows writes on a moved board when the setting is off and never looks it up", func(t *testing.T) {
		svc, checker := setup(t, false, true, false)

		for _, permission := range writePermissions {
			assert.Truef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}
		assert.Zero(t, checker.calls)
	})

	t.Run("follows the configuration: read-only only applies in Legacy Mode", func(t *testing.T) {
		cfg := &config.Configuration{LegacyMode: false, MovedBoardsReadOnly: true}
		svc, checker := setupWithEnabled(t, cfg.MovedBoardsReadOnlyActive, true, false)

		for _, permission := range writePermissions {
			assert.Truef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}
		assert.Zero(t, checker.calls)

		cfg.LegacyMode = true
		for _, permission := range writePermissions {
			assert.Falsef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}

		cfg.MovedBoardsReadOnly = false
		for _, permission := range writePermissions {
			assert.Truef(t, svc.HasPermissionToBoard(testUserID, testBoardID, permission), "permission %s", permission.Id)
		}
	})

	t.Run("does not look up moved status for view checks", func(t *testing.T) {
		svc, checker := setup(t, true, true, false)

		assert.True(t, svc.HasPermissionToBoard(testUserID, testBoardID, model.PermissionViewBoard))
		assert.Zero(t, checker.calls)
	})
}
