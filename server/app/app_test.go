// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/stretchr/testify/require"
)

func TestSetConfig(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	t.Run("Test Update Config", func(t *testing.T) {
		require.False(t, th.App.config.EnablePublicSharedBoards)
		newConfiguration := config.Configuration{}
		newConfiguration.EnablePublicSharedBoards = true
		th.App.SetConfig(&newConfiguration)

		require.True(t, th.App.config.EnablePublicSharedBoards)
	})
}
