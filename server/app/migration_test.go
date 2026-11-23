// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"

	"github.com/stretchr/testify/require"
)

type fakeMigrationLookup struct {
	info model.BoardMigration
}

func (f fakeMigrationLookup) Get(_, _ string) model.BoardMigration {
	return f.info
}

func TestGetBoardMigration(t *testing.T) {
	moved := model.BoardMigration{Moved: true, ChannelID: "chan", TeamName: "eng", ChannelName: "roadmap", CanOpen: true}

	t.Run("reports not moved when Boards v2 lookup is unavailable", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()
		th.App.config.LegacyMode = true

		require.Equal(t, model.BoardMigration{}, th.App.GetBoardMigration("board", "user"))
	})

	t.Run("passes through a moved board and is not read-only by default", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()
		th.App.config.LegacyMode = true
		th.App.migrationLookup = fakeMigrationLookup{info: moved}

		require.Equal(t, moved, th.App.GetBoardMigration("board", "user"))
	})

	t.Run("is read-only for a moved board when the setting is on", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()
		th.App.config.LegacyMode = true
		th.App.migrationLookup = fakeMigrationLookup{info: moved}
		th.App.config.MovedBoardsReadOnly = true

		info := th.App.GetBoardMigration("board", "user")
		require.True(t, info.Moved)
		require.True(t, info.ReadOnly)
	})

	t.Run("is not read-only for a board that did not move", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()
		th.App.config.LegacyMode = true
		th.App.migrationLookup = fakeMigrationLookup{}
		th.App.config.MovedBoardsReadOnly = true

		require.Equal(t, model.BoardMigration{}, th.App.GetBoardMigration("board", "user"))
	})

	t.Run("reports not moved and skips the lookup outside Legacy Mode", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()
		lookup := &countingMigrationLookup{info: moved}
		th.App.migrationLookup = lookup
		th.App.config.LegacyMode = false
		th.App.config.MovedBoardsReadOnly = true

		require.Equal(t, model.BoardMigration{}, th.App.GetBoardMigration("board", "user"))
		require.Zero(t, lookup.calls)
	})
}

type countingMigrationLookup struct {
	info  model.BoardMigration
	calls int
}

func (f *countingMigrationLookup) Get(_, _ string) model.BoardMigration {
	f.calls++
	return f.info
}
