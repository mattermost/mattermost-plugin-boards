// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"
	"github.com/mattermost/mattermost-plugin-boards/server/services/github"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	MaxGitHubRequestSize = 1024 * 1024 // 1MB limit for JSON request body
)

func (a *API) registerGitHubRoutes(r *mux.Router) {
	// GitHub integration APIs
	r.HandleFunc("/github/connected", a.sessionRequired(a.handleGetGitHubConnected)).Methods("GET")
	r.HandleFunc("/github/repositories", a.sessionRequired(a.handleGetGitHubRepositories)).Methods("GET")
	r.HandleFunc("/github/issues", a.sessionRequired(a.handleCreateGitHubIssue)).Methods("POST")
	r.HandleFunc("/github/issues", a.sessionRequired(a.handleSearchGitHubIssues)).Methods("GET")
	r.HandleFunc("/github/branches", a.sessionRequired(a.handleCreateGitHubBranch)).Methods("POST")
}

func (a *API) handleGetGitHubConnected(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /github/connected getGitHubConnected
	//
	// Check if the user has connected their GitHub account
	//
	// ---
	// produces:
	// - application/json
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//       type: object
	//       properties:
	//         connected:
	//           type: boolean
	//         github_username:
	//           type: string
	//   '500':
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	userID := getUserID(r)

	auditRec := a.makeAuditRecord(r, "getGitHubConnected", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)

	githubService := a.app.GetGitHubService()
	if githubService == nil {
		a.errorResponse(w, r, model.NewErrNotImplemented("GitHub service not available"))
		return
	}

	response, err := githubService.GetConnectedStatus(userID)
	if err != nil {
		a.logger.Error("Failed to check GitHub connection",
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		a.errorResponse(w, r, err)
		return
	}

	data, err := json.Marshal(response)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleGetGitHubRepositories(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /github/repositories getGitHubRepositories
	//
	// Get the list of GitHub repositories for the user
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: channel_id
	//   in: query
	//   description: Optional channel ID to filter repositories
	//   required: false
	//   type: string
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//       type: array
	//       items:
	//         type: object
	//   '500':
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	userID := getUserID(r)
	query := r.URL.Query()
	channelID := query.Get("channel_id")

	auditRec := a.makeAuditRecord(r, "getGitHubRepositories", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)

	githubService := a.app.GetGitHubService()
	if githubService == nil {
		a.errorResponse(w, r, model.NewErrNotImplemented("GitHub service not available"))
		return
	}

	repos, err := githubService.GetRepositories(userID, channelID)
	if err != nil {
		a.logger.Error("Failed to get GitHub repositories",
			mlog.String("userID", userID),
			mlog.String("channelID", channelID),
			mlog.Err(err),
		)
		a.errorResponse(w, r, err)
		return
	}

	data, err := json.Marshal(repos)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleCreateGitHubIssue(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /github/issues createGitHubIssue
	//
	// Create a new GitHub issue
	//
	// ---
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - owner
	//       - repo
	//       - title
	//     properties:
	//       owner:
	//         type: string
	//       repo:
	//         type: string
	//       title:
	//         type: string
	//       body:
	//         type: string
	//       labels:
	//         type: array
	//         items:
	//           type: string
	//       assignees:
	//         type: array
	//         items:
	//           type: string
	//       milestone:
	//         type: integer
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//       type: object
	//   '400':
	//     description: bad request
	//   '500':
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	// Limit request body size to prevent memory pressure
	r.Body = http.MaxBytesReader(w, r.Body, MaxGitHubRequestSize)

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.HasSuffix(err.Error(), "http: request body too large") {
			a.errorResponse(w, r, model.ErrRequestEntityTooLarge)
			return
		}
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	var req github.CreateIssueRequest
	if unmarshalErr := json.Unmarshal(requestBody, &req); unmarshalErr != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(unmarshalErr.Error()))
		return
	}

	if req.Owner == "" || req.Repo == "" || req.Title == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("owner, repo, and title are required"))
		return
	}

	userID := getUserID(r)

	auditRec := a.makeAuditRecord(r, "createGitHubIssue", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("owner", req.Owner)
	auditRec.AddMeta("repo", req.Repo)
	auditRec.AddMeta("title", req.Title)

	githubService := a.app.GetGitHubService()
	if githubService == nil {
		a.errorResponse(w, r, model.NewErrNotImplemented("GitHub service not available"))
		return
	}

	issue, err := githubService.CreateIssue(userID, req)
	if err != nil {
		a.logger.Error("Failed to create GitHub issue",
			mlog.String("userID", userID),
			mlog.String("owner", req.Owner),
			mlog.String("repo", req.Repo),
			mlog.Err(err),
		)
		a.errorResponse(w, r, err)
		return
	}

	data, err := json.Marshal(issue)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.AddMeta("issueNumber", issue.Number)
	auditRec.Success()
}

