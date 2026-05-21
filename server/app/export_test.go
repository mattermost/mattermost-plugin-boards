// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeArchiveFilename(t *testing.T) {
	t.Run("valid simple filename passes unchanged", func(t *testing.T) {
		result, err := sanitizeArchiveFilename("abc123.png")
		require.NoError(t, err)
		assert.Equal(t, "abc123.png", result)
	})

	t.Run("valid filename with no extension passes unchanged", func(t *testing.T) {
		result, err := sanitizeArchiveFilename("myfile")
		require.NoError(t, err)
		assert.Equal(t, "myfile", result)
	})

	t.Run("filename with subdirectory returns base name only", func(t *testing.T) {
		result, err := sanitizeArchiveFilename("subdir/abc123.png")
		require.NoError(t, err)
		assert.Equal(t, "abc123.png", result)
	})

	t.Run("path traversal with .. is rejected", func(t *testing.T) {
		_, err := sanitizeArchiveFilename("../../etc/passwd")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidFilenamePathTraversal))
	})

	t.Run("traversal embedded in path is rejected", func(t *testing.T) {
		_, err := sanitizeArchiveFilename("subdir/../../etc/passwd")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidFilenamePathTraversal))
	})

	t.Run("filename that is just .. is rejected", func(t *testing.T) {
		_, err := sanitizeArchiveFilename("..")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidFilenamePathTraversal))
	})

	t.Run("root path is rejected", func(t *testing.T) {
		_, err := sanitizeArchiveFilename("/")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidFilename))
	})

	t.Run("dot path is rejected", func(t *testing.T) {
		_, err := sanitizeArchiveFilename(".")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidFilename))
	})

	t.Run("absolute path returns base name only", func(t *testing.T) {
		result, err := sanitizeArchiveFilename("/etc/image.png")
		require.NoError(t, err)
		assert.Equal(t, "image.png", result)
	})
}
