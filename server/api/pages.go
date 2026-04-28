// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

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

	// File attachments (image upload/download). Team-scoped permissions.
	r.HandleFunc("/teams/{teamID}/pages/{pageID}/files", a.sessionRequired(a.handleUploadPageFile)).Methods("POST")
	r.HandleFunc("/files/teams/{teamID}/pages/{pageID}/{filename}", a.attachSession(a.handleGetPageFile)).Methods("GET")

	// Page categories (per-user, per-team)
	r.HandleFunc("/teams/{teamID}/page-categories", a.sessionRequired(a.handleListPageCategories)).Methods("GET")
	r.HandleFunc("/teams/{teamID}/page-categories", a.sessionRequired(a.handleCreatePageCategory)).Methods("POST")
	r.HandleFunc("/teams/{teamID}/page-categories/{categoryID}", a.sessionRequired(a.handleUpdatePageCategory)).Methods("PUT")
	r.HandleFunc("/teams/{teamID}/page-categories/{categoryID}", a.sessionRequired(a.handleDeletePageCategory)).Methods("DELETE")
	// Page→category assignments (Slice 2)
	r.HandleFunc("/teams/{teamID}/page-category-assignments", a.sessionRequired(a.handleListPageCategoryAssignments)).Methods("GET")
	r.HandleFunc("/teams/{teamID}/page-categories/{categoryID}/pages/{pageID}", a.sessionRequired(a.handleSetPageCategory)).Methods("POST")
	r.HandleFunc("/teams/{teamID}/page-category-assignments/{pageID}", a.sessionRequired(a.handleUnsetPageCategory)).Methods("DELETE")

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

// handleUploadPageFile accepts a multipart upload and stores it under
// pageID as the rootID. Reuses app.SaveFile (which is opaque to the rootID
// being a board vs a page) but enforces team-level access since pages are
// team-scoped.
func (a *API) handleUploadPageFile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	pageID := mux.Vars(r)["pageID"]

	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if page.TeamID != teamID {
		a.errorResponse(w, r, model.NewErrPermission("page does not belong to the specified team"))
		return
	}

	if a.app.GetConfig().MaxFileSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, a.app.GetConfig().MaxFileSize)
	}

	file, handle, err := r.FormFile(UploadFormFileKey)
	if err != nil {
		if strings.HasSuffix(err.Error(), "http: request body too large") {
			a.errorResponse(w, r, model.ErrRequestEntityTooLarge)
			return
		}
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}
	defer file.Close()

	auditRec := a.makeAuditRecord(r, "uploadPageFile", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("pageID", pageID)
	auditRec.AddMeta("teamID", teamID)
	auditRec.AddMeta("filename", handle.Filename)

	fileID, err := a.app.SaveFile(file, teamID, pageID, handle.Filename, false)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("uploadPageFile",
		mlog.String("filename", handle.Filename),
		mlog.String("fileID", fileID),
	)
	data, err := json.Marshal(FileUploadResponse{FileID: fileID})
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

// handleGetPageFile streams an uploaded page file. Auth is team-scoped —
// any team member can view files attached to a page in their team.
func (a *API) handleGetPageFile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	pageID := mux.Vars(r)["pageID"]
	filename := mux.Vars(r)["filename"]

	if userID == "" {
		a.errorResponse(w, r, model.NewErrUnauthorized("access denied"))
		return
	}
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	page, err := a.app.GetPage(pageID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if page.TeamID != teamID {
		a.errorResponse(w, r, model.NewErrPermission("page does not belong to the specified team"))
		return
	}

	auditRec := a.makeAuditRecord(r, "getPageFile", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)
	auditRec.AddMeta("pageID", pageID)
	auditRec.AddMeta("teamID", teamID)
	auditRec.AddMeta("filename", filename)

	fileInfo, fileReader, err := a.app.GetFile(teamID, pageID, filename)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if fileReader == nil {
		a.errorResponse(w, r, model.NewErrNotFound("file reader is nil"))
		return
	}
	defer fileReader.Close()

	mimeType := ""
	if fileInfo != nil {
		mimeType = fileInfo.MimeType
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if _, err := io.Copy(w, fileReader); err != nil {
		a.logger.Error("handleGetPageFile copy", mlog.Err(err))
	}
	auditRec.Success()
}

// ─── Page category handlers ──────────────────────────────────────────

func (a *API) handleListPageCategories(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	cats, err := a.app.GetPageCategories(userID, teamID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	data, err := json.Marshal(cats)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
}

func (a *API) handleCreatePageCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	var req struct {
		Name      string `json:"name"`
		Collapsed bool   `json:"collapsed"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}
	cat := &model.PageCategory{
		Name:      req.Name,
		UserID:    userID,
		TeamID:    teamID,
		Collapsed: req.Collapsed,
	}

	auditRec := a.makeAuditRecord(r, "createPageCategory", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("teamID", teamID)
	auditRec.AddMeta("name", req.Name)

	created, err := a.app.CreatePageCategory(cat)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	data, err := json.Marshal(created)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.AddMeta("categoryID", created.ID)
	auditRec.Success()
}

func (a *API) handleUpdatePageCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	categoryID := mux.Vars(r)["categoryID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	var req struct {
		Name      string `json:"name"`
		SortOrder int64  `json:"sortOrder"`
		Collapsed bool   `json:"collapsed"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	auditRec := a.makeAuditRecord(r, "updatePageCategory", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("categoryID", categoryID)

	updated, err := a.app.UpdatePageCategory(&model.PageCategory{
		ID:        categoryID,
		Name:      req.Name,
		UserID:    userID,
		TeamID:    teamID,
		SortOrder: req.SortOrder,
		Collapsed: req.Collapsed,
	})
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	data, err := json.Marshal(updated)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleDeletePageCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	categoryID := mux.Vars(r)["categoryID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	auditRec := a.makeAuditRecord(r, "deletePageCategory", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("categoryID", categoryID)

	if err := a.app.DeletePageCategory(categoryID, userID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
	auditRec.Success()
}

// handleListPageCategoryAssignments returns this user's page→category map.
func (a *API) handleListPageCategoryAssignments(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	out, err := a.app.GetPageCategoryAssignments(userID, teamID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	data, err := json.Marshal(out)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, data)
}

// handleSetPageCategory assigns a page to one of the user's categories.
// Body (optional): {"sortOrder": <number>}.
func (a *API) handleSetPageCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	categoryID := mux.Vars(r)["categoryID"]
	pageID := mux.Vars(r)["pageID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		SortOrder int64 `json:"sortOrder"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	auditRec := a.makeAuditRecord(r, "setPageCategory", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("categoryID", categoryID)
	auditRec.AddMeta("pageID", pageID)

	if err := a.app.SetPageCategory(userID, pageID, categoryID, req.SortOrder); err != nil {
		a.errorResponse(w, r, err)
		return
	}
	jsonBytesResponse(w, http.StatusOK, []byte(`{"status":"ok"}`))
	auditRec.Success()
}

// handleUnsetPageCategory removes a user's category assignment for a page.
func (a *API) handleUnsetPageCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	teamID := mux.Vars(r)["teamID"]
	pageID := mux.Vars(r)["pageID"]
	if !a.permissions.HasPermissionToTeam(userID, teamID, mmModel.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied"))
		return
	}

	auditRec := a.makeAuditRecord(r, "unsetPageCategory", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("pageID", pageID)

	if err := a.app.UnsetPageCategory(userID, teamID, pageID); err != nil {
		a.errorResponse(w, r, err)
		return
	}
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
