// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
package app

import (
	"context"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/mattermost/mattermost-plugin-boards/server/auth"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/metrics"
	"github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mmpermissions"
	mmpermissionsMocks "github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mmpermissions/mocks"
	permissionsMocks "github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mocks"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/mockstore"
	"github.com/mattermost/mattermost-plugin-boards/server/services/webhook"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/mattermost/mattermost-plugin-boards/server/ws"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore/mocks"
)

// synchronousChangeNotifier runs each callback inline on Enqueue so a broadcast
// is delivered before the operation that triggered it returns. SetupTestHelper
// injects it so a mutating op's mock calls (GetBlock, GetMembersForBoard, ...)
// are satisfied before the next subtest runs, which stops callbacks from leaking
// across subtests that share one gomock.Controller.
//
// Trade-off: because the callback no longer runs on a worker goroutine
// concurrently with its caller, -race cannot observe a conflict between the two
// inside these unit tests. The production utils.CallbackQueue stays covered
// under -race by TestShutdownWaitsForInFlightChangeNotification,
// utils/callbackqueue_test.go, and server/integrationtests.
type synchronousChangeNotifier struct{}

func (n *synchronousChangeNotifier) Enqueue(f utils.CallbackFunc) {
	if f == nil {
		return
	}
	_ = f()
}

func (n *synchronousChangeNotifier) Shutdown(_ context.Context) bool {
	return true
}

type TestHelper struct {
	App          *App
	Store        *mockstore.MockStore
	FilesBackend *mocks.FileBackend
	logger       mlog.LoggerIFace
	API          *mmpermissionsMocks.MockAPI
	PermStore    *permissionsMocks.MockStore
}

func SetupTestHelper(t *testing.T) (*TestHelper, func()) {
	return setupTestHelper(t, nil)
}

// setupTestHelper builds a TestHelper. When wsAdapter is nil a real ws.Server is
// used; tests that need to observe broadcasts can pass a recording adapter. In
// both cases a synchronous BlockChangeNotifier is injected (see
// synchronousChangeNotifier) so change notifications are delivered inline.
func setupTestHelper(t *testing.T, wsAdapter ws.Adapter) (*TestHelper, func()) {
	ctrl := gomock.NewController(t)
	cfg := config.Configuration{}
	store := mockstore.NewMockStore(ctrl)
	filesBackend := &mocks.FileBackend{}
	auth := auth.New(&cfg, store, nil)
	logger, _ := mlog.NewLogger()
	if wsAdapter == nil {
		wsAdapter = ws.NewServer(auth, logger, store)
	}
	webhook := webhook.NewClient(&cfg, logger)
	metricsService := metrics.NewMetrics(metrics.InstanceInfo{})

	mockStore := permissionsMocks.NewMockStore(ctrl)
	mockAPI := mmpermissionsMocks.NewMockAPI(ctrl)
	permissions := mmpermissions.New(mockStore, mockAPI, mlog.CreateConsoleTestLogger(t))

	appServices := Services{
		Auth:                auth,
		Store:               store,
		FilesBackend:        filesBackend,
		Webhook:             webhook,
		Metrics:             metricsService,
		Logger:              logger,
		SkipTemplateInit:    true,
		Permissions:         permissions,
		BlockChangeNotifier: &synchronousChangeNotifier{},
	}
	app2 := New(&cfg, wsAdapter, appServices)

	var once sync.Once
	tearDown := func() {
		once.Do(func() {
			app2.Shutdown()
			if logger != nil {
				_ = logger.Shutdown()
			}
		})
	}
	// Register teardown so it runs even when a caller discards the returned
	// func, and so it runs before gomock verifies expectations: t.Cleanup is
	// LIFO and gomock.NewController already registered its own cleanup.
	t.Cleanup(tearDown)

	return &TestHelper{
		App:          app2,
		Store:        store,
		FilesBackend: filesBackend,
		logger:       logger,
		API:          mockAPI,
		PermStore:    mockStore,
	}, tearDown
}

func (th *TestHelper) expectBoardAdmin(userID, boardID, teamID string) { //nolint:unparam
	th.PermStore.EXPECT().GetBoard(boardID).Return(&model.Board{
		ID:     boardID,
		TeamID: teamID,
	}, nil)
	th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionViewTeam).Return(true)
	th.PermStore.EXPECT().GetMemberForBoard(boardID, userID).Return(&model.BoardMember{
		BoardID:     boardID,
		UserID:      userID,
		SchemeAdmin: true,
	}, nil)
	// SchemeAdmin path triggers isGuest() in mmpermissions which loads the user
	th.PermStore.EXPECT().GetUserByID(userID).Return(&model.User{ID: userID}, nil)
}

func (th *TestHelper) expectBoardEditor(userID, boardID, teamID string) {
	th.PermStore.EXPECT().GetBoard(boardID).Return(&model.Board{
		ID:     boardID,
		TeamID: teamID,
	}, nil)
	th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionViewTeam).Return(true)
	th.PermStore.EXPECT().GetMemberForBoard(boardID, userID).Return(&model.BoardMember{
		BoardID:      boardID,
		UserID:       userID,
		SchemeEditor: true,
	}, nil)
	th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionManageTeam).Return(false)
}

func (th *TestHelper) expectBoardImportPermissions(userID, teamID string, boardType model.BoardType) {
	if boardType == model.BoardTypeOpen {
		th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionCreatePublicChannel).Return(true)
		return
	}
	th.API.EXPECT().HasPermissionToTeam(userID, teamID, model.PermissionCreatePrivateChannel).Return(true)
}
