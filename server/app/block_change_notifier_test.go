// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/mockstore"
	"github.com/mattermost/mattermost-plugin-boards/server/ws"
)

const notifierWaitTimeout = 5 * time.Second

// recordedBroadcast captures a single websocket broadcast so a test can assert
// on exactly what was sent (and from which team/user) rather than on timing.
type recordedBroadcast struct {
	method  string
	teamID  string
	userID  string
	payload interface{}
}

// parkingAdapter is a ws.Adapter fake that records every broadcast and, while a
// release channel is open, parks inside the broadcast until the test releases
// it. Parking lets the assertions be causal: the operation (or App.Shutdown)
// provably cannot complete until the broadcast has been entered and released.
type parkingAdapter struct {
	mu       sync.Mutex
	events   []recordedBroadcast
	entered  chan struct{}
	release  chan struct{}
	enterOne sync.Once
}

func newParkingAdapter() *parkingAdapter {
	return &parkingAdapter{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (a *parkingAdapter) record(b recordedBroadcast) {
	a.mu.Lock()
	a.events = append(a.events, b)
	a.mu.Unlock()

	a.enterOne.Do(func() { close(a.entered) })
	<-a.release
}

func (a *parkingAdapter) recorded() []recordedBroadcast {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]recordedBroadcast(nil), a.events...)
}

func (a *parkingAdapter) BroadcastBlockChange(teamID string, block *model.Block) {
	a.record(recordedBroadcast{method: "BlockChange", teamID: teamID, payload: block})
}

func (a *parkingAdapter) BroadcastCategoryChange(category model.Category) {
	a.record(recordedBroadcast{method: "CategoryChange", teamID: category.TeamID, userID: category.UserID, payload: category})
}

func (a *parkingAdapter) BroadcastCategoryReorder(teamID, userID string, categoryOrder []string) {
	a.record(recordedBroadcast{method: "CategoryReorder", teamID: teamID, userID: userID, payload: categoryOrder})
}

func (a *parkingAdapter) BroadcastCategoryBoardsReorder(teamID, userID, categoryID string, boardsOrder []string) {
	a.record(recordedBroadcast{method: "CategoryBoardsReorder", teamID: teamID, userID: userID, payload: boardsOrder})
}

func (a *parkingAdapter) BroadcastBlockDelete(string, string, string)              {}
func (a *parkingAdapter) BroadcastBoardChange(string, *model.Board)                {}
func (a *parkingAdapter) BroadcastBoardDelete(string, string)                      {}
func (a *parkingAdapter) BroadcastMemberChange(string, string, *model.BoardMember) {}
func (a *parkingAdapter) BroadcastMemberDelete(string, string, string)             {}
func (a *parkingAdapter) BroadcastConfigChange(model.ClientConfig)                 {}
func (a *parkingAdapter) BroadcastCategoryBoardChange(string, string, []*model.BoardCategoryWebsocketData) {
}
func (a *parkingAdapter) BroadcastCardLimitTimestampChange(int64)                 {}
func (a *parkingAdapter) BroadcastSubscriptionChange(string, *model.Subscription) {}