func (a *API) handleSearchGitHubIssues(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /github/issues searchGitHubIssues
	//
	// Search for GitHub issues
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: q
	//   in: query
	//   description: Search query (GitHub search syntax)
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
	//         type: object
	//   '400':
	//     description: bad request
	//   '500':
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	query := r.URL.Query()
	searchTerm := query.Get("q")

	if searchTerm == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("search query parameter 'q' is required"))
		return
	}

	userID := getUserID(r)

	auditRec := a.makeAuditRecord(r, "searchGitHubIssues", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)
	auditRec.AddMeta("searchTerm", searchTerm)

	githubService := a.app.GetGitHubService()
	if githubService == nil {
		a.errorResponse(w, r, model.NewErrNotImplemented("GitHub service not available"))
		return
	}

	issues, err := githubService.SearchIssues(userID, searchTerm)
	if err != nil {
		a.logger.Error("Failed to search GitHub issues",
			mlog.String("userID", userID),
			mlog.String("searchTerm", searchTerm),
			mlog.Err(err),
		)
		a.errorResponse(w, r, err)
		return
	}

	data, err := json.Marshal(issues)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
	auditRec.Success()
}

func (a *API) handleCreateGitHubBranch(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /github/branches createGitHubBranch
	//
	// Create a new GitHub branch
	//
	// ---
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - owner
	//       - repo
	//       - branch_name
	//     properties:
	//       owner:
	//         type: string
	//       repo:
	//         type: string
	//       branch_name:
	//         type: string
	//       base_branch:
	//         type: string
	//         description: Optional base branch (defaults to repo's default branch)
	// security:
	// - BearerAuth: []
	// responses:
	//   '201':
	//     description: branch created
	//     schema:
	//       type: object
	//       properties:
	//         ref:
	//           type: string
	//         url:
	//           type: string
	//   '400':
	//     description: bad request
	//   '500':
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxGitHubRequestSize)

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.HasSuffix(err.Error(), "http: request body too large") {
			a.errorResponse(w, r, model.ErrRequestEntityTooLarge)
			return
		}
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	var req github.CreateBranchRequest
	if unmarshalErr := json.Unmarshal(requestBody, &req); unmarshalErr != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(unmarshalErr.Error()))
		return
	}

	if req.Owner == "" || req.Repo == "" || req.BranchName == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("owner, repo, and branch_name are required"))
		return
	}

	userID := getUserID(r)

	auditRec := a.makeAuditRecord(r, "createGitHubBranch", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("owner", req.Owner)
	auditRec.AddMeta("repo", req.Repo)
	auditRec.AddMeta("branchName", req.BranchName)

	githubService := a.app.GetGitHubService()
	if githubService == nil {
		a.errorResponse(w, r, model.NewErrNotImplemented("GitHub service not available"))
		return
	}

	branch, err := githubService.CreateBranch(userID, req)
	if err != nil {
		a.logger.Error("Failed to create GitHub branch",
			mlog.String("userID", userID),
			mlog.String("owner", req.Owner),
			mlog.String("repo", req.Repo),
			mlog.String("branchName", req.BranchName),
			mlog.Err(err),
		)
		a.errorResponse(w, r, err)
		return
	}

	data, err := json.Marshal(branch)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusCreated, data)
	auditRec.AddMeta("branchRef", branch.Ref)
	auditRec.Success()
}
