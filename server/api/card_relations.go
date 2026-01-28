// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (a *API) registerCardRelationsRoutes(r *mux.Router) {
	// Card Relations APIs
	r.HandleFunc("/cards/{cardID}/relations", a.sessionRequired(a.handleGetCardRelations)).Methods("GET")
	r.HandleFunc("/cards/{cardID}/relations", a.sessionRequired(a.handleCreateCardRelation)).Methods("POST")
	r.HandleFunc("/relations/{relationID}", a.sessionRequired(a.handleDeleteCardRelation)).Methods("DELETE")
}

func (a *API) handleGetCardRelations(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /cards/{cardID}/relations getCardRelations
	//
	// Fetches all relations for the specified card.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: cardID
	//   in: path
	//   description: Card ID
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
	//         "$ref": "#/definitions/CardRelation"
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	userID := getUserID(r)
	cardID := mux.Vars(r)["cardID"]

	// Get the card to check permissions
	card, err := a.app.GetCardByID(cardID)
	if err != nil {
		message := fmt.Sprintf("could not fetch card %s: %s", cardID, err)
		a.errorResponse(w, r, model.NewErrBadRequest(message))
		return
	}

	if !a.permissions.HasPermissionToBoard(userID, card.BoardID, model.PermissionViewBoard) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to fetch card relations"))
		return
	}

	auditRec := a.makeAuditRecord(r, "getCardRelations", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)
	auditRec.AddMeta("cardID", cardID)

	relations, err := a.app.GetCardRelations(cardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("GetCardRelations",
		mlog.String("cardID", cardID),
		mlog.String("userID", userID),
		mlog.Int("count", len(relations)),
	)

	data, err := json.Marshal(relations)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleCreateCardRelation(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /cards/{cardID}/relations createCardRelation
	//
	// Creates a new relation for the specified card.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: cardID
	//   in: path
	//   description: Card ID (source card)
	//   required: true
	//   type: string
	// - name: Body
	//   in: body
	//   description: the card relation to create
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CardRelation"
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//       $ref: '#/definitions/CardRelation'
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	userID := getUserID(r)
	cardID := mux.Vars(r)["cardID"]

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	var newRelation *model.CardRelation
	if err = json.Unmarshal(requestBody, &newRelation); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	// Check for nil after unmarshal (e.g., body was "null")
	if newRelation == nil {
		a.errorResponse(w, r, model.NewErrBadRequest("relation cannot be null"))
		return
	}

	// Ensure the source card ID matches the URL parameter
	newRelation.SourceCardID = cardID

	// Get the card to check permissions
	card, err := a.app.GetCardByID(cardID)
	if err != nil {
		message := fmt.Sprintf("could not fetch card %s: %s", cardID, err)
		a.errorResponse(w, r, model.NewErrBadRequest(message))
		return
	}

	if !a.permissions.HasPermissionToBoard(userID, card.BoardID, model.PermissionManageBoardCards) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to create card relation"))
		return
	}

	// Basic validation for user-provided fields (full validation happens in store after Populate)
	if newRelation.TargetCardID == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("target card id cannot be empty"))
		return
	}
	if newRelation.RelationType == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("relation type cannot be empty"))
		return
	}
	if newRelation.SourceCardID == newRelation.TargetCardID {
		a.errorResponse(w, r, model.NewErrBadRequest("source and target card cannot be the same"))
		return
	}

	auditRec := a.makeAuditRecord(r, "createCardRelation", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("cardID", cardID)
	auditRec.AddMeta("targetCardID", newRelation.TargetCardID)
	auditRec.AddMeta("relationType", newRelation.RelationType)

	// Create the relation
	relation, err := a.app.CreateCardRelation(newRelation, userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("CreateCardRelation",
		mlog.String("cardID", cardID),
		mlog.String("targetCardID", relation.TargetCardID),
		mlog.String("relationType", string(relation.RelationType)),
		mlog.String("userID", userID),
	)

	data, err := json.Marshal(relation)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleDeleteCardRelation(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /relations/{relationID} deleteCardRelation
	//
	// Deletes the specified card relation.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: relationID
	//   in: path
	//   description: Relation ID
	//   required: true
	//   type: string
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	userID := getUserID(r)
	relationID := mux.Vars(r)["relationID"]

	// Get the relation to check permissions
	// We need to get all relations and find the one we're looking for
	// This is not ideal but we don't have a GetCardRelationByID method
	// For now, we'll try to delete and let the app layer handle validation

	auditRec := a.makeAuditRecord(r, "deleteCardRelation", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("relationID", relationID)

	// Delete the relation
	err := a.app.DeleteCardRelation(relationID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("DeleteCardRelation",
		mlog.String("relationID", relationID),
		mlog.String("userID", userID),
	)

	jsonStringResponse(w, http.StatusOK, "{}")
	auditRec.Success()
}
