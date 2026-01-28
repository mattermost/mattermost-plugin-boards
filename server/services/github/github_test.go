// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package github

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockServicesAPI is a mock implementation of ServicesAPI for testing.
type mockServicesAPI struct {
	pluginHTTPFunc func(req *http.Request) *http.Response
	logger         mlog.LoggerIFace
}

func (m *mockServicesAPI) PluginHTTP(req *http.Request) *http.Response {
	if m.pluginHTTPFunc != nil {
		return m.pluginHTTPFunc(req)
	}
	return nil
}

func (m *mockServicesAPI) GetLogger() mlog.LoggerIFace {
	if m.logger != nil {
		return m.logger
	}
	logger, _ := mlog.NewLogger()
	return logger
}

func TestIsUserConnected(t *testing.T) {
	t.Run("user is connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, endpointConnected, req.URL.Path)
				assert.Equal(t, "user123", req.Header.Get(headerUserID))

				resp := ConnectedResponse{
					Connected:      true,
					GitHubUsername: "testuser",
				}
				body, _ := json.Marshal(resp)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		connected, err := service.IsUserConnected("user123")

		require.NoError(t, err)
		assert.True(t, connected)
	})

	t.Run("user is not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				resp := ConnectedResponse{
					Connected: false,
				}
				body, _ := json.Marshal(resp)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		connected, err := service.IsUserConnected("user123")

		require.NoError(t, err)
		assert.False(t, connected)
	})

	t.Run("error response", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				errResp := ErrorResponse{
					Error:   "not_connected",
					Message: "User is not connected to GitHub",
				}
				body, _ := json.Marshal(errResp)

				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		connected, err := service.IsUserConnected("user123")

		require.Error(t, err)
		assert.False(t, connected)
		assert.Contains(t, err.Error(), "not_connected")
	})
}

func TestGetRepositories(t *testing.T) {
	t.Run("successfully get repositories", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, "user123", req.Header.Get(headerUserID))

				repos := []Repository{
					{
						ID:       1,
						Name:     "test-repo",
						FullName: "owner/test-repo",
						Owner:    "owner",
						HTMLURL:  "https://github.com/owner/test-repo",
					},
				}
				body, _ := json.Marshal(repos)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		repos, err := service.GetRepositories("user123", "")

		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "test-repo", repos[0].Name)
		assert.Equal(t, "owner/test-repo", repos[0].FullName)
	})

	t.Run("get repositories with channel ID", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, "user123", req.Header.Get(headerUserID))
				assert.Contains(t, req.URL.String(), "channel_id=channel123")

				repos := []Repository{}
				body, _ := json.Marshal(repos)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		repos, err := service.GetRepositories("user123", "channel123")

		require.NoError(t, err)
		assert.Empty(t, repos)
	})
}

func TestCreateIssue(t *testing.T) {
	t.Run("successfully create issue", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "user123", req.Header.Get(headerUserID))
				assert.Equal(t, contentTypeJSON, req.Header.Get(headerContentType))

				var createReq CreateIssueRequest
				_ = json.NewDecoder(req.Body).Decode(&createReq)
				assert.Equal(t, "owner", createReq.Owner)
				assert.Equal(t, "repo", createReq.Repo)
				assert.Equal(t, "Test Issue", createReq.Title)

				issue := Issue{
					Number:  1,
					Title:   createReq.Title,
					Body:    createReq.Body,
					State:   "open",
					HTMLURL: "https://github.com/owner/repo/issues/1",
				}
				body, _ := json.Marshal(issue)

				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		req := CreateIssueRequest{
			Owner: "owner",
			Repo:  "repo",
			Title: "Test Issue",
			Body:  "Test body",
		}
		issue, err := service.CreateIssue("user123", req)

		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, 1, issue.Number)
		assert.Equal(t, "Test Issue", issue.Title)
		assert.Equal(t, "open", issue.State)
	})

	t.Run("error creating issue", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				errResp := ErrorResponse{
					Error:   "validation_failed",
					Message: "Title is required",
				}
				body, _ := json.Marshal(errResp)

				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		req := CreateIssueRequest{
			Owner: "owner",
			Repo:  "repo",
		}
		issue, err := service.CreateIssue("user123", req)

		require.Error(t, err)
		assert.Nil(t, issue)
		assert.Contains(t, err.Error(), "validation_failed")
	})
}

