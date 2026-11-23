// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package migration

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-boards/server/model"

	"github.com/mattermost/mattermost/server/public/shared/mlog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePluginAPI struct {
	mu       sync.Mutex
	requests []*http.Request
	respond  func(*http.Request) *http.Response
}

func (f *fakePluginAPI) PluginHTTP(request *http.Request) *http.Response {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
	return f.respond(request)
}

func (f *fakePluginAPI) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

const movedBody = `{"boards":{"board-1":{"channel_id":"chan-1","team_id":"team-1","team_name":"eng","channel_name":"roadmap","channel_display_name":"Roadmap","moved_at":1700000000000,"can_open":true}}}`

func newTestLookup(t *testing.T, respond func(*http.Request) *http.Response) (*Lookup, *fakePluginAPI, *time.Time) {
	api := &fakePluginAPI{respond: respond}
	l := NewLookup(api, mlog.CreateConsoleTestLogger(t))
	now := time.Unix(1700000100, 0)
	l.now = func() time.Time { return now }
	return l, api, &now
}

func TestLookupGet(t *testing.T) {
	t.Run("parses a moved board and queries Boards v2 with the user", func(t *testing.T) {
		l, api, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, movedBody) })

		info := l.Get("board-1", "user-1")

		assert.Equal(t, model.BoardMigration{
			Moved:              true,
			ChannelID:          "chan-1",
			TeamName:           "eng",
			ChannelName:        "roadmap",
			ChannelDisplayName: "Roadmap",
			MovedAt:            1700000000000,
			CanOpen:            true,
		}, info)

		require.Equal(t, 1, api.callCount())
		req := api.requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/com.mattermost.boardsv2/api/v1/migration/legacy-map", req.URL.Path)
		assert.Equal(t, "board-1", req.URL.Query().Get("board_ids"))
		assert.Equal(t, "user-1", req.URL.Query().Get("user_id"))
	})

	t.Run("omits user_id when there is no user", func(t *testing.T) {
		l, api, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, movedBody) })

		assert.True(t, l.IsMoved("board-1"))

		require.Equal(t, 1, api.callCount())
		assert.False(t, api.requests[0].URL.Query().Has("user_id"))
	})

	t.Run("reports a board missing from the response as not moved", func(t *testing.T) {
		l, _, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, `{"boards":{}}`) })

		assert.Equal(t, model.BoardMigration{}, l.Get("board-2", "user-1"))
		assert.False(t, l.IsMoved("board-2"))
	})

	t.Run("treats a nil response as no information", func(t *testing.T) {
		l, _, _ := newTestLookup(t, func(*http.Request) *http.Response { return nil })

		assert.Equal(t, model.BoardMigration{}, l.Get("board-1", "user-1"))
	})

	t.Run("treats a non-200 response as no information", func(t *testing.T) {
		for _, status := range []int{http.StatusNotFound, http.StatusServiceUnavailable, http.StatusForbidden} {
			l, _, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(status, movedBody) })

			assert.Equal(t, model.BoardMigration{}, l.Get("board-1", "user-1"), "status %d", status)
		}
	})

	t.Run("treats an invalid body as no information", func(t *testing.T) {
		l, _, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, "not json") })

		assert.Equal(t, model.BoardMigration{}, l.Get("board-1", "user-1"))
	})
}

func TestLookupEnabled(t *testing.T) {
	t.Run("does not call Boards v2 and reports not moved while disabled", func(t *testing.T) {
		enabled := false
		l, api, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, movedBody) })
		WithEnabled(func() bool { return enabled })(l)

		assert.Equal(t, model.BoardMigration{}, l.Get("board-1", "user-1"))
		assert.False(t, l.IsMoved("board-1"))
		assert.Zero(t, api.callCount())

		enabled = true
		assert.True(t, l.IsMoved("board-1"))
		assert.Equal(t, 1, api.callCount())

		enabled = false
		assert.False(t, l.IsMoved("board-1"))
		assert.Equal(t, 1, api.callCount())
	})

	t.Run("is enabled by default", func(t *testing.T) {
		l, api, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, movedBody) })

		assert.True(t, l.IsMoved("board-1"))
		assert.Equal(t, 1, api.callCount())
	})
}

func TestLookupCache(t *testing.T) {
	t.Run("serves repeated lookups from the cache until it expires", func(t *testing.T) {
		l, api, now := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, movedBody) })

		l.Get("board-1", "user-1")
		l.Get("board-1", "user-1")
		assert.Equal(t, 1, api.callCount())

		*now = now.Add(cacheTTL - time.Second)
		l.Get("board-1", "user-1")
		assert.Equal(t, 1, api.callCount())

		*now = now.Add(2 * time.Second)
		l.Get("board-1", "user-1")
		assert.Equal(t, 2, api.callCount())
	})

	t.Run("caches per board and user", func(t *testing.T) {
		l, api, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, movedBody) })

		l.Get("board-1", "user-1")
		l.Get("board-1", "user-2")
		l.Get("board-2", "user-1")
		assert.Equal(t, 3, api.callCount())
	})

	t.Run("retries failures sooner than successes", func(t *testing.T) {
		l, api, now := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusNotFound, "") })

		l.Get("board-1", "user-1")
		l.Get("board-1", "user-1")
		assert.Equal(t, 1, api.callCount())

		*now = now.Add(failureCacheTTL + time.Second)
		l.Get("board-1", "user-1")
		assert.Equal(t, 2, api.callCount())
	})

	t.Run("keeps a board known to be moved when a later lookup fails", func(t *testing.T) {
		fail := false
		l, api, now := newTestLookup(t, func(*http.Request) *http.Response {
			if fail {
				return jsonResponse(http.StatusInternalServerError, "")
			}
			return jsonResponse(http.StatusOK, movedBody)
		})

		require.True(t, l.IsMoved("board-1"))

		fail = true
		*now = now.Add(cacheTTL + time.Second)
		assert.True(t, l.IsMoved("board-1"))
		assert.Equal(t, 2, api.callCount())

		*now = now.Add(failureCacheTTL - time.Second)
		assert.True(t, l.IsMoved("board-1"))
		assert.Equal(t, 2, api.callCount())

		*now = now.Add(2 * time.Second)
		assert.True(t, l.IsMoved("board-1"))
		assert.Equal(t, 3, api.callCount())
	})

	t.Run("reports not moved when a lookup fails for a board never seen as moved", func(t *testing.T) {
		l, _, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusInternalServerError, "") })

		assert.False(t, l.IsMoved("board-1"))
	})

	t.Run("stays within the size bound", func(t *testing.T) {
		l, _, _ := newTestLookup(t, func(*http.Request) *http.Response { return jsonResponse(http.StatusOK, `{"boards":{}}`) })

		for i := 0; i < maxCacheEntries+10; i++ {
			l.store(fmt.Sprintf("board-%d|user", i), model.BoardMigration{}, cacheTTL)
		}
		assert.LessOrEqual(t, len(l.cache), maxCacheEntries)
	})
}
