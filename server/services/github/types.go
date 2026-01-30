// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package github

import "time"

// Repository represents a GitHub repository.
type Repository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
}

// Issue represents a GitHub issue.
type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	User      User       `json:"user"`
	Labels    []Label    `json:"labels"`
	Assignees []User     `json:"assignees"`
	Milestone *Milestone `json:"milestone,omitempty"`
}

// User represents a GitHub user.
type User struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// Label represents a GitHub label.
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// Milestone represents a GitHub milestone.
type Milestone struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// PRDetails represents GitHub pull request details.
type PRDetails struct {
	Number             int        `json:"number"`
	Title              string     `json:"title"`
	Body               string     `json:"body"`
	State              string     `json:"state"`
	HTMLURL            string     `json:"html_url"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	MergedAt           *time.Time `json:"merged_at,omitempty"`
	User               User       `json:"user"`
	Head               PRBranch   `json:"head"`
	Base               PRBranch   `json:"base"`
	Mergeable          bool       `json:"mergeable"`
	Merged             bool       `json:"merged"`
	Labels             []Label    `json:"labels"`
	Assignees          []User     `json:"assignees"`
	RequestedReviewers []User     `json:"requested_reviewers"`
}

// PRBranch represents a branch in a pull request.
type PRBranch struct {
	Ref  string     `json:"ref"`
	SHA  string     `json:"sha"`
	Repo Repository `json:"repo"`
}

// CreateIssueRequest represents a request to create a GitHub issue.
type CreateIssueRequest struct {
	Owner     string   `json:"owner"`
	Repo      string   `json:"repo"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Milestone int      `json:"milestone,omitempty"`
}

// ConnectedResponse represents the response from the connected endpoint.
type ConnectedResponse struct {
	Connected      bool   `json:"connected"`
	GitHubUsername string `json:"github_username,omitempty"`
}

// ErrorResponse represents an error response from the GitHub plugin.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// CreateBranchRequest represents a request to create a new GitHub branch.
type CreateBranchRequest struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BranchName string `json:"branch_name"`
	BaseBranch string `json:"base_branch,omitempty"` // defaults to repo's default branch
}

// Branch represents a GitHub branch reference.
type Branch struct {
	Ref    string `json:"ref"`
	URL    string `json:"url"`
	Object struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	} `json:"object"`
}

// TokenResponse represents the OAuth token response from the GitHub plugin's
// /api/v1/token endpoint. The plugin returns an oauth2.Token struct.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Expiry       string `json:"expiry,omitempty"`
}

// RepoInfo represents minimal repository info for default branch lookup.
type RepoInfo struct {
	DefaultBranch string `json:"default_branch"`
}

// BranchInfo represents a simplified branch information.
type BranchInfo struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

// --- Internal types for GitHub API responses ---

// ghRepository is the raw GitHub API repository response.
type ghRepository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         User   `json:"owner"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
}

// ghCreateIssueRequest is the GitHub API request body for creating an issue.
type ghCreateIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Milestone *int     `json:"milestone,omitempty"`
}

// ghSearchResult is the GitHub API search response.
type ghSearchResult struct {
	TotalCount int     `json:"total_count"`
	Items      []Issue `json:"items"`
}
