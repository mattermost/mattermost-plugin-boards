// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"context"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// initialize is called when the App is first created.
func (a *App) initialize(skipTemplateInit bool, skipMigrations bool) {
	if !skipTemplateInit {
		if err := a.InitTemplates(); err != nil {
			a.logger.Error(`InitializeTemplates failed`, mlog.Err(err))
		}
	}
	if !skipMigrations {
		// RunFileOwnershipMigration is a *sqlstore.SQLStore method not in the Store
		// interface (the generator can't handle cross-package parameter types), so we
		// call it via a narrow structural type assertion.
		type fileOwnershipMigrator interface {
			RunFileOwnershipMigration(
				moveFile func(oldPath, newPath string) error,
				fileExists func(path string) (bool, error),
			)
		}
		if m, ok := a.store.(fileOwnershipMigrator); ok {
			m.RunFileOwnershipMigration(a.filesBackend.MoveFile, a.filesBackend.FileExists)
		}
	}
}

func (a *App) Shutdown() {
	if a.blockChangeNotifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), blockChangeNotifierShutdownTimeout)
		defer cancel()
		if !a.blockChangeNotifier.Shutdown(ctx) {
			a.logger.Warn("blockChangeNotifier shutdown timed out")
		}
	}
}
