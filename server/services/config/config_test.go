// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMovedBoardsReadOnlyActive(t *testing.T) {
	tests := []struct {
		name       string
		legacyMode bool
		readOnly   bool
		want       bool
	}{
		{"legacy mode and setting on", true, true, true},
		{"legacy mode on and setting off", true, false, false},
		{"legacy mode off and setting on", false, true, false},
		{"legacy mode off and setting off", false, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Configuration{LegacyMode: tc.legacyMode, MovedBoardsReadOnly: tc.readOnly}
			require.Equal(t, tc.want, cfg.MovedBoardsReadOnlyActive())
		})
	}
}

func TestReadConfigFileMovedBoardsReadOnlyDefault(t *testing.T) {
	read := func(t *testing.T, content string) *Configuration {
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		cfg, err := ReadConfigFile(path)
		require.NoError(t, err)
		return cfg
	}

	t.Run("defaults to on when unset", func(t *testing.T) {
		require.True(t, read(t, `{}`).MovedBoardsReadOnly)
	})

	t.Run("keeps an explicit off", func(t *testing.T) {
		require.False(t, read(t, `{"movedBoardsReadOnly": false}`).MovedBoardsReadOnly)
	})
}
