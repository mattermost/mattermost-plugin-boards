// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-boards/server/app"
	"github.com/mattermost/mattermost-plugin-boards/server/auth"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/metrics"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/mockstore"
	"github.com/mattermost/mattermost-plugin-boards/server/services/webhook"
	"github.com/mattermost/mattermost-plugin-boards/server/ws"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore/mocks"
)

type allowAllPermissions struct{}

func (allowAllPermissions) HasPermissionTo(string, *mmModel.Permission) bool { return true }
func (allowAllPermissions) HasPermissionToTeam(string, string, *mmModel.Permission) bool {
	return true
}
func (allowAllPermissions) HasPermissionToChannel(string, string, *mmModel.Permission) bool {
	return true
}
func (allowAllPermissions) HasPermissionToBoard(string, string, *mmModel.Permission) bool {
	return true
}

func TestHandleUploadFile(t *testing.T) {
	const (
		teamID  = "abcdefghijklmnopqrstuvwxyz"
		boardID = "bfoi6yy6pa3yzika53spj7pq9ee"
	)

	setup := func(t *testing.T) (*API, *mockstore.MockStore, *mocks.FileBackend) {
		t.Helper()
		ctrl := gomock.NewController(t)
		cfg := config.Configuration{}
		store := mockstore.NewMockStore(ctrl)
		filesBackend := &mocks.FileBackend{}
		authService := auth.New(&cfg, store, nil)
		logger := mlog.CreateConsoleTestLogger(t)

		testApp := app.New(&cfg, ws.NewServer(authService, logger, store), app.Services{
			Auth:             authService,
			Store:            store,
			FilesBackend:     filesBackend,
			Webhook:          webhook.NewClient(&cfg, logger),
			Metrics:          metrics.NewMetrics(metrics.InstanceInfo{}),
			Logger:           logger,
			SkipTemplateInit: true,
			Permissions:      allowAllPermissions{},
		})
		t.Cleanup(testApp.Shutdown)

		auditService, err := audit.NewAudit()
		require.NoError(t, err)
		t.Cleanup(func() { _ = auditService.Shutdown() })

		store.EXPECT().GetBoard(boardID).Return(&model.Board{ID: boardID, TeamID: teamID}, nil)
		return NewAPI(testApp, "", "", allowAllPermissions{}, logger, auditService), store, filesBackend
	}

	uploadRequest := func(t *testing.T, content []byte) *http.Request {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile(UploadFormFileKey, "file.txt")
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID+"/"+boardID+"/files", &body)
		req = mux.SetURLVars(req, map[string]string{"teamID": teamID, "boardID": boardID})
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req.WithContext(context.WithValue(req.Context(), sessionContextKey, &model.Session{UserID: "user"}))
	}

	t.Run("rejects an empty file without writing it", func(t *testing.T) {
		api, _, filesBackend := setup(t)

		w := httptest.NewRecorder()
		api.handleUploadFile(w, uploadRequest(t, []byte{}))

		res := w.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Contains(t, string(b), "file is empty")
		filesBackend.AssertNotCalled(t, "WriteFile", mock.Anything, mock.Anything)
	})

	t.Run("saves a non-empty file", func(t *testing.T) {
		api, store, filesBackend := setup(t)
		store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil)
		filesBackend.On("WriteFile", mock.Anything, mock.Anything).Return(
			func(r io.Reader, _ string) int64 { n, _ := io.Copy(io.Discard, r); return n },
			func(io.Reader, string) error { return nil },
		).Once()

		w := httptest.NewRecorder()
		api.handleUploadFile(w, uploadRequest(t, []byte("content")))

		res := w.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		var resp FileUploadResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		require.NotEmpty(t, resp.FileID)
		filesBackend.AssertNumberOfCalls(t, "WriteFile", 1)
	})
}
