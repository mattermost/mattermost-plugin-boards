// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-boards/server/auth"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/metrics"
	"github.com/mattermost/mattermost-plugin-boards/server/services/notify"
	"github.com/mattermost/mattermost-plugin-boards/server/services/permissions"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/mattermost/mattermost-plugin-boards/server/services/webhook"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/mattermost/mattermost-plugin-boards/server/ws"

	mm_model "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
)

const (
	blockChangeNotifierQueueSize       = 1000
	blockChangeNotifierPoolSize        = 10
	blockChangeNotifierShutdownTimeout = time.Second * 10
)

type servicesAPI interface {
	GetUsersFromProfiles(options *mm_model.UserGetOptions) ([]*mm_model.User, error)
}

// BlockChangeNotifier dispatches the websocket/webhook/notification callbacks
// that follow a mutating operation. Production uses an asynchronous
// utils.CallbackQueue; tests can supply a synchronous implementation so a
// broadcast is delivered before the operation returns.
type BlockChangeNotifier interface {
	Enqueue(f utils.CallbackFunc)
	Shutdown(ctx context.Context) bool
}

type ReadCloseSeeker = filestore.ReadCloseSeeker

type fileBackend interface {
	Reader(path string) (ReadCloseSeeker, error)
	FileExists(path string) (bool, error)
	CopyFile(oldPath, newPath string) error
	MoveFile(oldPath, newPath string) error
	WriteFile(fr io.Reader, path string) (int64, error)
	RemoveFile(path string) error
}

type Services struct {
	Auth             *auth.Auth
	Store            store.Store
	FilesBackend     fileBackend
	Webhook          *webhook.Client
	Metrics          *metrics.Metrics
	Notifications    *notify.Service
	Logger           mlog.LoggerIFace
	Permissions      permissions.PermissionsService
	SkipTemplateInit bool
	ServicesAPI      servicesAPI
	// BlockChangeNotifier is optional. When nil, New falls back to the
	// production asynchronous utils.CallbackQueue.
	BlockChangeNotifier BlockChangeNotifier
}

type App struct {
	config              *config.Configuration
	store               store.Store
	auth                *auth.Auth
	wsAdapter           ws.Adapter
	filesBackend        fileBackend
	webhook             *webhook.Client
	metrics             *metrics.Metrics
	notifications       *notify.Service
	logger              mlog.LoggerIFace
	permissions         permissions.PermissionsService
	blockChangeNotifier BlockChangeNotifier
	servicesAPI         servicesAPI

	cardLimitMux sync.RWMutex
	cardLimit    int
}

func (a *App) SetConfig(config *config.Configuration) {
	a.config = config
}

func (a *App) GetConfig() *config.Configuration {
	return a.config
}

func New(config *config.Configuration, wsAdapter ws.Adapter, services Services) *App {
	blockChangeNotifier := services.BlockChangeNotifier
	if blockChangeNotifier == nil {
		blockChangeNotifier = utils.NewCallbackQueue("blockChangeNotifier", blockChangeNotifierQueueSize, blockChangeNotifierPoolSize, services.Logger)
	}

	app := &App{
		config:              config,
		store:               services.Store,
		auth:                services.Auth,
		wsAdapter:           wsAdapter,
		filesBackend:        services.FilesBackend,
		webhook:             services.Webhook,
		metrics:             services.Metrics,
		notifications:       services.Notifications,
		logger:              services.Logger,
		permissions:         services.Permissions,
		blockChangeNotifier: blockChangeNotifier,
		servicesAPI:         services.ServicesAPI,
	}
	app.initialize(services.SkipTemplateInit)
	return app
}

func (a *App) CardLimit() int {
	a.cardLimitMux.RLock()
	defer a.cardLimitMux.RUnlock()
	return a.cardLimit
}

func (a *App) SetCardLimit(cardLimit int) {
	a.cardLimitMux.Lock()
	defer a.cardLimitMux.Unlock()
	a.cardLimit = cardLimit
}

func (a *App) GetLicense() *mm_model.License {
	return a.store.GetLicense()
}
