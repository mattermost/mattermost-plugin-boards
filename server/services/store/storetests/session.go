// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package storetests

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
)

func StoreTestSessionStore(t *testing.T, setup func(t *testing.T) (store.Store, func())) {
	t.Run("GetActiveUserCount", func(t *testing.T) {
		t.Skip("Skipping standalone-only test: GetActiveUserCount is only used in standalone server telemetry")
	})
}
