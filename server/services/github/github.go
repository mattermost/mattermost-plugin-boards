// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// Static errors for GitHub service.
var (
	ErrNoResponse    = errors.New("no response from GitHub plugin")
	ErrPluginError   = errors.New("GitHub plugin error")
	ErrPluginStatus  = errors.New("GitHub plugin returned error status")
	ErrGitHubAPICall = errors.New("GitHub API error")
	ErrNotConnected  = errors.New("user is not connected to GitHub")
)

const (
	// GitHubPluginID is the ID of the GitHub plugin.
	GitHubPluginID = "github"

	// IPC endpoint for token retrieval (uses query params, not User-ID header).
	// This is the only PluginHTTP endpoint we use, because other endpoints rely on
	// Mattermost-User-ID header which gets overwritten by PluginHTTP.
	endpointToken = "/github/api/v1/token" //nolint:gosec // G101: This is an endpoint path, not a credential

	// GitHub API base URL.
	githubAPIBase = "https://api.github.com"

	// GitHub API endpoint patterns.
	githubAPIUser         = "/user"
	githubAPIUserRepos    = "/user/repos"
	githubAPIRepoIssues   = "/repos/%s/%s/issues"
	githubAPIRepoIssue    = "/repos/%s/%s/issues/%d"
	githubAPISearchIssues = "/search/issues"
	githubAPIRepoPull     = "/repos/%s/%s/pulls/%d"
	githubAPIRefs         = "/repos/%s/%s/git/refs"
	githubAPIRef          = "/repos/%s/%s/git/refs/heads/%s"
	githubAPIRepo         = "/repos/%s/%s"

	// Headers.
	headerPluginID    = "Mattermost-Plugin-ID"
	headerContentType = "Content-Type"
	headerAuthBearer  = "Authorization"
	headerAccept      = "Accept"
	contentTypeJSON   = "application/json"
	githubAccept      = "application/vnd.github+json"
)

// ServicesAPI defines the interface for interacting with Mattermost services.
type ServicesAPI interface {
	// PluginHTTP allows inter-plugin requests to plugin APIs.
	PluginHTTP(req *http.Request) *http.Response

	// GetLogger returns the logger instance.
	GetLogger() mlog.LoggerIFace
}

// Service provides GitHub integration functionality.
// It retrieves the user's OAuth token via IPC (PluginHTTP → /api/v1/token)
// and then calls the GitHub API directly, bypassing the Mattermost-User-ID
// header issue in PluginHTTP.
type Service struct {
	api        ServicesAPI
	httpClient *http.Client
}

// New creates a new GitHub service instance.
func New(api ServicesAPI) *Service {
	return &Service{
		api: api,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetUserToken retrieves the OAuth token for a user from the GitHub plugin.
// This is the only IPC call we make — it uses query params (not the User-ID
// header that gets overwritten by PluginHTTP).
func (s *Service) GetUserToken(userID string) (string, error) {
	reqURL := fmt.Sprintf("%s?userID=%s", endpointToken, url.QueryEscape(userID))

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerPluginID, "focalboard")

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return "", ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", s.handleIPCError(resp)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// IsUserConnected checks if a user has connected their GitHub account.
// Returns true if the user has a valid OAuth token.
func (s *Service) IsUserConnected(userID string) (bool, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		// If plugin returns an error, user is likely not connected
		s.api.GetLogger().Debug("GitHub IsUserConnected: token retrieval failed",
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		return false, nil
	}

	return token != "", nil
}

// GetConnectedStatus returns the full connection status including GitHub username.
func (s *Service) GetConnectedStatus(userID string) (*ConnectedResponse, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		s.api.GetLogger().Debug("GitHub GetConnectedStatus: token retrieval failed",
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		return &ConnectedResponse{Connected: false}, nil
	}

	if token == "" {
		return &ConnectedResponse{Connected: false}, nil
	}

	// Get the GitHub username via the API
	username, err := s.getAuthenticatedUser(token)
	if err != nil {
		s.api.GetLogger().Warn("GitHub GetConnectedStatus: failed to get username",
			mlog.String("userID", userID),
			mlog.Err(err),
		)
		// Token exists but we can't get the username — still connected
		return &ConnectedResponse{Connected: true}, nil
	}

	return &ConnectedResponse{
		Connected:      true,
		GitHubUsername: username,
	}, nil
}

// GetRepositories retrieves the list of GitHub repositories for a user.
func (s *Service) GetRepositories(userID, channelID string) ([]Repository, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}
	if token == "" {
		return nil, ErrNotConnected
	}

	reqURL := fmt.Sprintf("%s%s?per_page=100&sort=full_name", githubAPIBase, githubAPIUserRepos)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleGitHubError(resp)
	}

	var ghRepos []ghRepository
	if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	repos := make([]Repository, 0, len(ghRepos))
	for _, r := range ghRepos {
		repos = append(repos, Repository{
			ID:            r.ID,
			Name:          r.Name,
			FullName:      r.FullName,
			Owner:         r.Owner.Login,
			Private:       r.Private,
			HTMLURL:       r.HTMLURL,
			Description:   r.Description,
			DefaultBranch: r.DefaultBranch,
		})
	}

	return repos, nil
}

