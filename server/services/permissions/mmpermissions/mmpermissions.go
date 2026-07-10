// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmpermissions

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/permissions"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

type APIInterface interface {
	HasPermissionTo(userID string, permission *mmModel.Permission) bool
	HasPermissionToTeam(userID string, teamID string, permission *mmModel.Permission) bool
	HasPermissionToChannel(userID string, channelID string, permission *mmModel.Permission) bool
}

type Service struct {
	store  permissions.Store
	api    APIInterface
	logger mlog.LoggerIFace
}

func New(store permissions.Store, api APIInterface, logger mlog.LoggerIFace) *Service {
	return &Service{
		store:  store,
		api:    api,
		logger: logger,
	}
}

func (s *Service) HasPermissionTo(userID string, permission *mmModel.Permission) bool {
	if userID == "" || permission == nil {
		return false
	}
	return s.api.HasPermissionTo(userID, permission)
}

func (s *Service) HasPermissionToTeam(userID, teamID string, permission *mmModel.Permission) bool {
	if userID == "" || teamID == "" || permission == nil {
		return false
	}
	return s.api.HasPermissionToTeam(userID, teamID, permission)
}

func (s *Service) HasPermissionToChannel(userID, channelID string, permission *mmModel.Permission) bool {
	if userID == "" || channelID == "" || permission == nil {
		return false
	}
	return s.api.HasPermissionToChannel(userID, channelID, permission)
}

// isGuest reports whether the given user must be treated as a System Guest
// for the purpose of board admin checks. It fails closed: if the user
// record cannot be resolved (transient DB error, missing row, etc.) the
// caller drops any stale SchemeAdmin so that a real demotion can't be
// bypassed by a lookup hiccup. Legitimate Team / System Admins are still
// elevated downstream via the existing PermissionManageTeam path, so they
// retain access to the operations they're actually entitled to.
func (s *Service) isGuest(userID string) bool {
	if userID == "" || userID == model.SystemUserID {
		return false
	}
	user, err := s.store.GetUserByID(userID)
	if err != nil || user == nil {
		s.logger.Error("error getting user to evaluate guest status; treating as guest",
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		return true
	}
	return user.IsGuest
}

func (s *Service) HasPermissionToBoard(userID, boardID string, permission *mmModel.Permission) bool {
	if userID == "" || boardID == "" || permission == nil {
		return false
	}

	board, err := s.store.GetBoard(boardID)
	if model.IsErrNotFound(err) {
		var boards []*model.Board
		boards, err = s.store.GetBoardHistory(boardID, model.QueryBoardHistoryOptions{Limit: 1, Descending: true})
		if err != nil {
			return false
		}
		if len(boards) == 0 {
			return false
		}
		board = boards[0]
	} else if err != nil {
		s.logger.Error("error getting board",
			mlog.String("boardID", boardID),
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		return false
	}

	// we need to check that the user has permission to see the team
	// regardless of its local permissions to the board
	if !s.HasPermissionToTeam(userID, board.TeamID, model.PermissionViewTeam) {
		return false
	}
	member, err := s.store.GetMemberForBoard(boardID, userID)
	if model.IsErrNotFound(err) {
		return false
	}
	if err != nil {
		s.logger.Error("error getting member for board",
			mlog.String("boardID", boardID),
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		return false
	}
	if member == nil {
		s.logger.Error("member is nil for board",
			mlog.String("boardID", boardID),
			mlog.String("userID", userID),
		)
		return false
	}

	switch member.MinimumRole {
	case "admin":
		member.SchemeAdmin = true
	case "editor":
		member.SchemeEditor = true
	case "commenter":
		member.SchemeCommenter = true
	case "viewer":
		member.SchemeViewer = true
	}

	// Guests must never hold admin rights on a board, regardless of any
	// stale SchemeAdmin flag persisted from before they were demoted.
	if member.SchemeAdmin && s.isGuest(userID) {
		member.SchemeAdmin = false
	}

	// Admins become member of boards, but get minimal role
	// if they are a System/Team Admin (model.PermissionManageTeam)
	// elevate their permissions
	if !member.SchemeAdmin && s.HasPermissionToTeam(userID, board.TeamID, model.PermissionManageTeam) {
		return true
	}

	switch permission {
	case model.PermissionManageBoardType, model.PermissionDeleteBoard, model.PermissionManageBoardRoles, model.PermissionShareBoard, model.PermissionDeleteOthersComments:
		return member.SchemeAdmin
	case model.PermissionManageBoardCards, model.PermissionManageBoardProperties:
		return member.SchemeAdmin || member.SchemeEditor
	case model.PermissionCommentBoardCards:
		return member.SchemeAdmin || member.SchemeEditor || member.SchemeCommenter
	case model.PermissionViewBoard:
		return member.SchemeAdmin || member.SchemeEditor || member.SchemeCommenter || member.SchemeViewer
	default:
		return false
	}
}