func TestGetIssue(t *testing.T) {
	t.Run("successfully get issue", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, "user123", req.Header.Get(headerUserID))
				assert.Contains(t, req.URL.Path, "owner")
				assert.Contains(t, req.URL.Path, "repo")
				assert.Contains(t, req.URL.Path, "42")

				issue := Issue{
					Number:  42,
					Title:   "Test Issue",
					Body:    "Test body",
					State:   "open",
					HTMLURL: "https://github.com/owner/repo/issues/42",
				}
				body, _ := json.Marshal(issue)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		issue, err := service.GetIssue("user123", "owner", "repo", 42)

		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, 42, issue.Number)
		assert.Equal(t, "Test Issue", issue.Title)
	})

	t.Run("issue not found", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				errResp := ErrorResponse{
					Error:   "not_found",
					Message: "Issue not found",
				}
				body, _ := json.Marshal(errResp)

				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		issue, err := service.GetIssue("user123", "owner", "repo", 999)

		require.Error(t, err)
		assert.Nil(t, issue)
		assert.Contains(t, err.Error(), "not_found")
	})
}

func TestSearchIssues(t *testing.T) {
	t.Run("successfully search issues", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, "user123", req.Header.Get(headerUserID))
				assert.Contains(t, req.URL.String(), "q=")
				assert.Contains(t, req.URL.String(), "bug")

				issues := []Issue{
					{
						Number:  1,
						Title:   "Bug in feature X",
						State:   "open",
						HTMLURL: "https://github.com/owner/repo/issues/1",
					},
					{
						Number:  2,
						Title:   "Another bug",
						State:   "closed",
						HTMLURL: "https://github.com/owner/repo/issues/2",
					},
				}
				body, _ := json.Marshal(issues)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		issues, err := service.SearchIssues("user123", "bug")

		require.NoError(t, err)
		require.Len(t, issues, 2)
		assert.Equal(t, "Bug in feature X", issues[0].Title)
		assert.Equal(t, "Another bug", issues[1].Title)
	})

	t.Run("no results found", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				issues := []Issue{}
				body, _ := json.Marshal(issues)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		issues, err := service.SearchIssues("user123", "nonexistent")

		require.NoError(t, err)
		assert.Empty(t, issues)
	})
}

func TestGetPRDetails(t *testing.T) {
	t.Run("successfully get PR details", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Equal(t, "user123", req.Header.Get(headerUserID))
				assert.Contains(t, req.URL.Path, "owner")
				assert.Contains(t, req.URL.Path, "repo")
				assert.Contains(t, req.URL.Path, "10")

				pr := PRDetails{
					Number:    10,
					Title:     "Add new feature",
					Body:      "This PR adds a new feature",
					State:     "open",
					HTMLURL:   "https://github.com/owner/repo/pull/10",
					Mergeable: true,
					Merged:    false,
					Head: PRBranch{
						Ref: "feature-branch",
						SHA: "abc123",
					},
					Base: PRBranch{
						Ref: "main",
						SHA: "def456",
					},
				}
				body, _ := json.Marshal(pr)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		pr, err := service.GetPRDetails("user123", "owner", "repo", 10)

		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, 10, pr.Number)
		assert.Equal(t, "Add new feature", pr.Title)
		assert.Equal(t, "open", pr.State)
		assert.True(t, pr.Mergeable)
		assert.False(t, pr.Merged)
		assert.Equal(t, "feature-branch", pr.Head.Ref)
		assert.Equal(t, "main", pr.Base.Ref)
	})

	t.Run("PR not found", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				errResp := ErrorResponse{
					Error:   "not_found",
					Message: "Pull request not found",
				}
				body, _ := json.Marshal(errResp)

				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			},
		}

		service := New(mockAPI)
		pr, err := service.GetPRDetails("user123", "owner", "repo", 999)

		require.Error(t, err)
		assert.Nil(t, pr)
		assert.Contains(t, err.Error(), "not_found")
	})
}

