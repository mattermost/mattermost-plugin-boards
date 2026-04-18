// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
)

// SetupMattermostTablesForIntegration creates Mattermost tables for integration tests
// without requiring testing.T (for use in non-test contexts like NewTestServerPluginMode).
//
// IMPORTANT: Schema Versioning and Maintenance
// =============================================
// The table schemas defined in this file are based on Mattermost server schemas and are
// intentionally minimal - they only include columns required for plugin migrations and tests.
//
// Schema Source: These schemas are derived from Mattermost server table definitions, but
// are NOT kept in sync with Mattermost's schema evolution. They represent a snapshot at
// the time of implementation.
//
// Maintenance Strategy:
//  1. When Mattermost adds new columns that our code uses, update the relevant table
//     definition in this file and support_for_test.go
//  2. When tests fail due to missing columns, add the required columns to the table definitions
func SetupMattermostTablesForIntegration(db *sql.DB) error {
	if err := setupChannelsTableForIntegration(db); err != nil {
		return err
	}
	if err := setupChannelMembersTableForIntegration(db); err != nil {
		return err
	}
	if err := setupTeamMembersTableForIntegration(db); err != nil {
		return err
	}
	if err := setupTeamsTableForIntegration(db); err != nil {
		return err
	}
	if err := setupUsersTableForIntegration(db); err != nil {
		return err
	}
	if err := setupPreferencesTableForIntegration(db); err != nil {
		return err
	}
	if err := setupBotsTableForIntegration(db); err != nil {
		return err
	}
	if err := setupFileInfoTableForIntegration(db); err != nil {
		return err
	}
	if err := setupSessionsTableForIntegration(db); err != nil {
		return err
	}
	return nil
}

func setupChannelsTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	return err
}

func setupChannelMembersTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	return err
}

func setupTeamMembersTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS teammembers (
			teamid character varying(26) NOT NULL,
			userid character varying(26) NOT NULL,
			roles character varying(64),
			deleteat bigint,
			schemeuser boolean,
			schemeadmin boolean,
			PRIMARY KEY (teamid, userid)
		);
	`)
	return err
}

func setupTeamsTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	return err
}

func setupUsersTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	return err
}

func setupPreferencesTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS preferences (
			userid character varying(26) NOT NULL,
			category character varying(32) NOT NULL,
			name character varying(32) NOT NULL,
			value character varying(2000),
			PRIMARY KEY (userid, category, name)
		);
	`)
	return err
}

func setupBotsTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bots (
			userid character varying(26) NOT NULL PRIMARY KEY,
			description character varying(1024),
			ownerid character varying(190)
		);
	`)
	return err
}

func setupFileInfoTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS fileinfo (
			id character varying(26) NOT NULL,
			creatorid character varying(26),
			postid character varying(26),
			createat bigint,
			updateat bigint,
			deleteat bigint,
			path character varying(512),
			thumbnailpath character varying(512),
			previewpath character varying(512),
			name character varying(256),
			extension character varying(64),
			size bigint,
			mimetype character varying(256),
			width integer,
			height integer,
			haspreviewimage boolean,
			minipreview text,
			content text,
			remoteid character varying(26),
			archived boolean,
			PRIMARY KEY (id)
		);
	`)
	return err
}

func setupSessionsTableForIntegration(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id character varying(26) NOT NULL PRIMARY KEY,
			token character varying(64),
			createat bigint,
			expiresat bigint,
			lastactivityat bigint,
			userid character varying(26),
			roles character varying(64),
			isoauth boolean,
			props text,
			deviceid character varying(512)
		);
	`)
	return err
}
