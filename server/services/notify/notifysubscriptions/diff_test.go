// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package notifysubscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
)

func TestSafeBlockID(t *testing.T) {
	require.Equal(t, "", safeBlockID(nil), "nil block must yield empty id")
	require.Equal(t, "abc", safeBlockID(&model.Block{ID: "abc"}))
}
