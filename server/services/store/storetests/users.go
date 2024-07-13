// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package storetests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

//nolint:dupl
func StoreTestUserStore(t *testing.T, setup func(t *testing.T) (store.Store, func())) {

	t.Run("CreateAndGetUser", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()
		testCreateAndGetUser(t, store)
	})
}

func testCreateAndGetUser(t *testing.T, store store.Store) {
	user := &model.User{
		ID:       utils.NewID(utils.IDTypeUser),
		Username: "damao",
		Email:    "mock@email.com",
	}

	t.Run("GetUserByID", func(t *testing.T) {
		got, err := store.GetUserByID(user.ID)
		require.NoError(t, err)
		require.Equal(t, user.ID, got.ID)
		require.Equal(t, user.Username, got.Username)
		require.Equal(t, user.Email, got.Email)
	})

	t.Run("GetUserByID nonexistent", func(t *testing.T) {
		got, err := store.GetUserByID("nonexistent-id")
		var nf *model.ErrNotFound
		require.ErrorAs(t, err, &nf)
		require.Nil(t, got)
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		got, err := store.GetUserByUsername(user.Username)
		require.NoError(t, err)
		require.Equal(t, user.ID, got.ID)
		require.Equal(t, user.Username, got.Username)
		require.Equal(t, user.Email, got.Email)
	})

	t.Run("GetUserByUsername nonexistent", func(t *testing.T) {
		got, err := store.GetUserByID("nonexistent-username")
		var nf *model.ErrNotFound
		require.ErrorAs(t, err, &nf)
		require.Nil(t, got)
	})

	t.Run("GetUserByEmail", func(t *testing.T) {
		got, err := store.GetUserByEmail(user.Email)
		require.NoError(t, err)
		require.Equal(t, user.ID, got.ID)
		require.Equal(t, user.Username, got.Username)
		require.Equal(t, user.Email, got.Email)
	})

	t.Run("GetUserByEmail nonexistent", func(t *testing.T) {
		got, err := store.GetUserByID("nonexistent-email")
		var nf *model.ErrNotFound
		require.ErrorAs(t, err, &nf)
		require.Nil(t, got)
	})
}
