// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package storetests

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	mmModel "github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

// dbHandle is an interface to access the database handle from the store
type dbHandle interface {
	DBHandle() *sql.DB
	DBType() string
}

//nolint:dupl
func StoreTestUserStore(t *testing.T, setup func(t *testing.T) (store.Store, func())) {
	t.Run("CreateAndGetUser", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()
		testCreateAndGetUser(t, store)
	})
}

func testCreateAndGetUser(t *testing.T, store store.Store) {
	// Mattermost user IDs are 26 characters (not 27 like blocks)
	// Use mmModel.NewId() directly to get a 26-character ID
	user := &model.User{
		ID:       mmModel.NewId(),
		Username: "damao",
		Email:    "mock@email.com",
		CreateAt: utils.GetMillis(),
		UpdateAt: utils.GetMillis(),
	}

	// Insert user into the Mattermost Users table
	// We need to access the underlying database to insert the user
	// The store interface doesn't expose DB access, so we'll use type assertion
	dbStore, ok := store.(dbHandle)
	require.True(t, ok, "store must implement dbHandle interface")

	// Get the database connection from the store
	db := dbStore.DBHandle()
	require.NotNil(t, db)

	// Insert user into Users table
	dbType := dbStore.DBType()
	var insertSQL string
	if dbType == model.PostgresDBType {
		insertSQL = `INSERT INTO users (id, username, email, createat, updateat, deleteat) VALUES ($1, $2, $3, $4, $5, $6)`
	} else {
		insertSQL = `INSERT INTO Users (Id, Username, Email, CreateAt, UpdateAt, DeleteAt) VALUES (?, ?, ?, ?, ?, ?)`
	}
	_, err := db.Exec(insertSQL, user.ID, user.Username, user.Email, user.CreateAt, user.UpdateAt, 0)
	require.NoError(t, err)

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
