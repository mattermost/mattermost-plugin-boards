// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// Pages feature — REST API (Model Y).
//
// Pages are team-scoped. Cross-references between boards and pages are
// bidirectional via /pages/{id}/board-refs and /boards/{id}/page-refs.
//
// See docs/PAGES_PLAN.md.

func (a *API) registerPagesRoutes(r *mux.Router) {
	// Team-scoped pages
	r.HandleFunc("/teams/{teamID}/pages", a.sessionRequired(a.handleCreatePage)).Methods("POST")
	r.HandleFunc("/teams/{teamID}/pages", a.sessionRequired(a.handleListPagesForTeam)).Methods("GET")

	// Single page
	r.HandleFunc("/pages/{pageID}", a.sessionRequired(a.handleGetPage)).Methods("GET")
	r.HandleFunc("/pages/{pageID}/children", a.sessionRequired(a.handleGetChildPages)).Methods("GET")
	r.HandleFunc("/pages/{pageID}/content", a.sessionRequired(a.handleGetPageContent)).Methods("GET")
	r.HandleFunc("/pages/{pageID}/content", a.sessionRequired(a.handleSavePageContent)).Methods("PUT")
	r.HandleFunc("/pages/{pageID}/yjs-snapshot", a.sessionRequired(a.handleSaveYjsSnapshot)).Methods("PUT")

	// Cross-references
	r.HandleFunc("/boards/{boardID}/page-refs", a.sessionRequired(a.handleGetBoardPageRefs)).Methods("GET")
	r.HandleFunc("/boards/{boardID}/page-refs/{pageID}", a.sessionRequired(a.handleLinkBoardToPage)).Methods("POST")
	r.HandleFunc("/boards/{boardID}/page-refs/{pageID}", a.sessionRequired(a.handleUnlinkBoardFromPage)).Methods("DELETE")
	r.HandleFunc("/pages/{pageID}/board-refs", a.sessionRequired(a.handleGetPageBoardRefs)).Methods("GET")
	r.HandleFunc("/pages/{pageID}/board-refs/{boardID}", a.sessionRequired(a.handleLinkPageToBoard)).Methods("POST")
	r.HandleFunc("/pages/{pageID}/board-refs/{boardID}", a.sessionRequired(a.handleUnlinkPageFromBoard)).Methods("DELETE")

	// Phase 2+ stubs
	r.HandleFunc("/pages/{pageID}", a.sessionRequired(a.handleUpdatePage)).Methods("PATCH")
	r.HandleFunc("/pages/{pageID}/move", a.sessionRequired(a.handleMovePage)).Methods("POST")
	r.HandleFunc("/pages/{pageID}/duplicate", a.sessionRequired(a.handleDuplicatePage)).Methods("POST")
	r.HandleFunc("/pages/{pageID}", a.sessionRequired(a.handleDeletePage)).Methods("DELETE")
	r.HandleFunc("/pages/{pageID}/channels", a.sessionRequired(a.handleGetPageChannelLinks)).Methods("GET")
	r.HandleFunc("/pages/{pageID}/channels/{channelID}", a.sessionRequired(a.handleLinkPageToChannel)).Methods("POST")
	r.HandleFunc("/pages/{pageID}/channels/{channelID}", a.sessionRequired(a.handleUnlinkPageFromChannel)).Methods("DELETE")
	r.HandleFunc("/channels/{channelID}/pages", a.sessionRequired(a.handleListPagesForChannel)).Methods("GET")
}

// ─── Vertical slice handlers ─────────────────────────────────────────

