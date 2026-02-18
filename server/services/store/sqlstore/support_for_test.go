// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/stretchr/testify/require"
)

// setupMattermostTables creates all Mattermost tables needed for migrations and tests.
// This is similar to the Playbooks plugin approach but supports both MySQL and PostgreSQL.
//
// Schema Maintenance Note:
// The table schemas here are minimal snapshots of Mattermost's tables, containing only
// columns needed for plugin operations. When Mattermost's schema evolves and our code
// requires new columns, update the relevant setup*Table functions accordingly.
// See mattermost_tables.go for more details on the maintenance strategy.
func setupMattermostTables(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	setupChannelsTable(t, db, dbType)
	setupChannelMembersTable(t, db, dbType)
	setupTeamMembersTable(t, db, dbType)
	setupTeamsTable(t, db, dbType)
	setupUsersTable(t, db, dbType)
	setupPreferencesTable(t, db, dbType)
	setupBotsTable(t, db, dbType)
	setupFileInfoTable(t, db, dbType)
	setupSessionsTable(t, db, dbType)
}

func setupChannelsTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS channels (
				id character varying(26) NOT NULL,
				createat bigint,
				updateat bigint,
				deleteat bigint,
				teamid character varying(26),
				type character varying(1),
				displayname character varying(64),
				name character varying(64),
				header character varying(1024),
				purpose character varying(250),
				lastpostat bigint,
				totalmsgcount bigint,
				extraupdateat bigint,
				creatorid character varying(26),
				schemeid character varying(26),
				PRIMARY KEY (id)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Channels (
				Id VARCHAR(26) NOT NULL,
				CreateAt BIGINT,
				UpdateAt BIGINT,
				DeleteAt BIGINT,
				TeamId VARCHAR(26),
				Type VARCHAR(1),
				DisplayName VARCHAR(64),
				Name VARCHAR(64),
				Header VARCHAR(1024),
				Purpose VARCHAR(250),
				LastPostAt BIGINT,
				TotalMsgCount BIGINT,
				ExtraUpdateAt BIGINT,
				CreatorId VARCHAR(26),
				SchemeId VARCHAR(26),
				PRIMARY KEY (Id)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupChannelMembersTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS channelmembers (
				channelid character varying(26) NOT NULL,
				userid character varying(26) NOT NULL,
				roles character varying(64),
				lastviewedat bigint,
				msgcount bigint,
				mentioncount bigint,
				notifyprops character varying(2000),
				lastupdateat bigint,
				schemeuser boolean,
				schemeadmin boolean,
				PRIMARY KEY (channelid, userid)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS ChannelMembers (
				ChannelId VARCHAR(26) NOT NULL,
				UserId VARCHAR(26) NOT NULL,
				Roles VARCHAR(64),
				LastViewedAt BIGINT,
				MsgCount BIGINT,
				MentionCount BIGINT,
				NotifyProps VARCHAR(2000),
				LastUpdateAt BIGINT,
				SchemeUser BOOLEAN,
				SchemeAdmin BOOLEAN,
				PRIMARY KEY (ChannelId, UserId)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupTeamMembersTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS teammembers (
				teamid character varying(26) NOT NULL,
				userid character varying(26) NOT NULL,
				roles character varying(64),
				deleteat bigint,
				schemeuser boolean,
				schemeadmin boolean
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS TeamMembers (
				TeamId VARCHAR(26) NOT NULL,
				UserId VARCHAR(26) NOT NULL,
				Roles VARCHAR(64),
				DeleteAt BIGINT,
				SchemeUser BOOLEAN,
				SchemeAdmin BOOLEAN
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupTeamsTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS teams (
				id character varying(26) NOT NULL,
				createat bigint,
				updateat bigint,
				deleteat bigint,
				displayname character varying(64),
				name character varying(64),
				description character varying(255),
				email character varying(128),
				type character varying(255),
				companyname character varying(64),
				alloweddomains character varying(1000),
				inviteid character varying(32),
				schemeid character varying(26),
				allowopeninvite boolean,
				lastteamiconupdate bigint,
				groupconstrained boolean,
				PRIMARY KEY (id)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Teams (
				Id VARCHAR(26) NOT NULL,
				CreateAt BIGINT,
				UpdateAt BIGINT,
				DeleteAt BIGINT,
				DisplayName VARCHAR(64),
				Name VARCHAR(64),
				Description VARCHAR(255),
				Email VARCHAR(128),
				Type VARCHAR(255),
				CompanyName VARCHAR(64),
				AllowedDomains VARCHAR(1000),
				InviteId VARCHAR(32),
				SchemeId VARCHAR(26),
				AllowOpenInvite BOOLEAN,
				LastTeamIconUpdate BIGINT,
				GroupConstrained BOOLEAN,
				PRIMARY KEY (Id)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupUsersTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS users (
				id character varying(26) NOT NULL,
				createat bigint,
				updateat bigint,
				deleteat bigint,
				username character varying(64),
				password character varying(128),
				authdata character varying(128),
				authservice character varying(32),
				email character varying(128),
				emailverified boolean,
				nickname character varying(64),
				firstname character varying(64),
				lastname character varying(64),
				position character varying(128),
				roles character varying(256),
				allowmarketing boolean,
				props character varying(4000),
				notifyprops character varying(2000),
				lastpasswordupdate bigint,
				lastpictureupdate bigint,
				failedattempts integer,
				locale character varying(5),
				timezone character varying(256),
				mfaactive boolean,
				mfasecret character varying(128),
				PRIMARY KEY (id)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Users (
				Id VARCHAR(26) NOT NULL,
				CreateAt BIGINT,
				UpdateAt BIGINT,
				DeleteAt BIGINT,
				Username VARCHAR(64),
				Password VARCHAR(128),
				AuthData VARCHAR(128),
				AuthService VARCHAR(32),
				Email VARCHAR(128),
				EmailVerified BOOLEAN,
				Nickname VARCHAR(64),
				FirstName VARCHAR(64),
				LastName VARCHAR(64),
				Position VARCHAR(128),
				Roles VARCHAR(256),
				AllowMarketing BOOLEAN,
				Props VARCHAR(4000),
				NotifyProps VARCHAR(2000),
				LastPasswordUpdate BIGINT,
				LastPictureUpdate BIGINT,
				FailedAttempts INTEGER,
				Locale VARCHAR(5),
				Timezone VARCHAR(256),
				MfaActive BOOLEAN,
				MfaSecret VARCHAR(128),
				PRIMARY KEY (Id)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupPreferencesTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS preferences (
				userid character varying(26) NOT NULL,
				category character varying(32) NOT NULL,
				name character varying(32) NOT NULL,
				value character varying(2000),
				PRIMARY KEY (userid, category, name)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Preferences (
				UserId VARCHAR(26) NOT NULL,
				Category VARCHAR(32) NOT NULL,
				Name VARCHAR(32) NOT NULL,
				Value VARCHAR(2000),
				PRIMARY KEY (UserId, Category, Name)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupBotsTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS bots (
				userid character varying(26) NOT NULL PRIMARY KEY,
				description character varying(1024),
				ownerid character varying(190)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Bots (
				UserId VARCHAR(26) NOT NULL PRIMARY KEY,
				Description VARCHAR(1024),
				OwnerId VARCHAR(190)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupFileInfoTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS FileInfo (
				Id VARCHAR(26) NOT NULL,
				CreatorId VARCHAR(26),
				PostId VARCHAR(26),
				CreateAt BIGINT,
				UpdateAt BIGINT,
				DeleteAt BIGINT,
				Path VARCHAR(512),
				ThumbnailPath VARCHAR(512),
				PreviewPath VARCHAR(512),
				Name VARCHAR(256),
				Extension VARCHAR(64),
				Size BIGINT,
				MimeType VARCHAR(256),
				Width INTEGER,
				Height INTEGER,
				HasPreviewImage BOOLEAN,
				MiniPreview TEXT,
				Content TEXT,
				RemoteId VARCHAR(26),
				Archived BOOLEAN,
				PRIMARY KEY (Id)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS FileInfo (
				Id VARCHAR(26) NOT NULL,
				CreatorId VARCHAR(26),
				PostId VARCHAR(26),
				CreateAt BIGINT,
				UpdateAt BIGINT,
				DeleteAt BIGINT,
				Path VARCHAR(512),
				ThumbnailPath VARCHAR(512),
				PreviewPath VARCHAR(512),
				Name VARCHAR(256),
				Extension VARCHAR(64),
				Size BIGINT,
				MimeType VARCHAR(256),
				Width INTEGER,
				Height INTEGER,
				HasPreviewImage BOOLEAN,
				MiniPreview TEXT,
				Content TEXT,
				RemoteId VARCHAR(26),
				Archived BOOLEAN,
				PRIMARY KEY (Id)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
		return
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}

func setupSessionsTable(t *testing.T, db *sql.DB, dbType string) {
	t.Helper()

	var createTableSQL string
	switch dbType {
	case model.PostgresDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Sessions (
				Id character varying(26) NOT NULL PRIMARY KEY,
				Token character varying(64),
				CreateAt bigint,
				ExpiresAt bigint,
				LastActivityAt bigint,
				UserId character varying(26),
				Roles character varying(64),
				IsOAuth boolean,
				Props text,
				DeviceId character varying(512)
			);
		`
	case model.MysqlDBType:
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS Sessions (
				Id VARCHAR(26) NOT NULL PRIMARY KEY,
				Token VARCHAR(64),
				CreateAt BIGINT,
				ExpiresAt BIGINT,
				LastActivityAt BIGINT,
				UserId VARCHAR(26),
				Roles VARCHAR(64),
				IsOAuth BOOLEAN,
				Props TEXT,
				DeviceId VARCHAR(512)
			) DEFAULT CHARACTER SET utf8mb4;
		`
	default:
		t.Fatalf("unsupported database driver: %s", dbType)
		return
	}

	_, err := db.Exec(createTableSQL)
	require.NoError(t, err)
}