// TestChangeNotificationsRunInlineUnderTest proves that, with the synchronous
// notifier SetupTestHelper injects, a mutating operation does not return until
// its websocket broadcast has been delivered. This is what stops a callback from
// leaking across subtests that share one gomock.Controller (the MM-69840 flake).
// It also asserts the broadcast payload for each operation.
func TestChangeNotificationsRunInlineUnderTest(t *testing.T) {
	const (
		teamID = "team_id"
		userID = "user_id"
	)

	testCases := []struct {
		name string
		// arrange registers the mock expectations and returns the operation to run.
		arrange func(th *TestHelper) func() error
		verify  func(t *testing.T, b recordedBroadcast)
	}{
		{
			name: "block patch",
			arrange: func(th *TestHelper) func() error {
				block1 := &model.Block{ID: "block1"}
				patches := &model.BlockPatchBatch{
					BlockIDs:     []string{"block1"},
					BlockPatches: []model.BlockPatch{{Title: mmModel.NewPointer("new title")}},
				}
				th.Store.EXPECT().GetBlocksByIDs([]string{"block1"}).Return([]*model.Block{block1}, nil)
				th.Store.EXPECT().PatchBlocks(gomock.Eq(patches), gomock.Eq("user-id-1")).Return(nil)
				th.Store.EXPECT().GetBlock("block1").Return(block1, nil)
				return func() error {
					return th.App.PatchBlocks("team-id", patches, "user-id-1")
				}
			},
			verify: func(t *testing.T, b recordedBroadcast) {
				require.Equal(t, "BlockChange", b.method)
				block, ok := b.payload.(*model.Block)
				require.True(t, ok)
				require.Equal(t, "block1", block.ID)
			},
		},
		{
			name: "category create",
			arrange: func(th *TestHelper) func() error {
				th.Store.EXPECT().CreateCategory(gomock.Any()).Return(nil)
				th.Store.EXPECT().GetCategory(gomock.Any()).Return(&model.Category{
					ID: "stored-category-id", Name: "Stored", TeamID: teamID, UserID: userID,
				}, nil)
				return func() error {
					_, err := th.App.CreateCategory(&model.Category{Name: "Category", UserID: userID, TeamID: teamID, Type: "custom"})
					return err
				}
			},
			verify: func(t *testing.T, b recordedBroadcast) {
				require.Equal(t, "CategoryChange", b.method)
				category, ok := b.payload.(model.Category)
				require.True(t, ok)
				// The stored (re-read) category must be broadcast, not the submitted one.
				require.Equal(t, "stored-category-id", category.ID)
			},
		},
		{
			name: "category update",
			arrange: func(th *TestHelper) func() error {
				// UpdateCategory validates the category ID, so use a valid one.
				th.Store.EXPECT().GetCategory(gomock.Any()).Return(&model.Category{
					ID: testCategoryID1, Name: "Category", TeamID: teamID, UserID: userID, Type: "custom",
				}, nil)
				th.Store.EXPECT().UpdateCategory(gomock.Any()).Return(nil)
				th.Store.EXPECT().GetCategory(testCategoryID1).Return(&model.Category{
					ID: testCategoryID1, Name: "Updated", TeamID: teamID, UserID: userID,
				}, nil)
				return func() error {
					_, err := th.App.UpdateCategory(&model.Category{
						ID: testCategoryID1, Name: "Category", UserID: userID, TeamID: teamID, Type: "custom",
					})
					return err
				}
			},
			verify: func(t *testing.T, b recordedBroadcast) {
				require.Equal(t, "CategoryChange", b.method)
				category, ok := b.payload.(model.Category)
				require.True(t, ok)
				require.Equal(t, "Updated", category.Name)
			},
		},
		{
			name: "category delete",
			arrange: func(th *TestHelper) func() error {
				th.Store.EXPECT().GetCategory("stored-category-id").Return(&model.Category{
					ID: "stored-category-id", DeleteAt: 0, UserID: userID, TeamID: teamID, Type: "custom",
				}, nil)
				th.Store.EXPECT().GetUserCategoryBoards(userID, teamID).Return([]model.CategoryBoards{
					{Category: model.Category{ID: "default-id", UserID: userID, TeamID: teamID, Type: "default", Name: defaultCategoryBoards}},
					{Category: model.Category{ID: "stored-category-id", UserID: userID, TeamID: teamID, Type: "custom", Name: "Category"}},
				}, nil)
				th.Store.EXPECT().DeleteCategory("stored-category-id", userID, teamID).Return(nil)
				// The post-delete re-read: a DeleteAt timestamp proves the deleted row is broadcast.
				th.Store.EXPECT().GetCategory("stored-category-id").Return(&model.Category{
					ID: "stored-category-id", DeleteAt: 10000, UserID: userID, TeamID: teamID,
				}, nil)
				return func() error {
					_, err := th.App.DeleteCategory("stored-category-id", userID, teamID)
					return err
				}
			},
			verify: func(t *testing.T, b recordedBroadcast) {
				require.Equal(t, "CategoryChange", b.method)
				category, ok := b.payload.(model.Category)
				require.True(t, ok)
				require.Equal(t, int64(10000), category.DeleteAt)
			},
		},
		{
			name: "categories reorder",
			arrange: func(th *TestHelper) func() error {
				th.Store.EXPECT().GetUserCategories(userID, teamID).Return([]model.Category{
					{ID: "cat1"}, {ID: "cat2"},
				}, nil)
				th.Store.EXPECT().ReorderCategories(userID, teamID, []string{"cat2", "cat1"}).
					Return([]string{"cat2", "cat1"}, nil)
				return func() error {
					_, err := th.App.ReorderCategories(userID, teamID, []string{"cat2", "cat1"})
					return err
				}
			},
			verify: func(t *testing.T, b recordedBroadcast) {
				require.Equal(t, "CategoryReorder", b.method)
				require.Equal(t, teamID, b.teamID)
				require.Equal(t, userID, b.userID)
				require.Equal(t, []string{"cat2", "cat1"}, b.payload)
			},
		},
		{
			name: "category boards reorder",
			arrange: func(th *TestHelper) func() error {
				th.Store.EXPECT().GetUserCategoryBoards(userID, teamID).Return([]model.CategoryBoards{
					{
						Category: model.Category{ID: "cat1", Name: "Category 1"},
						BoardMetadata: []model.CategoryBoardMetadata{
							{BoardID: "board1"}, {BoardID: "board2"},
						},
					},
					// A default "Boards" category must exist, else GetUserCategoryBoards
					// would try to create one.
					{Category: model.Category{ID: "default-cat", Name: defaultCategoryBoards, Type: "system"}},
				}, nil)
				th.Store.EXPECT().ReorderCategoryBoards("cat1", []string{"board2", "board1"}).
					Return([]string{"board2", "board1"}, nil)
				return func() error {
					_, err := th.App.ReorderCategoryBoards(userID, teamID, "cat1", []string{"board2", "board1"})
					return err
				}
			},
			verify: func(t *testing.T, b recordedBroadcast) {
				require.Equal(t, "CategoryBoardsReorder", b.method)
				require.Equal(t, teamID, b.teamID)
				require.Equal(t, userID, b.userID)
				require.Equal(t, []string{"board2", "board1"}, b.payload)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := newParkingAdapter()
			th, tearDown := setupTestHelper(t, adapter)
			defer tearDown()

			// Release the broadcast on cleanup so a failed assertion never leaks
			// the parked operation goroutine.
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(adapter.release) }) }
			defer release()

			runOp := tc.arrange(th)

			done := make(chan error, 1)
			go func() { done <- runOp() }()

			select {
			case <-adapter.entered:
			case err := <-done:
				t.Fatalf("operation returned before delivering its broadcast: %v", err)
			case <-time.After(notifierWaitTimeout):
				t.Fatal("broadcast was never delivered")
			}

			// The operation must still be parked inside the broadcast: proof that
			// delivery happens inline, before the operation returns.
			select {
			case err := <-done:
				t.Fatalf("operation returned while its broadcast was still in flight: %v", err)
			case <-time.After(50 * time.Millisecond):
			}

			release()

			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(notifierWaitTimeout):
				t.Fatal("operation did not return after broadcast completed")
			}

			// Each covered operation emits exactly one broadcast; Len guards
			// against a regression that emits an extra, unexpected one.
			events := adapter.recorded()
			require.Len(t, events, 1)
			tc.verify(t, events[0])
		})
	}
}

