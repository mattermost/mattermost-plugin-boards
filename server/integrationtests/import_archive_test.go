// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestImportArchiveStripsGuestSchemeAdmin(t *testing.T) {
	t.Run("import archive with guest member schemeAdmin true", func(t *testing.T) {
		th := SetupTestHelperPluginMode(t)
		defer th.TearDown()

		clients := setupClients(th)
		th.Client = clients.TeamMember

		const boardTitle = "MM-68841 guest import archive test"
		teamID := mmModel.NewId()

		board := &model.Board{
			ID:        utils.NewID(utils.IDTypeBoard),
			TeamID:    teamID,
			Title:     boardTitle,
			CreatedBy: userTeamMember,
			Type:      model.BoardTypeOpen,
			CreateAt:  utils.GetMillis(),
			UpdateAt:  utils.GetMillis(),
		}
		block := &model.Block{
			ID:        utils.NewID(utils.IDTypeCard),
			ParentID:  board.ID,
			Type:      model.TypeCard,
			BoardID:   board.ID,
			Title:     "card",
			CreatedBy: userTeamMember,
			CreateAt:  utils.GetMillis(),
			UpdateAt:  utils.GetMillis(),
		}

		babs, resp := th.Client.CreateBoardsAndBlocks(&model.BoardsAndBlocks{
			Boards: []*model.Board{board},
			Blocks: []*model.Block{block},
		})
		th.CheckOK(resp)
		sourceBoard := babs.Boards[0]

		_, err := th.Server.App().AddMemberToBoard(&model.BoardMember{
			UserID:          userGuest,
			BoardID:         sourceBoard.ID,
			SchemeViewer:    true,
			SchemeCommenter: true,
			SchemeEditor:    true,
		})
		require.NoError(t, err)

		archive, resp := th.Client.ExportBoardArchive(sourceBoard.ID)
		th.CheckOK(resp)
		require.NotEmpty(t, archive)

		craftedArchive, err := setGuestSchemeAdminInArchive(archive, userGuest)
		require.NoError(t, err)

		resp = th.Client.ImportArchive(teamID, bytes.NewReader(craftedArchive))
		th.CheckOK(resp)
		require.NoError(t, resp.Error)

		boards, err := th.Server.App().GetBoardsForUserAndTeam(userTeamMember, teamID, true)
		require.NoError(t, err)

		var importedBoard *model.Board
		for _, b := range boards {
			if b.Title == boardTitle && b.ID != sourceBoard.ID {
				importedBoard = b
				break
			}
		}
		require.NotNil(t, importedBoard, "imported board with title %q not found", boardTitle)

		guestMember, err := th.Server.App().GetMemberForBoard(importedBoard.ID, userGuest)
		require.NoError(t, err)
		require.NotNil(t, guestMember)
		require.False(t, guestMember.SchemeAdmin)
		require.True(t, guestMember.SchemeViewer)
		require.True(t, guestMember.SchemeCommenter)
		require.True(t, guestMember.SchemeEditor)
	})
}

func setGuestSchemeAdminInArchive(archive []byte, guestUserID string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	zw := zip.NewWriter(out)

	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(rc)
		closeErr := rc.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}

		if strings.HasSuffix(f.Name, "board.jsonl") {
			content, err = patchBoardJSONLGuestSchemeAdmin(content, guestUserID)
			if err != nil {
				return nil, err
			}
		}

		w, err := zw.Create(f.Name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(content); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func patchBoardJSONLGuestSchemeAdmin(jsonl []byte, guestUserID string) ([]byte, error) {
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(jsonl))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var archiveLine model.ArchiveLine
		if err := json.Unmarshal(line, &archiveLine); err != nil {
			return nil, err
		}

		if archiveLine.Type == "boardMember" {
			var member model.BoardMember
			if err := json.Unmarshal(archiveLine.Data, &member); err != nil {
				return nil, err
			}
			if member.UserID == guestUserID {
				member.SchemeAdmin = true
				member.SchemeViewer = true
				member.SchemeCommenter = true
				member.SchemeEditor = true
				data, err := json.Marshal(&member)
				if err != nil {
					return nil, err
				}
				archiveLine.Data = data
			}
		}

		patched, err := json.Marshal(&archiveLine)
		if err != nil {
			return nil, err
		}
		if _, err := out.Write(patched); err != nil {
			return nil, err
		}
		if err := out.WriteByte('\n'); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
