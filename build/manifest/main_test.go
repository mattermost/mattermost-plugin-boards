// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestDistManifest(t *testing.T) {
	t.Run("writes plugin.json with the resolved version into the bundle directory", func(t *testing.T) {
		destDir := t.TempDir()

		manifest := &model.Manifest{
			Id:              "focalboard",
			Name:            "Mattermost Boards",
			Version:         "9.3.0",
			ReleaseNotesURL: "https://github.com/mattermost/mattermost-plugin-boards/releases/tag/v9.3.0",
		}

		require.NoError(t, distManifest(manifest, destDir))

		written, err := os.ReadFile(filepath.Join(destDir, "plugin.json"))
		require.NoError(t, err)

		var got model.Manifest
		require.NoError(t, json.Unmarshal(written, &got))

		require.Equal(t, "focalboard", got.Id)
		require.Equal(t, "9.3.0", got.Version)
		require.Equal(t, manifest.ReleaseNotesURL, got.ReleaseNotesURL)
	})

	t.Run("fails when the destination directory does not exist", func(t *testing.T) {
		manifest := &model.Manifest{Id: "focalboard", Version: "9.3.0"}

		err := distManifest(manifest, filepath.Join(t.TempDir(), "does-not-exist"))
		require.Error(t, err)
	})
}

func TestFindManifestResolvesVersion(t *testing.T) {
	// The source plugin.json intentionally ships without a version field; findManifest must
	// fall back to a build-time value so the bundled manifest always carries a version.
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{
		"id": "focalboard",
		"homepage_url": "https://github.com/mattermost/mattermost-plugin-boards"
	}`), 0600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Run("uses tag at current commit when present", func(t *testing.T) {
		BuildTagCurrent = "v9.3.0"
		BuildTagLatest = "v9.2.0"
		BuildHashShort = "abc1234"
		t.Cleanup(func() { BuildTagCurrent, BuildTagLatest, BuildHashShort = "", "", "" })

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "9.3.0", manifest.Version)
	})

	t.Run("falls back to dev version when no tags exist", func(t *testing.T) {
		BuildTagCurrent = ""
		BuildTagLatest = ""
		BuildHashShort = "abc1234"
		t.Cleanup(func() { BuildHashShort = "" })

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "0.0.0+abc1234", manifest.Version)
	})
}
