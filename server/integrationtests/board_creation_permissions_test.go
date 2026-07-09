// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

// TestBoardCreationPermissionEnforcement verifies that the per-type board
// creation permission (create_public_channel / create_private_channel) is
// enforced consistently across every board creation endpoint, guarding against
// a regression where /boards-and-blocks and the archive import endpoints
// skipped the check (MM-69622).
func TestBoardCreationPermissionEnforcement(t *testing.T) {
	t.Run("createBoard honors revoked create permissions", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()
		clients := setupClients(th)
		th.Client = clients.TeamMember

		teamID := mmModel.NewId()
		th.PermissionAPI.DenyTeamPermission(userTeamMember, model.PermissionCreatePublicChannel)
		th.PermissionAPI.DenyTeamPermission(userTeamMember, model.PermissionCreatePrivateChannel)

		_, resp := th.Client.CreateBoard(&model.Board{Title: "public", TeamID: teamID, Type: model.BoardTypeOpen})
		th.CheckForbidden(resp)

		_, resp = th.Client.CreateBoard(&model.Board{Title: "private", TeamID: teamID, Type: model.BoardTypePrivate})
		th.CheckForbidden(resp)
	})

	t.Run("createBoardsAndBlocks honors revoked create permissions", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()
		clients := setupClients(th)
		th.Client = clients.TeamMember

		teamID := mmModel.NewId()

		// Revoke only the private board permission: creating a private board
		// must be denied while a public board is still allowed.
		th.PermissionAPI.DenyTeamPermission(userTeamMember, model.PermissionCreatePrivateChannel)

		_, resp := th.Client.CreateBoardsAndBlocks(newBoardAndBlocks(teamID, model.BoardTypePrivate))
		th.CheckForbidden(resp)

		bab, resp := th.Client.CreateBoardsAndBlocks(newBoardAndBlocks(teamID, model.BoardTypeOpen))
		th.CheckOK(resp)
		require.Len(t, bab.Boards, 1)

		// Revoke the public permission too: now both types are denied.
		th.PermissionAPI.DenyTeamPermission(userTeamMember, model.PermissionCreatePublicChannel)

		_, resp = th.Client.CreateBoardsAndBlocks(newBoardAndBlocks(teamID, model.BoardTypeOpen))
		th.CheckForbidden(resp)
	})

	t.Run("archiveImport honors revoked create permissions", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()
		clients := setupClients(th)
		th.Client = clients.TeamMember

		teamID := mmModel.NewId()

		// Build a source private board and export it so we have a valid archive.
		sourceBab, resp := th.Client.CreateBoardsAndBlocks(newBoardAndBlocks(teamID, model.BoardTypePrivate))
		th.CheckOK(resp)
		require.Len(t, sourceBab.Boards, 1)

		archive, resp := th.Client.ExportBoardArchive(sourceBab.Boards[0].ID)
		th.CheckOK(resp)
		require.NotEmpty(t, archive)

		// Revoke the private board permission and confirm the import is blocked.
		th.PermissionAPI.DenyTeamPermission(userTeamMember, model.PermissionCreatePrivateChannel)

		resp = th.Client.ImportArchive(teamID, bytes.NewReader(archive))
		th.CheckForbidden(resp)
	})
}

func newBoardAndBlocks(teamID string, boardType model.BoardType) *model.BoardsAndBlocks {
	boardID := utils.NewID(utils.IDTypeBoard)
	return &model.BoardsAndBlocks{
		Boards: []*model.Board{{
			ID:     boardID,
			Title:  "Test Board",
			TeamID: teamID,
			Type:   boardType,
		}},
		Blocks: []*model.Block{{
			ID:       utils.NewID(utils.IDTypeCard),
			Title:    "Test Block",
			BoardID:  boardID,
			Type:     model.TypeCard,
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		}},
	}
}
