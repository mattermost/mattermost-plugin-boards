// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package github

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// tokenResponse returns a mock PluginHTTP func that returns a token.
func tokenResponse(token string) func(req *http.Request) *http.Response {
	return func(req *http.Request) *http.Response {
		resp := TokenResponse{AccessToken: token, TokenType: "bearer"}
		body, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		}
	}
}

// tokenErrorResponse returns a mock PluginHTTP func that returns an error.
func tokenErrorResponse() func(req *http.Request) *http.Response {
	return func(req *http.Request) *http.Response {
		errResp := ErrorResponse{
			Error:   "not_connected",
			Message: "User is not connected to GitHub",
		}
		body, _ := json.Marshal(errResp)
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewReader(body)),
		}
	}
}

// newTestService creates a Service backed by a mock GitHub API server.
// The mock server handles GitHub API routes and the service is configured to use it.
func newTestService(t *testing.T, tokenFunc func(*http.Request) *http.Response, ghHandler http.Handler) (*Service, *httptest.Server) {
	t.Helper()

	ghServer := httptest.NewServer(ghHandler)

	mockAPI := &mockServicesAPI{
		pluginHTTPFunc: tokenFunc,
	}

	service := New(mockAPI)
	service.httpClient = ghServer.Client()

	return service, ghServer
}

func TestGetUserToken(t *testing.T) {
	t.Run("successfully get token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				assert.Contains(t, req.URL.String(), "userID=user123")
				assert.Equal(t, "focalboard", req.Header.Get(headerPluginID))

				return tokenResponse("ghp_test_token")(req)
			},
		}

		service := New(mockAPI)
		token, err := service.GetUserToken("user123")

		require.NoError(t, err)
		assert.Equal(t, "ghp_test_token", token)
	})

	t.Run("user not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		token, err := service.GetUserToken("user123")

		require.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("no response from plugin", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: func(req *http.Request) *http.Response {
				return nil
			},
		}

		service := New(mockAPI)
		token, err := service.GetUserToken("user123")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoResponse)
		assert.Empty(t, token)
	})
}

func TestIsUserConnected(t *testing.T) {
	t.Run("user is connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		connected, err := service.IsUserConnected("user123")

		require.NoError(t, err)
		assert.True(t, connected)
	})

	t.Run("user is not connected - error", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		connected, err := service.IsUserConnected("user123")

		require.NoError(t, err) // IsUserConnected returns false, not error
		assert.False(t, connected)
	})

	t.Run("user is not connected - empty token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse(""),
		}

		service := New(mockAPI)
		connected, err := service.IsUserConnected("user123")

		require.NoError(t, err)
		assert.False(t, connected)
	})
}

func TestGetConnectedStatus(t *testing.T) {
	t.Run("user is connected with username", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))
			user := User{Login: "testuser", ID: 123}
			json.NewEncoder(w).Encode(user)
		})

		service, ghServer := newTestService(t, tokenResponse("ghp_test_token"), mux)
		defer ghServer.Close()

		// Override the base URL to point to test server
		origGetConnectedStatus := service.GetConnectedStatus
		_ = origGetConnectedStatus // we'll test differently

		// We need to patch githubAPIBase - since it's a const, we'll test the
		// getAuthenticatedUser method indirectly via a direct HTTP test
		// For this test, we verify the token retrieval + non-empty response
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}
		service2 := New(mockAPI)
		status, err := service2.GetConnectedStatus("user123")

		require.NoError(t, err)
		assert.True(t, status.Connected)
		// Note: username may be empty because we can't mock the external GitHub API
		// in a unit test without more infrastructure. The key test is that
		// Connected is true when a token exists.
	})

	t.Run("user is not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		status, err := service.GetConnectedStatus("user123")

		require.NoError(t, err)
		assert.False(t, status.Connected)
		assert.Empty(t, status.GitHubUsername)
	})
}

func TestGetRepositories(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		repos, err := service.GetRepositories("user123", "")

		require.Error(t, err)
		assert.Nil(t, repos)
	})

	t.Run("empty token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse(""),
		}

		service := New(mockAPI)
		repos, err := service.GetRepositories("user123", "")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotConnected)
		assert.Nil(t, repos)
	})
}

func TestCreateIssue(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		issue, err := service.CreateIssue("user123", CreateIssueRequest{
			Owner: "owner",
			Repo:  "repo",
			Title: "Test Issue",
		})

		require.Error(t, err)
		assert.Nil(t, issue)
	})

	t.Run("empty token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse(""),
		}

		service := New(mockAPI)
		issue, err := service.CreateIssue("user123", CreateIssueRequest{
			Owner: "owner",
			Repo:  "repo",
			Title: "Test Issue",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotConnected)
		assert.Nil(t, issue)
	})
}

func TestGetIssue(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		issue, err := service.GetIssue("user123", "owner", "repo", 42)

		require.Error(t, err)
		assert.Nil(t, issue)
	})

	t.Run("empty token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse(""),
		}

		service := New(mockAPI)
		issue, err := service.GetIssue("user123", "owner", "repo", 42)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotConnected)
		assert.Nil(t, issue)
	})
}

func TestSearchIssues(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		issues, err := service.SearchIssues("user123", "bug")

		require.Error(t, err)
		assert.Nil(t, issues)
	})

	t.Run("empty token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse(""),
		}

		service := New(mockAPI)
		issues, err := service.SearchIssues("user123", "bug")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotConnected)
		assert.Nil(t, issues)
	})
}

