// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package assets

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// Some storage backends (e.g. Azure Blob) cannot store empty files, so the
// default templates must not contain any.
func TestDefaultTemplatesArchiveHasNoEmptyFiles(t *testing.T) {
	zr, err := zip.NewReader(bytes.NewReader(DefaultTemplatesArchive), int64(len(DefaultTemplatesArchive)))
	require.NoError(t, err)
	require.NotEmpty(t, zr.File)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		require.NotZero(t, f.UncompressedSize64, "empty file in templates archive: %s", f.Name)
	}
}
