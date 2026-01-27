// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (a *API) registerStatusTransitionRulesRoutes(r *mux.Router) {
	// Status Transition Rules APIs
	r.HandleFunc("/boards/{boardID}/status-transition-rules", a.sessionRequired(a.handleGetStatusTransitionRules)).Methods("GET")
	r.HandleFunc("/boards/{boardID}/status-transition-rules", a.sessionRequired(a.handleSaveStatusTransitionRules)).Methods("POST")
}

func (a *API) handleGetStatusTransitionRules(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /boards/{boardID}/status-transition-rules getStatusTransitionRules
	//
	// Get status transition rules for a board
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: boardID
	//   in: path
	//   description: Board ID
	//   required: true
	//   type: string
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/StatusTransitionRule"
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	boardID := mux.Vars(r)["boardID"]
	userID := getUserID(r)

	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionViewBoard) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to view board"))
		return
	}

	auditRec := a.makeAuditRecord(r, "getStatusTransitionRules", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)
	auditRec.AddMeta("boardID", boardID)

	rules, err := a.app.GetStatusTransitionRules(boardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("GetStatusTransitionRules",
		mlog.String("boardID", boardID),
		mlog.Int("rulesCount", len(rules)),
	)

	data, err := json.Marshal(rules)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleSaveStatusTransitionRules(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /boards/{boardID}/status-transition-rules saveStatusTransitionRules
	//
	// Save status transition rules for a board
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: boardID
	//   in: path
	//   description: Board ID
	//   required: true
	//   type: string
	// - name: Body
	//   in: body
	//   description: Array of status transition rules
	//   required: true
	//   schema:
	//     type: array
	//     items:
	//       "$ref": "#/definitions/StatusTransitionRule"
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	boardID := mux.Vars(r)["boardID"]
	userID := getUserID(r)

	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionManageBoardProperties) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to manage board properties"))
		return
	}

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	rules, err := model.StatusTransitionRulesFromJSON(bytes.NewReader(requestBody))
	if err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest("invalid request body: "+err.Error()))
		return
	}

	// Validate that all rules are for the correct board
	for _, rule := range rules {
		if rule.BoardID != boardID {
			a.errorResponse(w, r, model.NewErrBadRequest("rule board ID does not match URL board ID"))
			return
		}
	}

	auditRec := a.makeAuditRecord(r, "saveStatusTransitionRules", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("boardID", boardID)
	auditRec.AddMeta("rulesCount", len(rules))

	// Delete existing rules for the board before saving new ones
	err = a.app.DeleteStatusTransitionRulesForBoard(boardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	// Save the new rules
	err = a.app.SaveStatusTransitionRules(rules)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("SaveStatusTransitionRules",
		mlog.String("boardID", boardID),
		mlog.Int("rulesCount", len(rules)),
	)

	jsonStringResponse(w, http.StatusOK, "{}")
	auditRec.Success()
}

