// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/client"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestUploadFile(t *testing.T) {
	t.Run("a non authenticated user should be rejected", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		// Use unauthenticated client
		th.Client = client.NewClient(th.Server.Config().ServerRoot, "")
		teamID := mmModel.NewId()
		file, resp := th.Client.TeamUploadFile(teamID, "test-board-id", bytes.NewBuffer([]byte("test")))
		th.CheckUnauthorized(resp)
		require.Nil(t, file)
	})

	t.Run("upload a file to an existing team and board without permissions", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember
		teamID := mmModel.NewId()
		file, resp := th.Client.TeamUploadFile(teamID, "not-valid-board", bytes.NewBuffer([]byte("test")))
		th.CheckForbidden(resp)
		require.Nil(t, file)
	})

	t.Run("upload a file to an existing team and board with permissions", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember
		teamID := mmModel.NewId()
		testBoard := th.CreateBoard(teamID, model.BoardTypeOpen)
		file, resp := th.Client.TeamUploadFile(teamID, testBoard.ID, bytes.NewBuffer([]byte("test")))
		th.CheckOK(resp)
		require.NoError(t, resp.Error)
		require.NotNil(t, file)
		require.NotNil(t, file.FileID)
	})

	t.Run("upload a file to an existing team and board with permissions but reaching the MaxFileLimit", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember
		teamID := mmModel.NewId()
		testBoard := th.CreateBoard(teamID, model.BoardTypeOpen)

		config := th.Server.App().GetConfig()
		config.MaxFileSize = 1
		th.Server.App().SetConfig(config)

		file, resp := th.Client.TeamUploadFile(teamID, testBoard.ID, bytes.NewBuffer([]byte("test")))
		th.CheckRequestEntityTooLarge(resp)
		require.Nil(t, file)

		config.MaxFileSize = 100000
		th.Server.App().SetConfig(config)

		file, resp = th.Client.TeamUploadFile(teamID, testBoard.ID, bytes.NewBuffer([]byte("test")))
		th.CheckOK(resp)
		require.NoError(t, resp.Error)
		require.NotNil(t, file)
		require.NotNil(t, file.FileID)
	})
}

func TestFileInfo(t *testing.T) {
	t.Run("Retrieving file info", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember
		teamID := mmModel.NewId()
		testBoard := th.CreateBoard(teamID, model.BoardTypeOpen)

		file, resp := th.Client.TeamUploadFile(teamID, testBoard.ID, bytes.NewBuffer([]byte("test file content")))
		th.CheckOK(resp)
		require.NoError(t, resp.Error)
		require.NotNil(t, file)
		require.NotNil(t, file.FileID)

		userID := th.GetUser1().ID
		imageBlock := &model.Block{
			ID:       utils.NewID(utils.IDTypeBlock),
			BoardID:  testBoard.ID,
			Type:     model.TypeImage,
			CreateAt: utils.GetMillis(),
			UpdateAt: utils.GetMillis(),
			Fields: map[string]interface{}{
				model.BlockFieldFileId: file.FileID,
			},
		}
		err := th.Server.App().InsertBlock(imageBlock, userID)
		require.NoError(t, err)

		// Now retrieve the file info - it should work because the file is referenced by a block
		fileInfo, resp := th.Client.TeamUploadFileInfo(teamID, testBoard.ID, file.FileID)
		th.CheckOK(resp)
		require.NotNil(t, fileInfo)
		require.NotNil(t, fileInfo.Id)
		fileIDWithoutPrefix := file.FileID
		if len(fileIDWithoutPrefix) > 0 && fileIDWithoutPrefix[0] == '7' {
			fileIDWithoutPrefix = fileIDWithoutPrefix[1:]
		}
		if idx := strings.LastIndex(fileIDWithoutPrefix, "."); idx != -1 {
			fileIDWithoutPrefix = fileIDWithoutPrefix[:idx]
		}
		require.Equal(t, fileIDWithoutPrefix, fileInfo.Id)
	})
}