func (a *API) handleCreatePage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]

	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to create page"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	var req struct {
		ParentID string          `json:"parentId"`
		Title    string          `json:"title"`
		Icon     string          `json:"icon"`
		Cover    string          `json:"cover"`
		Content  json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	page := &model.Page{
		TeamID:   teamID,
		ParentID: req.ParentID,
		Title:    req.Title,
		Icon:     req.Icon,
		Cover:    req.Cover,
	}

	auditRec := a.makeAuditRecord(r, "createPage", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("teamID", teamID)

	created, err := a.app.CreatePage(page, []byte(req.Content), userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	auditRec.AddMeta("pageID", created.ID)

	data, err := json.Marshal(created)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleListPagesForTeam(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]

	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	pages, err := a.app.GetPagesForTeam(teamID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if pages == nil {
		pages = []*model.Page{}
	}
	data, err := json.Marshal(pages)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleGetPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	data, err := json.Marshal(page)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleGetChildPages(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	parent, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, parent.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	children, err := a.app.GetChildPages(parent.TeamID, pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if children == nil {
		children = []*model.Page{}
	}
	data, err := json.Marshal(children)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleGetPageContent(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	content, err := a.app.GetPageContent(pageID)
	if err != nil {
		if model.IsErrNotFound(err) {
			empty := map[string]interface{}{
				"pageId":          pageID,
				"tiptapJson":      json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`),
				"yjsUpdatesCount": 0,
				"lastSnapshotAt":  0,
				"updateAt":        0,
				"updateBy":        "",
			}
			data, _ := json.Marshal(empty)
			jsonBytesResponse(w, http.StatusOK, data)
			return
		}
		a.errorResponse(w, r, err)
		return
	}
	data, err := json.Marshal(content)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleSavePageContent(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	var req struct {
		TiptapJSON json.RawMessage `json:"tiptapJson"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	auditRec := a.makeAuditRecord(r, "savePageContent", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("pageID", pageID)

	if err := a.app.SavePageContent(pageID, []byte(req.TiptapJSON), userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	a.logger.Debug("SavePageContent",
		mlog.String("pageID", pageID),
		mlog.String("userID", userID),
	)
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
	auditRec.Success()
}

// handleSaveYjsSnapshot accepts {stateB64, tiptapJson} and writes both columns.
// Body is JSON: stateB64 is base64-encoded Y.Doc state; tiptapJson is the
// derived ProseMirror doc (so reading paths and Markdown export keep working).
func (a *API) handleSaveYjsSnapshot(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	var req struct {
		StateB64   string          `json:"stateB64"`
		TiptapJSON json.RawMessage `json:"tiptapJson"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}
	state, err := base64.StdEncoding.DecodeString(req.StateB64)
	if err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest("stateB64 not base64: "+err.Error()))
		return
	}

	auditRec := a.makeAuditRecord(r, "saveYjsSnapshot", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("pageID", pageID)
	auditRec.AddMeta("stateBytes", len(state))

	if err := a.app.SaveYjsSnapshot(pageID, state, []byte(req.TiptapJSON), userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	a.logger.Debug("SaveYjsSnapshot",
		mlog.String("pageID", pageID),
		mlog.String("userID", userID),
		mlog.Int("stateBytes", len(state)),
	)
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
	auditRec.Success()
}

// ─── Cross-reference handlers ────────────────────────────────────────

func (a *API) handleGetBoardPageRefs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	boardID := mux.Vars(r)["boardID"]
	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionViewBoard) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	refs, err := a.app.GetBoardPageRefs(boardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if refs == nil {
		refs = []*model.BoardPageRef{}
	}
	data, _ := json.Marshal(refs)
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleLinkBoardToPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	boardID := mux.Vars(r)["boardID"]
	pageID := mux.Vars(r)["pageID"]
	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionManageBoardCards) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Label     string `json:"label"`
		SortOrder int64  `json:"sortOrder"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	if err := a.app.LinkBoardToPage(boardID, pageID, userID, req.Label, req.SortOrder); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
}

func (a *API) handleUnlinkBoardFromPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	boardID := mux.Vars(r)["boardID"]
	pageID := mux.Vars(r)["pageID"]
	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionManageBoardCards) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	if err := a.app.UnlinkBoardFromPage(boardID, pageID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
}

func (a *API) handleGetPageBoardRefs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]
	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	refs, err := a.app.GetPageBoardRefs(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if refs == nil {
		refs = []*model.PageBoardRef{}
	}
	data, _ := json.Marshal(refs)
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleLinkPageToBoard(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]
	boardID := mux.Vars(r)["boardID"]
	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Label string `json:"label"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	if err := a.app.LinkPageToBoard(pageID, boardID, userID, req.Label); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
}

func (a *API) handleUnlinkPageFromBoard(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]
	boardID := mux.Vars(r)["boardID"]
	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	if err := a.app.UnlinkPageFromBoard(pageID, boardID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
}

// ─── Phase 2+ stub handlers ──────────────────────────────────────────

func (a *API) handleUpdatePage(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

func (a *API) handleMovePage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	var req struct {
		ParentID  string `json:"parentId"`
		SortOrder int64  `json:"sortOrder"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	auditRec := a.makeAuditRecord(r, "movePage", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("pageID", pageID)
	auditRec.AddMeta("newParentID", req.ParentID)

	if err := a.app.MovePage(pageID, req.ParentID, req.SortOrder, userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
	auditRec.Success()
}

func (a *API) handleDuplicatePage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	cascade := r.URL.Query().Get("cascade") == "true"

	body, _ := io.ReadAll(r.Body)
	var req struct {
		ParentID string `json:"parentId"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	// default: same parent as source
	if req.ParentID == "" {
		req.ParentID = page.ParentID
	}

	auditRec := a.makeAuditRecord(r, "duplicatePage", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("sourcePageID", pageID)
	auditRec.AddMeta("cascade", cascade)

	newID, err := a.app.DuplicatePage(pageID, req.ParentID, cascade, userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	auditRec.AddMeta("newPageID", newID)
	jsonBytesResponse(w, http.StatusOK, []byte(`{"id":"`+newID+`"}`))
	auditRec.Success()
}

func (a *API) handleDeletePage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	cascade := r.URL.Query().Get("cascade") == "true"

	auditRec := a.makeAuditRecord(r, "deletePage", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("pageID", pageID)
	auditRec.AddMeta("cascade", cascade)

	if err := a.app.DeletePage(pageID, cascade, userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
	auditRec.Success()
}

func (a *API) handleGetPageChannelLinks(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]
	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	links, err := a.app.GetPageChannelLinks(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if links == nil {
		links = []*model.PageChannelLink{}
	}
	data, _ := json.Marshal(links)
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleLinkPageToChannel(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]
	channelID := mux.Vars(r)["channelID"]
	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	if !a.permissions.HasPermissionToChannel(userID, channelID, mmModel.PermissionReadChannel) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to channel"))
		return
	}
	if err := a.app.LinkPageToChannel(pageID, channelID, userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
}

func (a *API) handleUnlinkPageFromChannel(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	pageID := mux.Vars(r)["pageID"]
	channelID := mux.Vars(r)["channelID"]
	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, page.TeamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	if err := a.app.UnlinkPageFromChannel(pageID, channelID, userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
}

func (a *API) handleListPagesForChannel(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID := mux.Vars(r)["channelID"]
	if !a.permissions.HasPermissionToChannel(userID, channelID, mmModel.PermissionReadChannel) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	pages, err := a.app.GetPagesForChannel(channelID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if pages == nil {
		pages = []*model.Page{}
	}
	data, _ := json.Marshal(pages)
	jsonBytesResponse(w, http.StatusOK, data)
}

func notImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"pages api: not implemented"}`))
}
