// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestBoardIsValid(t *testing.T) {
	t.Run("Should return nil for valid board", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      model.NewId(),
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: BoardRoleViewer,
			Title:       "Valid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValid()
		require.NoError(t, err)
	})

	t.Run("Should return error for using global team ID for valid board", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      GlobalTeamID,
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: BoardRoleViewer,
			Title:       "Valid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-team-id")
	})

	t.Run("Should return error for invalid team ID", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      "invalid-team-id",
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: BoardRoleViewer,
			Title:       "Invalid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-team-id")
	})

	t.Run("Should return error for invalid board type", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      model.NewId(),
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        "invalid-type",
			MinimumRole: BoardRoleViewer,
			Title:       "Invalid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-board-type")
	})

	t.Run("Should return error for invalid minimum role", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      model.NewId(),
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: "invalid-role",
			Title:       "Invalid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-board-minimum-role")
	})
}

func TestBoardIsValidForImport(t *testing.T) {
	t.Run("Should return nil for valid board", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      model.NewId(),
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: BoardRoleViewer,
			Title:       "Valid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValidForImport()
		require.NoError(t, err)
	})

	t.Run("Should not error for using global team ID for valid board", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      GlobalTeamID,
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: BoardRoleViewer,
			Title:       "Valid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValidForImport()
		require.NoError(t, err)
	})

	t.Run("Should return error for invalid team ID", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      "invalid-team-id",
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: BoardRoleViewer,
			Title:       "Invalid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-team-id")
	})

	t.Run("Should return error for invalid board type", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      model.NewId(),
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        "invalid-type",
			MinimumRole: BoardRoleViewer,
			Title:       "Invalid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-board-type")
	})

	t.Run("Should return error for invalid minimum role", func(t *testing.T) {
		board := &Board{
			ID:          model.NewId(),
			TeamID:      model.NewId(),
			CreatedBy:   model.NewId(),
			ModifiedBy:  model.NewId(),
			Type:        BoardTypeOpen,
			MinimumRole: "invalid-role",
			Title:       "Invalid Board",
			CreateAt:    1234567890,
			UpdateAt:    1234567890,
		}
		err := board.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-board-minimum-role")
	})
}

func TestBoardPatchIsValid(t *testing.T) {
	t.Run("Should return nil for valid board patch with channel ID", func(t *testing.T) {
		validChannelID := model.NewId()
		patch := &BoardPatch{
			ChannelID: &validChannelID,
		}
		err := patch.IsValid()
		require.NoError(t, err)
	})

	t.Run("Should return nil for board patch with empty channel ID (unlinking)", func(t *testing.T) {
		emptyChannelID := ""
		patch := &BoardPatch{
			ChannelID: &emptyChannelID,
		}
		err := patch.IsValid()
		require.NoError(t, err)
	})

	t.Run("Should return error for board patch with invalid channel ID", func(t *testing.T) {
		invalidChannelID := "invalid-channel-id"
		patch := &BoardPatch{
			ChannelID: &invalidChannelID,
		}
		err := patch.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "invalid-channel-id")
	})

	t.Run("Should return nil for board patch with nil channel ID", func(t *testing.T) {
		patch := &BoardPatch{
			ChannelID: nil,
		}
		err := patch.IsValid()
		require.NoError(t, err)
	})
}