// CreateIssue creates a new GitHub issue.
func (s *Service) CreateIssue(userID string, req CreateIssueRequest) (*Issue, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}
	if token == "" {
		return nil, ErrNotConnected
	}

	reqURL := fmt.Sprintf("%s"+githubAPIRepoIssues, githubAPIBase, req.Owner, req.Repo)

	payload := ghCreateIssueRequest{
		Title:     req.Title,
		Body:      req.Body,
		Labels:    req.Labels,
		Assignees: req.Assignees,
	}
	if req.Milestone > 0 {
		payload.Milestone = &req.Milestone
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(httpReq, token)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, s.handleGitHubError(resp)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// GetIssue retrieves a specific GitHub issue.
func (s *Service) GetIssue(userID, owner, repo string, number int) (*Issue, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}
	if token == "" {
		return nil, ErrNotConnected
	}

	reqURL := fmt.Sprintf("%s"+githubAPIRepoIssue, githubAPIBase, owner, repo, number)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleGitHubError(resp)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// SearchIssues searches for GitHub issues using the search API.
func (s *Service) SearchIssues(userID, term string) ([]Issue, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}
	if token == "" {
		return nil, ErrNotConnected
	}

	reqURL := fmt.Sprintf("%s%s?q=%s", githubAPIBase, githubAPISearchIssues, url.QueryEscape(term))

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleGitHubError(resp)
	}

	var searchResult ghSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return searchResult.Items, nil
}

// GetPRDetails retrieves details for a specific pull request.
func (s *Service) GetPRDetails(userID, owner, repo string, number int) (*PRDetails, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}
	if token == "" {
		return nil, ErrNotConnected
	}

	reqURL := fmt.Sprintf("%s"+githubAPIRepoPull, githubAPIBase, owner, repo, number)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleGitHubError(resp)
	}

	var pr PRDetails
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}

// CreateBranch creates a new branch in a GitHub repository.
func (s *Service) CreateBranch(userID string, req CreateBranchRequest) (*Branch, error) {
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}
	if token == "" {
		return nil, ErrNotConnected
	}

	// Get base branch SHA
	baseBranch := req.BaseBranch
	if baseBranch == "" {
		defaultBranch, dbErr := s.getDefaultBranch(token, req.Owner, req.Repo)
		if dbErr != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", dbErr)
		}
		baseBranch = defaultBranch
	}

	baseSHA, err := s.getBranchSHA(token, req.Owner, req.Repo, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get base branch SHA: %w", err)
	}

	branch, err := s.createRef(token, req.Owner, req.Repo, req.BranchName, baseSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	return branch, nil
}

// --- Internal helpers ---

// setGitHubHeaders sets common headers for GitHub API requests.
func (s *Service) setGitHubHeaders(req *http.Request, token string) {
	req.Header.Set(headerAuthBearer, "Bearer "+token)
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(headerAccept, githubAccept)
}

// getAuthenticatedUser returns the login name of the authenticated user.
func (s *Service) getAuthenticatedUser(token string) (string, error) {
	reqURL := githubAPIBase + githubAPIUser

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", s.handleGitHubError(resp)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return user.Login, nil
}

// getDefaultBranch retrieves the default branch name for a repository.
func (s *Service) getDefaultBranch(token, owner, repo string) (string, error) {
	reqURL := fmt.Sprintf("%s"+githubAPIRepo, githubAPIBase, owner, repo)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", s.handleGitHubError(resp)
	}

	var repoInfo RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return repoInfo.DefaultBranch, nil
}

// getBranchSHA retrieves the SHA of a branch.
func (s *Service) getBranchSHA(token, owner, repo, branch string) (string, error) {
	reqURL := fmt.Sprintf("%s"+githubAPIRef, githubAPIBase, owner, repo, branch)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(req, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", s.handleGitHubError(resp)
	}

	var ref Branch
	if err := json.NewDecoder(resp.Body).Decode(&ref); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ref.Object.SHA, nil
}

// createRef creates a new git reference (branch) in the repository.
func (s *Service) createRef(token, owner, repo, branchName, sha string) (*Branch, error) {
	reqURL := fmt.Sprintf("%s"+githubAPIRefs, githubAPIBase, owner, repo)

	payload := map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": sha,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setGitHubHeaders(httpReq, token)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, s.handleGitHubError(resp)
	}

	var branch Branch
	if err := json.NewDecoder(resp.Body).Decode(&branch); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &branch, nil
}

// handleIPCError processes error responses from the GitHub plugin IPC.
func (s *Service) handleIPCError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: status %d (failed to read body: %w)", ErrPluginStatus, resp.StatusCode, err)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("%w: status %d: %s", ErrPluginStatus, resp.StatusCode, string(body))
	}

	if errResp.Message != "" {
		return fmt.Errorf("%w: %s - %s", ErrPluginError, errResp.Error, errResp.Message)
	}

	return fmt.Errorf("%w: %s", ErrPluginError, errResp.Error)
}

// handleGitHubError processes error responses from the GitHub API.
func (s *Service) handleGitHubError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%w: status %d: %s", ErrGitHubAPICall, resp.StatusCode, string(body))
}
