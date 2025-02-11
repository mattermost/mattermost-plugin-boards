// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package storetests

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/stretchr/testify/require"
)

func StoreTestSessionStore(t *testing.T, setup func(t *testing.T) (store.Store, func())) {
	t.Run("GetActiveUserCount", func(t *testing.T) {
		store, tearDown := setup(t)
		defer tearDown()
		testGetActiveUserCount(t, store)
	})
}

func testGetActiveUserCount(t *testing.T, store store.Store) {
	t.Run("no active user", func(t *testing.T) {
		count, err := store.GetActiveUserCount(60)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("active user", func(t *testing.T) {
		// gen random count active user session
		count := int(time.Now().Unix() % 10)

		got, err := store.GetActiveUserCount(60)
		require.NoError(t, err)
		require.Equal(t, count, got)
	})
}
