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
	ErrNoResponse   = errors.New("no response from GitHub plugin")
	ErrPluginError  = errors.New("GitHub plugin error")
	ErrPluginStatus = errors.New("GitHub plugin returned error status")
)

const (
	// GitHubPluginID is the ID of the GitHub plugin.
	GitHubPluginID = "github"

	// API endpoints for the GitHub plugin.
	endpointConnected    = "/plugins/github/api/v1/connected"
	endpointRepos        = "/plugins/github/api/v1/repositories"
	endpointCreateIssue  = "/plugins/github/api/v1/issue"
	endpointGetIssue     = "/plugins/github/api/v1/issue/%s/%s/%d"
	endpointSearchIssues = "/plugins/github/api/v1/search/issues"
	endpointGetPR        = "/plugins/github/api/v1/pr/%s/%s/%d"
	endpointToken        = "/plugins/github/api/v1/token"

	// GitHub API endpoints (direct).
	githubAPIBase  = "https://api.github.com"
	githubAPIRefs  = "/repos/%s/%s/git/refs"
	githubAPIRef   = "/repos/%s/%s/git/refs/heads/%s"
	githubAPIRepo  = "/repos/%s/%s"

	// Headers for IPC communication.
	headerUserID      = "Mattermost-User-ID"
	headerPluginID    = "Mattermost-Plugin-ID"
	headerContentType = "Content-Type"
	headerAuthBearer  = "Authorization"
	contentTypeJSON   = "application/json"
)

// ServicesAPI defines the interface for interacting with Mattermost services.
type ServicesAPI interface {
	// PluginHTTP allows inter-plugin requests to plugin APIs.
	PluginHTTP(req *http.Request) *http.Response

	// GetLogger returns the logger instance.
	GetLogger() mlog.LoggerIFace
}

// Service provides GitHub plugin IPC functionality.
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

// IsUserConnected checks if a user has connected their GitHub account.
func (s *Service) IsUserConnected(userID string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, endpointConnected, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerUserID, userID)

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return false, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, s.handleErrorResponse(resp)
	}

	var connResp ConnectedResponse
	if err := json.NewDecoder(resp.Body).Decode(&connResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return connResp.Connected, nil
}

// GetConnectedStatus returns the full connection status including username.
func (s *Service) GetConnectedStatus(userID string) (*ConnectedResponse, error) {
	req, err := http.NewRequest(http.MethodGet, endpointConnected, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerUserID, userID)

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return nil, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var connResp ConnectedResponse
	if err := json.NewDecoder(resp.Body).Decode(&connResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &connResp, nil
}

// GetRepositories retrieves the list of repositories for a user in a channel.
func (s *Service) GetRepositories(userID, channelID string) ([]Repository, error) {
	reqURL := endpointRepos
	if channelID != "" {
		reqURL = fmt.Sprintf("%s?channel_id=%s", reqURL, url.QueryEscape(channelID))
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerUserID, userID)

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return nil, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return repos, nil
}

// CreateIssue creates a new GitHub issue.
func (s *Service) CreateIssue(userID string, req CreateIssueRequest) (*Issue, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, endpointCreateIssue, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set(headerUserID, userID)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	resp := s.api.PluginHTTP(httpReq)
	if resp == nil {
		return nil, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, s.handleErrorResponse(resp)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// GetIssue retrieves a specific GitHub issue.
func (s *Service) GetIssue(userID, owner, repo string, number int) (*Issue, error) {
	reqURL := fmt.Sprintf(endpointGetIssue, url.PathEscape(owner), url.PathEscape(repo), number)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerUserID, userID)

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return nil, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// SearchIssues searches for GitHub issues.
func (s *Service) SearchIssues(userID, term string) ([]Issue, error) {
	reqURL := fmt.Sprintf("%s?q=%s", endpointSearchIssues, url.QueryEscape(term))

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerUserID, userID)

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return nil, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return issues, nil
}

// GetPRDetails retrieves details for a specific pull request.
func (s *Service) GetPRDetails(userID, owner, repo string, number int) (*PRDetails, error) {
	reqURL := fmt.Sprintf(endpointGetPR, url.PathEscape(owner), url.PathEscape(repo), number)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerUserID, userID)

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return nil, ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var pr PRDetails
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}

// GetUserToken retrieves the OAuth token for a user from the GitHub plugin.
// This is an IPC endpoint that requires the plugin ID header.
func (s *Service) GetUserToken(userID string) (string, error) {
	reqURL := fmt.Sprintf("%s?userID=%s", endpointToken, url.QueryEscape(userID))

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// IPC endpoint requires plugin ID header instead of user ID
	req.Header.Set(headerPluginID, "focalboard")

	resp := s.api.PluginHTTP(req)
	if resp == nil {
		return "", ErrNoResponse
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", s.handleErrorResponse(resp)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResp.Token, nil
}

// CreateBranch creates a new branch in a GitHub repository.
// It uses the user's OAuth token to call GitHub API directly.
func (s *Service) CreateBranch(userID string, req CreateBranchRequest) (*Branch, error) {
	// Get user's OAuth token
	token, err := s.GetUserToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	// Get base branch SHA
	baseBranch := req.BaseBranch
	if baseBranch == "" {
		// Get default branch from repo
		defaultBranch, err := s.getDefaultBranch(token, req.Owner, req.Repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
		}
		baseBranch = defaultBranch
	}

	// Get the SHA of the base branch
	baseSHA, err := s.getBranchSHA(token, req.Owner, req.Repo, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get base branch SHA: %w", err)
	}

	// Create the new branch
	branch, err := s.createRef(token, req.Owner, req.Repo, req.BranchName, baseSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	return branch, nil
}

// getDefaultBranch retrieves the default branch name for a repository.
func (s *Service) getDefaultBranch(token, owner, repo string) (string, error) {
	reqURL := fmt.Sprintf("%s%s", githubAPIBase, fmt.Sprintf(githubAPIRepo, owner, repo))

	httpReq, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set(headerAuthBearer, "Bearer "+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	client := s.httpClient
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: status %d: %s", resp.StatusCode, string(body))
	}

	var repoInfo RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return repoInfo.DefaultBranch, nil
}

// getBranchSHA retrieves the SHA of a branch.
func (s *Service) getBranchSHA(token, owner, repo, branch string) (string, error) {
	reqURL := fmt.Sprintf("%s%s", githubAPIBase, fmt.Sprintf(githubAPIRef, owner, repo, branch))

	httpReq, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set(headerAuthBearer, "Bearer "+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	client := s.httpClient
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: status %d: %s", resp.StatusCode, string(body))
	}

	var ref Branch
	if err := json.NewDecoder(resp.Body).Decode(&ref); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ref.Object.SHA, nil
}

// createRef creates a new git reference (branch) in the repository.
func (s *Service) createRef(token, owner, repo, branchName, sha string) (*Branch, error) {
	reqURL := fmt.Sprintf("%s%s", githubAPIBase, fmt.Sprintf(githubAPIRefs, owner, repo))

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

	httpReq.Header.Set(headerAuthBearer, "Bearer "+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	client := s.httpClient
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: status %d: %s", resp.StatusCode, string(respBody))
	}

	var branch Branch
	if err := json.NewDecoder(resp.Body).Decode(&branch); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &branch, nil
}

// handleErrorResponse processes error responses from the GitHub plugin.
func (s *Service) handleErrorResponse(resp *http.Response) error {
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
