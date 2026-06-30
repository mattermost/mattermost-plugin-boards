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

// writeTempManifest writes a plugin.json into a temp dir, chdirs into it for the duration of the
// test, and resets the build-time vars afterwards.
func writeTempManifest(t *testing.T, contents string) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(contents), 0600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))

	t.Cleanup(func() {
		_ = os.Chdir(cwd)
		BuildTagCurrent, BuildTagLatest, BuildHashShort = "", "", ""
	})
}

func TestFindManifestResolvesVersion(t *testing.T) {
	const versionlessManifest = `{
		"id": "focalboard",
		"homepage_url": "https://github.com/mattermost/mattermost-plugin-boards"
	}`

	t.Run("uses the tag at the current commit over the latest tag", func(t *testing.T) {
		writeTempManifest(t, versionlessManifest)
		BuildTagCurrent = "v9.3.0"
		BuildTagLatest = "v9.2.0"
		BuildHashShort = "abc1234"

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "9.3.0", manifest.Version)
	})

	t.Run("falls back to the latest tag when no tag points at the current commit", func(t *testing.T) {
		writeTempManifest(t, versionlessManifest)
		BuildTagCurrent = ""
		BuildTagLatest = "v9.1.0"
		BuildHashShort = "abc1234"

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "9.1.0", manifest.Version)
	})

	t.Run("falls back to a dev version when no tags exist", func(t *testing.T) {
		writeTempManifest(t, versionlessManifest)
		BuildTagCurrent = ""
		BuildTagLatest = ""
		BuildHashShort = "abc1234"

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "0.0.0+abc1234", manifest.Version)
	})

	t.Run("preserves a version already present in the manifest", func(t *testing.T) {
		writeTempManifest(t, `{
			"id": "focalboard",
			"version": "1.2.3",
			"homepage_url": "https://github.com/mattermost/mattermost-plugin-boards"
		}`)
		BuildTagCurrent = "v9.3.0"
		BuildTagLatest = "v9.2.0"
		BuildHashShort = "abc1234"

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "1.2.3", manifest.Version)
	})
}

func TestFindManifestReleaseNotesURL(t *testing.T) {
	t.Run("generates a release notes URL from the latest tag when absent", func(t *testing.T) {
		writeTempManifest(t, `{
			"id": "focalboard",
			"homepage_url": "https://github.com/mattermost/mattermost-plugin-boards"
		}`)
		BuildTagLatest = "v9.3.0"

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "https://github.com/mattermost/mattermost-plugin-boards/releases/tag/v9.3.0", manifest.ReleaseNotesURL)
	})

	t.Run("does not overwrite an existing release notes URL", func(t *testing.T) {
		writeTempManifest(t, `{
			"id": "focalboard",
			"homepage_url": "https://github.com/mattermost/mattermost-plugin-boards",
			"release_notes_url": "https://example.com/custom"
		}`)
		BuildTagLatest = "v9.3.0"

		manifest, err := findManifest()
		require.NoError(t, err)
		require.Equal(t, "https://example.com/custom", manifest.ReleaseNotesURL)
	})
}

// TestBundledManifestFromSourceHasVersion is the regression test most coupled to MM-69594: the
// source plugin.json ships without a version, but the resolved manifest written into the bundle
// must carry one.
func TestBundledManifestFromSourceHasVersion(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Join(cwd, "..", "..")
	source, err := os.ReadFile(filepath.Join(repoRoot, "plugin.json"))
	require.NoError(t, err)

	var sourceManifest model.Manifest
	require.NoError(t, json.Unmarshal(source, &sourceManifest))
	require.Empty(t, sourceManifest.Version, "source plugin.json is expected to ship without a version field")

	require.NoError(t, os.Chdir(repoRoot))
	BuildTagCurrent = "v9.9.9"
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
		BuildTagCurrent = ""
	})

	resolved, err := findManifest()
	require.NoError(t, err)

	destDir := t.TempDir()
	require.NoError(t, distManifest(resolved, destDir))

	written, err := os.ReadFile(filepath.Join(destDir, "plugin.json"))
	require.NoError(t, err)

	var bundled model.Manifest
	require.NoError(t, json.Unmarshal(written, &bundled))
	require.Equal(t, "9.9.9", bundled.Version)
	require.Equal(t, sourceManifest.Id, bundled.Id)
}