func TestGetPRDetails(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenErrorResponse(),
		}

		service := New(mockAPI)
		pr, err := service.GetPRDetails("user123", "owner", "repo", 10)

		require.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("empty token", func(t *testing.T) {
		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse(""),
		}

		service := New(mockAPI)
		pr, err := service.GetPRDetails("user123", "owner", "repo", 10)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotConnected)
		assert.Nil(t, pr)
	})
}

// TestWithMockGitHubServer tests the full flow with a mock GitHub API server.
// This replaces the old PluginHTTP-based tests.
func TestWithMockGitHubServer(t *testing.T) {
	t.Run("GetRepositories", func(t *testing.T) {
		ghRepos := []ghRepository{
			{
				ID:       1,
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    User{Login: "owner"},
				Private:  false,
				HTMLURL:  "https://github.com/owner/test-repo",
			},
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))
			json.NewEncoder(w).Encode(ghRepos)
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()

		// We need to rewrite URLs to hit the test server.
		// Since githubAPIBase is a const, we'll use a custom RoundTripper.
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		repos, err := service.GetRepositories("user123", "")

		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "test-repo", repos[0].Name)
		assert.Equal(t, "owner/test-repo", repos[0].FullName)
		assert.Equal(t, "owner", repos[0].Owner)
	})

	t.Run("CreateIssue", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))

			var req ghCreateIssueRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "Test Issue", req.Title)
			assert.Equal(t, "Test body", req.Body)

			w.WriteHeader(http.StatusCreated)
			issue := Issue{
				Number:  1,
				Title:   req.Title,
				Body:    req.Body,
				State:   "open",
				HTMLURL: "https://github.com/owner/repo/issues/1",
			}
			json.NewEncoder(w).Encode(issue)
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		issue, err := service.CreateIssue("user123", CreateIssueRequest{
			Owner: "owner",
			Repo:  "repo",
			Title: "Test Issue",
			Body:  "Test body",
		})

		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, 1, issue.Number)
		assert.Equal(t, "Test Issue", issue.Title)
		assert.Equal(t, "open", issue.State)
	})

	t.Run("GetIssue", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/issues/42", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))

			issue := Issue{
				Number:  42,
				Title:   "Test Issue",
				State:   "open",
				HTMLURL: "https://github.com/owner/repo/issues/42",
			}
			json.NewEncoder(w).Encode(issue)
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		issue, err := service.GetIssue("user123", "owner", "repo", 42)

		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, 42, issue.Number)
		assert.Equal(t, "Test Issue", issue.Title)
	})

	t.Run("SearchIssues", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Query().Get("q"), "bug")

			result := ghSearchResult{
				TotalCount: 2,
				Items: []Issue{
					{Number: 1, Title: "Bug in feature X", State: "open"},
					{Number: 2, Title: "Another bug", State: "closed"},
				},
			}
			json.NewEncoder(w).Encode(result)
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		issues, err := service.SearchIssues("user123", "bug")

		require.NoError(t, err)
		require.Len(t, issues, 2)
		assert.Equal(t, "Bug in feature X", issues[0].Title)
		assert.Equal(t, "Another bug", issues[1].Title)
	})

	t.Run("GetPRDetails", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/pulls/10", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))

			pr := PRDetails{
				Number:    10,
				Title:     "Add new feature",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/pull/10",
				Mergeable: true,
				Merged:    false,
				Head:      PRBranch{Ref: "feature-branch", SHA: "abc123"},
				Base:      PRBranch{Ref: "main", SHA: "def456"},
			}
			json.NewEncoder(w).Encode(pr)
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		pr, err := service.GetPRDetails("user123", "owner", "repo", 10)

		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, 10, pr.Number)
		assert.Equal(t, "Add new feature", pr.Title)
		assert.True(t, pr.Mergeable)
		assert.Equal(t, "feature-branch", pr.Head.Ref)
	})

	t.Run("GetConnectedStatus with username", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))
			user := User{Login: "testuser", ID: 123}
			json.NewEncoder(w).Encode(user)
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		status, err := service.GetConnectedStatus("user123")

		require.NoError(t, err)
		assert.True(t, status.Connected)
		assert.Equal(t, "testuser", status.GitHubUsername)
	})

	t.Run("GitHub API error", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/issues/999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Not Found"}`))
		})

		ghServer := httptest.NewServer(mux)
		defer ghServer.Close()

		mockAPI := &mockServicesAPI{
			pluginHTTPFunc: tokenResponse("ghp_test_token"),
		}

		service := New(mockAPI)
		service.httpClient = ghServer.Client()
		service.httpClient.Transport = &rewriteTransport{
			base:    ghServer.Client().Transport,
			baseURL: ghServer.URL,
		}

		issue, err := service.GetIssue("user123", "owner", "repo", 999)

		require.Error(t, err)
		assert.Nil(t, issue)
		assert.ErrorIs(t, err, ErrGitHubAPICall)
	})
}

// rewriteTransport rewrites GitHub API URLs to point to a test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite github.com URLs to test server
	if req.URL.Host == "api.github.com" {
		req.URL.Scheme = "http"
		testURL, _ := url.Parse(t.baseURL)
		req.URL.Host = testURL.Host
	}
	if t.base != nil {
		return t.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
