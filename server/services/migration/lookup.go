// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package migration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-boards/server/model"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	boardsV2PluginID = "com.mattermost.boardsv2"
	legacyMapPath    = "/" + boardsV2PluginID + "/api/v1/migration/legacy-map"

	cacheTTL        = 60 * time.Second
	failureCacheTTL = 10 * time.Second
	maxCacheEntries = 10000
	maxResponseSize = 1 << 20
)

// PluginHTTPAPI is the part of the plugin API used to call Boards v2.
type PluginHTTPAPI interface {
	PluginHTTP(request *http.Request) *http.Response
}

type legacyMapResponse struct {
	Boards map[string]struct {
		ChannelID          string `json:"channel_id"`
		TeamName           string `json:"team_name"`
		ChannelName        string `json:"channel_name"`
		ChannelDisplayName string `json:"channel_display_name"`
		MovedAt            int64  `json:"moved_at"`
		CanOpen            bool   `json:"can_open"`
	} `json:"boards"`
}

type cacheEntry struct {
	info      model.BoardMigration
	expiresAt time.Time
}

// Lookup asks Boards v2 whether a legacy board was moved. Results are cached
// per board and user for a short time so board loads and permission checks
// don't each call the other plugin.
type Lookup struct {
	api    PluginHTTPAPI
	logger mlog.LoggerIFace
	now    func() time.Time

	enabled func() bool

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// Option configures optional behavior of a Lookup.
type Option func(*Lookup)

// WithEnabled makes the lookup report every board as not moved, without calling
// Boards v2, while enabled returns false.
func WithEnabled(enabled func() bool) Option {
	return func(l *Lookup) {
		l.enabled = enabled
	}
}

func NewLookup(api PluginHTTPAPI, logger mlog.LoggerIFace, opts ...Option) *Lookup {
	l := &Lookup{
		api:    api,
		logger: logger,
		now:    time.Now,
		cache:  make(map[string]cacheEntry),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Get returns the migration state of a board, or the zero value when the
// lookup is disabled. userID is passed to Boards v2 so it can report whether
// that user can open the new channel; an empty userID skips that check. When Boards v2 isn't installed or doesn't answer, the
// board is reported as not moved unless an earlier lookup found it moved.
func (l *Lookup) Get(boardID, userID string) model.BoardMigration {
	if l.enabled != nil && !l.enabled() {
		return model.BoardMigration{}
	}

	key := boardID + "|" + userID

	if info, ok := l.cached(key); ok {
		return info
	}

	info, ok := l.fetch(boardID, userID)
	ttl := cacheTTL
	if !ok {
		ttl = failureCacheTTL
		// Don't downgrade a board known to be moved because of a failed lookup,
		// or it would become writable while Boards v2 is unavailable.
		if last, found := l.lastKnown(key); found && last.Moved {
			info = last
		}
	}
	l.store(key, info, ttl)
	return info
}

// IsMoved reports whether a board was moved to Boards v2.
func (l *Lookup) IsMoved(boardID string) bool {
	return l.Get(boardID, "").Moved
}

func (l *Lookup) cached(key string) (model.BoardMigration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.cache[key]
	if !ok || !l.now().Before(entry.expiresAt) {
		return model.BoardMigration{}, false
	}
	return entry.info, true
}

func (l *Lookup) lastKnown(key string) (model.BoardMigration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.cache[key]
	return entry.info, ok
}

func (l *Lookup) store(key string, info model.BoardMigration, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.cache) >= maxCacheEntries {
		for k, e := range l.cache {
			if !now.Before(e.expiresAt) {
				delete(l.cache, k)
			}
		}
		if len(l.cache) >= maxCacheEntries {
			l.cache = make(map[string]cacheEntry)
		}
	}
	l.cache[key] = cacheEntry{info: info, expiresAt: now.Add(ttl)}
}

func (l *Lookup) fetch(boardID, userID string) (model.BoardMigration, bool) {
	query := url.Values{"board_ids": {boardID}}
	if userID != "" {
		query.Set("user_id", userID)
	}

	req, err := http.NewRequest(http.MethodGet, legacyMapPath+"?"+query.Encode(), nil)
	if err != nil {
		l.logger.Debug("Unable to build the Boards v2 migration lookup request", mlog.Err(err))
		return model.BoardMigration{}, false
	}

	resp := l.api.PluginHTTP(req)
	if resp == nil {
		return model.BoardMigration{}, false
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK || resp.Body == nil {
		return model.BoardMigration{}, false
	}

	var parsed legacyMapResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&parsed); err != nil {
		l.logger.Debug("Unable to parse the Boards v2 migration lookup response", mlog.Err(err))
		return model.BoardMigration{}, false
	}

	moved, ok := parsed.Boards[boardID]
	if !ok {
		return model.BoardMigration{}, true
	}
	return model.BoardMigration{
		Moved:              true,
		ChannelID:          moved.ChannelID,
		TeamName:           moved.TeamName,
		ChannelName:        moved.ChannelName,
		ChannelDisplayName: moved.ChannelDisplayName,
		MovedAt:            moved.MovedAt,
		CanOpen:            moved.CanOpen,
	}, true
}
