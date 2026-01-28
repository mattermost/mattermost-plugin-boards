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
}