// TestShutdownWaitsForInFlightChangeNotification runs the production
// asynchronous utils.CallbackQueue and proves App.Shutdown() blocks until an
// in-flight broadcast has finished, so a change notification can never outlive
// the App. (Draining of the queued-but-not-yet-started backlog is covered by
// utils/callbackqueue_test.go.)
func TestShutdownWaitsForInFlightChangeNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockstore.NewMockStore(ctrl)
	logger, _ := mlog.NewLogger()
	t.Cleanup(func() { _ = logger.Shutdown() })

	adapter := newParkingAdapter()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(adapter.release) }) }

	cfg := config.Configuration{}
	// BlockChangeNotifier is left unset so New falls back to the production queue.
	app := New(&cfg, adapter, Services{
		Store:            store,
		Logger:           logger,
		SkipTemplateInit: true,
	})

	// Release first so a parked callback can finish, then shut down the
	// production queue even if an assertion fails after New.
	var shutdownOnce sync.Once
	shutdown := func() { shutdownOnce.Do(app.Shutdown) }
	t.Cleanup(func() {
		release()
		shutdown()
	})

	store.EXPECT().CreateCategory(gomock.Any()).Return(nil)
	store.EXPECT().GetCategory(gomock.Any()).Return(&model.Category{ID: "stored-category-id"}, nil)

	_, err := app.CreateCategory(&model.Category{Name: "Category", UserID: "user_id", TeamID: "team_id", Type: "custom"})
	require.NoError(t, err)

	// Wait until a worker goroutine is actually inside the broadcast.
	select {
	case <-adapter.entered:
	case <-time.After(notifierWaitTimeout):
		t.Fatal("broadcast was never dispatched by the queue")
	}

	shutdownReturned := make(chan struct{})
	go func() {
		shutdown()
		close(shutdownReturned)
	}()

	// Shutdown cannot complete while the broadcast is parked on a worker.
	select {
	case <-shutdownReturned:
		t.Fatal("Shutdown returned while a broadcast was still in flight")
	case <-time.After(100 * time.Millisecond):
	}

	release()

	select {
	case <-shutdownReturned:
	case <-time.After(notifierWaitTimeout):
		t.Fatal("Shutdown did not return after the broadcast completed")
	}

	events := adapter.recorded()
	require.Len(t, events, 1)
	require.Equal(t, "CategoryChange", events[0].method)
	category, ok := events[0].payload.(model.Category)
	require.True(t, ok)
	require.Equal(t, "stored-category-id", category.ID)
}

var _ ws.Adapter = (*parkingAdapter)(nil)
