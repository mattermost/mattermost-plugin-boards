// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
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
)

type fakeBoardPermissions struct {
	canViewBoard bool
	canViewTeam  bool
}

func (f fakeBoardPermissions) HasPermissionTo(string, *mmModel.Permission) bool { return false }
func (f fakeBoardPermissions) HasPermissionToTeam(string, string, *mmModel.Permission) bool {
	return f.canViewTeam
}
func (f fakeBoardPermissions) HasPermissionToChannel(string, string, *mmModel.Permission) bool {
	return false
}
func (f fakeBoardPermissions) HasPermissionToBoard(string, string, *mmModel.Permission) bool {
	return f.canViewBoard
}

type fakeMigrationLookup struct {
	info   model.BoardMigration
	userID string
}

func (f *fakeMigrationLookup) Get(_, userID string) model.BoardMigration {
	f.userID = userID
	return f.info
}

func TestGetBoardMigration(t *testing.T) {
	moved := model.BoardMigration{
		Moved:              true,
		ChannelID:          "chan-1",
		TeamName:           "eng",
		ChannelName:        "roadmap",
		ChannelDisplayName: "Roadmap",
		MovedAt:            1700000000000,
		CanOpen:            true,
	}

	setup := func(t *testing.T, lookup *fakeMigrationLookup, perms fakeBoardPermissions, board *model.Board, readOnly bool) http.Handler {
		ctrl := gomock.NewController(t)
		store := mockstore.NewMockStore(ctrl)
		store.EXPECT().GetBoard("board-1").Return(board, nil).AnyTimes()
		store.EXPECT().GetUserByID("user-1").Return(&model.User{ID: "user-1"}, nil).AnyTimes()

		var migrationLookup app.MigrationLookup
		if lookup != nil {
			migrationLookup = lookup
		}

		cfg := &config.Configuration{MovedBoardsReadOnly: readOnly}
		logger := mlog.CreateConsoleTestLogger(t)
		authService := auth.New(cfg, store, nil)
		application := app.New(cfg, ws.NewServer(authService, logger, store), app.Services{
			Auth:             authService,
			Store:            store,
			Webhook:          webhook.NewClient(cfg, logger),
			Metrics:          metrics.NewMetrics(metrics.InstanceInfo{}),
			Logger:           logger,
			Permissions:      perms,
			SkipTemplateInit: true,
			MigrationLookup:  migrationLookup,
		})
		t.Cleanup(application.Shutdown)

		auditService, err := audit.NewAudit()
		require.NoError(t, err)

		router := mux.NewRouter()
		NewAPI(application, "", "", perms, logger, auditService).RegisterRoutes(router)
		return router
	}

	get := func(handler http.Handler, userID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/boards/board-1/migration", nil)
		req.Header.Set(HeaderRequestedWith, HeaderRequestedWithXML)
		if userID != "" {
			req.Header.Set("Mattermost-User-Id", userID)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	openBoard := &model.Board{ID: "board-1", TeamID: "team-1", Type: model.BoardTypeOpen}
	privateBoard := &model.Board{ID: "board-1", TeamID: "team-1", Type: model.BoardTypePrivate}

	t.Run("returns the migration info for the requesting user", func(t *testing.T) {
		lookup := &fakeMigrationLookup{info: moved}
		rec := get(setup(t, lookup, fakeBoardPermissions{canViewTeam: true}, openBoard, false), "user-1")

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "user-1", lookup.userID)

		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]interface{}{
			"moved":                true,
			"channel_id":           "chan-1",
			"team_name":            "eng",
			"channel_name":         "roadmap",
			"channel_display_name": "Roadmap",
			"moved_at":             float64(1700000000000),
			"can_open":             true,
			"read_only":            false,
		}, body)
	})

	t.Run("reports read-only when the setting is on", func(t *testing.T) {
		rec := get(setup(t, &fakeMigrationLookup{info: moved}, fakeBoardPermissions{canViewTeam: true}, openBoard, true), "user-1")

		require.Equal(t, http.StatusOK, rec.Code)
		var info model.BoardMigration
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &info))
		require.True(t, info.Moved)
		require.True(t, info.ReadOnly)
	})

	t.Run("returns the empty answer when the board did not move", func(t *testing.T) {
		rec := get(setup(t, &fakeMigrationLookup{}, fakeBoardPermissions{canViewTeam: true}, openBoard, true), "user-1")

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"moved":false,"can_open":false,"read_only":false}`, rec.Body.String())
	})

	t.Run("returns the empty answer when Boards v2 lookup is unavailable", func(t *testing.T) {
		handler := setup(t, nil, fakeBoardPermissions{canViewTeam: true}, openBoard, true)
		rec := get(handler, "user-1")

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"moved":false,"can_open":false,"read_only":false}`, rec.Body.String())
	})

	t.Run("requires a session", func(t *testing.T) {
		rec := get(setup(t, &fakeMigrationLookup{info: moved}, fakeBoardPermissions{canViewTeam: true}, openBoard, false), "")

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("denies users who can't view an open board's team", func(t *testing.T) {
		rec := get(setup(t, &fakeMigrationLookup{info: moved}, fakeBoardPermissions{}, openBoard, false), "user-1")

		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("denies users who can't view a private board", func(t *testing.T) {
		rec := get(setup(t, &fakeMigrationLookup{info: moved}, fakeBoardPermissions{canViewTeam: true}, privateBoard, false), "user-1")

		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("allows board members of a private board", func(t *testing.T) {
		rec := get(setup(t, &fakeMigrationLookup{info: moved}, fakeBoardPermissions{canViewBoard: true}, privateBoard, false), "user-1")

		require.Equal(t, http.StatusOK, rec.Code)
	})
}
