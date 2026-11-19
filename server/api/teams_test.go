// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

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

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore/mocks"
)

func TestHandleGetTeamNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := config.Configuration{}
	store := mockstore.NewMockStore(ctrl)
	authService := auth.New(&cfg, store, nil)
	logger := mlog.CreateConsoleTestLogger(t)

	pluginAPI := mmpermissionsMocks.NewMockAPI(ctrl)
	pluginAPI.EXPECT().HasPermissionToTeam("user", "undefined", model.PermissionViewTeam).Return(true)
	permissions := mmpermissions.New(permissionsMocks.NewMockStore(ctrl), pluginAPI, logger)

	testApp := app.New(&cfg, ws.NewServer(authService, logger, store), app.Services{
		Auth:             authService,
		Store:            store,
		FilesBackend:     &mocks.FileBackend{},
		Webhook:          webhook.NewClient(&cfg, logger),
		Metrics:          metrics.NewMetrics(metrics.InstanceInfo{}),
		Logger:           logger,
		SkipTemplateInit: true,
		Permissions:      permissions,
	})
	defer testApp.Shutdown()
	api := NewAPI(testApp, "", "", permissions, logger, nil)

	store.EXPECT().GetTeam("undefined").Return(nil, model.NewErrNotFound("team"))

	req := httptest.NewRequest(http.MethodGet, "/teams/undefined", nil)
	req = mux.SetURLVars(req, map[string]string{"teamID": "undefined"})
	req = req.WithContext(context.WithValue(req.Context(), sessionContextKey, &model.Session{UserID: "user"}))
	w := httptest.NewRecorder()

	require.NotPanics(t, func() { api.handleGetTeam(w, req) })
	require.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}
