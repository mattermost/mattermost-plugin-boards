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
