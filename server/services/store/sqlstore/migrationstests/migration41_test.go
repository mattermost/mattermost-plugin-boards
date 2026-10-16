// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package migrationstests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test41AttachmentIDToFileID(t *testing.T) {
	t.Run("SQLite", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		if th.IsMySQL() || th.IsPostgres() {
			return
		}

		runMigration41Assertions(t, th)
	})

	// The bundled Mattermost core only ships Postgres migration assets, so the
	// plugin-mode helper only runs against Postgres here. Mattermost 11+ does not
	// support MySQL (min_server_version is 12.0.0), so the MySQL branch cannot run
	// in production and is not exercised by this test; it is validated manually.
	t.Run("PostgreSQL", func(t *testing.T) {
		th, tearDown := SetupPluginTestHelper(t)
		defer tearDown()

		if th.IsSQLite() {
			return
		}

		runMigration41Assertions(t, th)
	})
}

func runMigration41Assertions(t *testing.T, th *TestHelper) {
	t.Helper()

	th.f.MigrateToStep(40).
		ExecFile("./fixtures/test41AttachmentIDToFileID.sql").
		MigrateToStep(41)

	getFields := func(t *testing.T, table, id string) map[string]interface{} {
		t.Helper()

		var raw string
		query := "SELECT fields FROM " + table + " WHERE id = $1"
		if th.IsMySQL() {
			query = "SELECT fields FROM " + table + " WHERE id = ?"
		}

		err := th.f.DB().Get(&raw, query, id)
		require.NoError(t, err)

		fields := map[string]interface{}{}
		require.NoError(t, json.Unmarshal([]byte(raw), &fields))
		return fields
	}

	assertMigrated := func(t *testing.T, fields map[string]interface{}, expectedFileID string) {
		t.Helper()
		_, hasAttachment := fields["attachmentId"]
		require.False(t, hasAttachment, "attachmentId must be removed")
		require.Equal(t, expectedFileID, fields["fileId"])
	}

	t.Run("a non-empty attachmentId is moved to fileId", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-attach-only")
		assertMigrated(t, fields, "7attachonly0000000000000001.txt")
	})

	t.Run("a non-empty attachmentId takes precedence over an existing fileId", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-attach-and-file")
		assertMigrated(t, fields, "7attachwins0000000000000002.txt")
	})

	t.Run("an empty attachmentId is dropped and the existing fileId is preserved", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-empty-attach-with-file")
		assertMigrated(t, fields, "7keepfile000000000000000004.png")
	})

	t.Run("a null attachmentId is dropped and the existing fileId is preserved", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-null-attach-with-file")
		assertMigrated(t, fields, "7keepfile000000000000000005.png")
	})

	t.Run("a non-string attachmentId is dropped and the existing fileId is preserved", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-nonstring-attach-with-file")
		assertMigrated(t, fields, "7keepfile000000000000000006.png")
	})

	t.Run("an object attachmentId is dropped and the existing fileId is preserved", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-object-attach-with-file")
		assertMigrated(t, fields, "7keepobj0000000000000009.png")
	})

	t.Run("an array attachmentId is dropped and the existing fileId is preserved", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-array-attach-with-file")
		assertMigrated(t, fields, "7keeparr0000000000000010.png")
	})

	t.Run("sibling fields are preserved when attachmentId is moved to fileId", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-move-with-siblings")
		assertMigrated(t, fields, "7siblingmove00000000000008.txt")
		require.Equal(t, []interface{}{"a", "b"}, fields["contentOrder"])
		require.EqualValues(t, 1, fields["other"])
	})

	t.Run("an empty attachmentId with no fileId drops the key without adding fileId", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-empty-attach-no-file")
		_, hasAttachment := fields["attachmentId"]
		require.False(t, hasAttachment, "attachmentId must be removed")
		_, hasFile := fields["fileId"]
		require.False(t, hasFile, "fileId must not be created from an empty attachmentId")
	})

	t.Run("a block already using fileId is untouched", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-file-only")
		assertMigrated(t, fields, "7fileonly000000000000000007.png")
	})

	t.Run("a block with other fields is untouched", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-no-file-fields")
		require.Equal(t, "hello", fields["value"])
		_, hasFile := fields["fileId"]
		require.False(t, hasFile)
	})

	t.Run("a block with empty fields is untouched", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks", "block-empty-fields")
		require.Empty(t, fields)
	})

	t.Run("blocks_history: a non-empty attachmentId is moved to fileId", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks_history", "hist-attach-only")
		assertMigrated(t, fields, "7histattach000000000000001.txt")
	})

	t.Run("blocks_history: an empty attachmentId is dropped and the existing fileId is preserved", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks_history", "hist-empty-attach-with-file")
		assertMigrated(t, fields, "7histkeep00000000000000002.png")
	})

	t.Run("blocks_history: a block already using fileId is untouched", func(t *testing.T) {
		fields := getFields(t, "focalboard_blocks_history", "hist-file-only")
		assertMigrated(t, fields, "7histfile00000000000000003.png")
	})
}
