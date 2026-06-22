// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
)

func TestSearchUsers(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()
	th.App.config.ShowEmailAddress = false
	th.App.config.ShowFullName = false

	teamID := "team-id-1"
	userID := "user-id-1"

	t.Run("return empty users", func(t *testing.T) {
		th.Store.EXPECT().SearchUsersByTeam(teamID, "", "", true, false, false).Return([]*model.User{}, nil)

		users, err := th.App.SearchTeamUsers(teamID, "", "", true)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(users))
	})

	t.Run("return user", func(t *testing.T) {
		th.Store.EXPECT().SearchUsersByTeam(teamID, "", "", true, false, false).Return([]*model.User{{ID: userID}}, nil)
		th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionManageTeam).Return(false).Times(1)
		th.API.EXPECT().HasPermissionTo(userID, model.PermissionManageSystem).Return(false).Times(1)

		users, err := th.App.SearchTeamUsers(teamID, "", "", true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, 0, len(users[0].Permissions))
	})

	t.Run("return team admin", func(t *testing.T) {
		th.Store.EXPECT().SearchUsersByTeam(teamID, "", "", true, false, false).Return([]*model.User{{ID: userID}}, nil)
		th.App.config.ShowEmailAddress = false
		th.App.config.ShowFullName = false
		th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionManageTeam).Return(true).Times(1)
		th.API.EXPECT().HasPermissionTo(userID, model.PermissionManageSystem).Return(false).Times(1)

		users, err := th.App.SearchTeamUsers(teamID, "", "", true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, users[0].Permissions[0], model.PermissionManageTeam.Id)
	})

	t.Run("return system admin", func(t *testing.T) {
		th.Store.EXPECT().SearchUsersByTeam(teamID, "", "", true, false, false).Return([]*model.User{{ID: userID}}, nil)
		th.App.config.ShowEmailAddress = false
		th.App.config.ShowFullName = false
		th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionManageTeam).Return(true).Times(1)
		th.API.EXPECT().HasPermissionTo(userID, model.PermissionManageSystem).Return(true).Times(1)

		users, err := th.App.SearchTeamUsers(teamID, "", "", true)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, users[0].Permissions[0], model.PermissionManageTeam.Id)
		assert.Equal(t, users[0].Permissions[1], model.PermissionManageSystem.Id)
	})

	t.Run("test user channels", func(t *testing.T) {
		channelID := "Channel1"
		th.Store.EXPECT().SearchUserChannels(teamID, userID, "").Return([]*mmModel.Channel{{Id: channelID}}, nil)
		th.API.EXPECT().HasPermissionToChannel(userID, channelID, model.PermissionCreatePost).Return(true).Times(1)

		channels, err := th.App.SearchUserChannels(teamID, userID, "")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(channels))
	})

	t.Run("test user channels- no permissions", func(t *testing.T) {
		channelID := "Channel1"
		th.Store.EXPECT().SearchUserChannels(teamID, userID, "").Return([]*mmModel.Channel{{Id: channelID}}, nil)
		th.API.EXPECT().HasPermissionToChannel(userID, channelID, model.PermissionCreatePost).Return(false).Times(1)

		channels, err := th.App.SearchUserChannels(teamID, userID, "")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(channels))
	})
}

func TestRevokeBoardAdminForGuest(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	userID := "user-id-1"

	t.Run("no-op for empty userID", func(t *testing.T) {
		assert.NoError(t, th.App.RevokeBoardAdminForGuest(""))
	})

	t.Run("no-op when user is not a guest", func(t *testing.T) {
		th.Store.EXPECT().GetUserByID(userID).Return(&model.User{ID: userID, IsGuest: false}, nil)

		assert.NoError(t, th.App.RevokeBoardAdminForGuest(userID))
	})

	t.Run("clears SchemeAdmin on every membership for a guest", func(t *testing.T) {
		members := []*model.BoardMember{
			{BoardID: "board-1", UserID: userID, SchemeAdmin: true, SchemeEditor: true},
			{BoardID: "board-2", UserID: userID, SchemeAdmin: false, SchemeEditor: true},
			{BoardID: "board-3", UserID: userID, SchemeAdmin: true},
		}

		th.Store.EXPECT().GetUserByID(userID).Return(&model.User{ID: userID, IsGuest: true}, nil)
		th.Store.EXPECT().GetMembersForUser(userID).Return(members, nil)

		th.Store.EXPECT().
			SaveMember(gomock.AssignableToTypeOf(&model.BoardMember{})).
			DoAndReturn(func(bm *model.BoardMember) (*model.BoardMember, error) {
				assert.False(t, bm.SchemeAdmin, "SchemeAdmin should be cleared for board %s", bm.BoardID)
				assert.Equal(t, userID, bm.UserID)
				return bm, nil
			}).
			Times(2)

		assert.NoError(t, th.App.RevokeBoardAdminForGuest(userID))
	})
}
