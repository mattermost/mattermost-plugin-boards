// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-boards/server/app"
	"github.com/mattermost/mattermost-plugin-boards/server/auth"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/metrics"
	"github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mmpermissions"
	mmpermissionsMocks "github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mmpermissions/mocks"
	permissionsMocks "github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mocks"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/mockstore"
	"github.com/mattermost/mattermost-plugin-boards/server/services/webhook"
	"github.com/mattermost/mattermost-plugin-boards/server/ws"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore/mocks"
)

func setupArchiveImportAPI(t *testing.T, maxFileSize int64) (*API, func()) {
	t.Helper()

	ctrl := gomock.NewController(t)
	cfg := config.Configuration{MaxFileSize: maxFileSize}
	store := mockstore.NewMockStore(ctrl)
	filesBackend := &mocks.FileBackend{}
	authService := auth.New(&cfg, store, nil)
	logger, _ := mlog.NewLogger()
	wsserver := ws.NewServer(authService, logger, store)
	webhookClient := webhook.NewClient(&cfg, logger)
	metricsService := metrics.NewMetrics(metrics.InstanceInfo{})

	permStore := permissionsMocks.NewMockStore(ctrl)
	pluginAPI := mmpermissionsMocks.NewMockAPI(ctrl)
	pluginAPI.EXPECT().HasPermissionToTeam(gomock.Any(), gomock.Any(), model.PermissionViewTeam).Return(true)
	store.EXPECT().GetUserByID(gomock.Any()).Return(&model.User{ID: "user", IsGuest: false}, nil)
	permissions := mmpermissions.New(permStore, pluginAPI, mlog.CreateConsoleTestLogger(t))

	testApp := app.New(&cfg, wsserver, app.Services{
		Auth:             authService,
		Store:            store,
		FilesBackend:     filesBackend,
		Webhook:          webhookClient,
		Metrics:          metricsService,
		Logger:           logger,
		SkipTemplateInit: true,
		Permissions:      permissions,
	})

	api := NewAPI(testApp, "", "", permissions, mlog.CreateConsoleTestLogger(t), nil)

	tearDown := func() {
		testApp.Shutdown()
		if logger != nil {
			_ = logger.Shutdown()
		}
	}

	return api, tearDown
}

func archiveImportRequest(t *testing.T, teamID string, body io.Reader, contentType string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID+"/archive/import", body)
	req = mux.SetURLVars(req, map[string]string{"teamID": teamID})
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	session := &model.Session{UserID: "user"}
	ctx := context.WithValue(req.Context(), sessionContextKey, session)
	return req.WithContext(ctx)
}

func TestHandleArchiveImportFormErrors(t *testing.T) {
	const teamID = "abcdefghijklmnopqrstuvwxyz"

	t.Run("returns 413 for oversized multipart body", func(t *testing.T) {
		api, tearDown := setupArchiveImportAPI(t, 1)
		defer tearDown()

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile(UploadFormFileKey, "archive.boardarchive")
		require.NoError(t, err)
		_, err = part.Write([]byte("too-large"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		req := archiveImportRequest(t, teamID, &body, writer.FormDataContentType())
		w := httptest.NewRecorder()

		api.handleArchiveImport(w, req)

		res := w.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
		require.Equal(t, "application/json", res.Header.Get("Content-Type"))
		b, readErr := io.ReadAll(res.Body)
		require.NoError(t, readErr)
		require.Contains(t, string(b), "entity too large")
	})

	t.Run("returns 400 for malformed upload", func(t *testing.T) {
		api, tearDown := setupArchiveImportAPI(t, 0)
		defer tearDown()

		req := archiveImportRequest(t, teamID, bytes.NewReader([]byte("not-a-multipart-body")), "text/plain")
		w := httptest.NewRecorder()

		api.handleArchiveImport(w, req)

		res := w.Result()
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t, "application/json", res.Header.Get("Content-Type"))
	})
}
