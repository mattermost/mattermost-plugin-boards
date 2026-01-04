// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/stretchr/testify/require"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/channels/store/storetest"
)

// testMutexAPI provides a no-op mutex for tests
type testMutexAPI struct{}

func (f *testMutexAPI) KVSetWithOptions(key string, value []byte, options mmModel.PluginKVSetOptions) (bool, *mmModel.AppError) {
	// Return true to simulate successful set (mutex acquired)
	return true, nil
}

func (f *testMutexAPI) LogError(msg string, keyValuePairs ...interface{}) {
	// No-op for tests
}

// testServicesAPIForUnitTests provides a minimal servicesAPI implementation for unit tests
type testServicesAPIForUnitTests struct {
	users  map[string]*model.User
	db     *sql.DB
	dbType string
}

func (t *testServicesAPIForUnitTests) GetUserByID(userID string) (*mmModel.User, error) {
	user := t.users[userID]
	if user != nil {
		return &mmModel.User{
			Id:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			CreateAt: user.CreateAt,
			UpdateAt: user.UpdateAt,
			DeleteAt: user.DeleteAt,
		}, nil
	}

	// Try to query the database if user not in map
	if t.db != nil {
		var dbUser struct {
			ID       string
			Username string
			Email    string
			CreateAt int64
			UpdateAt int64
			DeleteAt int64
		}

		var query string
		var args []interface{}
		if t.dbType == model.PostgresDBType {
			query = "SELECT id, username, email, createat, updateat, deleteat FROM users WHERE id = $1"
			args = []interface{}{userID}
		} else {
			query = "SELECT Id, Username, Email, CreateAt, UpdateAt, DeleteAt FROM Users WHERE Id = ?"
			args = []interface{}{userID}
		}

		err := t.db.QueryRow(query, args...).Scan(
			&dbUser.ID, &dbUser.Username, &dbUser.Email, &dbUser.CreateAt, &dbUser.UpdateAt, &dbUser.DeleteAt)
		if err == nil {
			return &mmModel.User{
				Id:       dbUser.ID,
				Username: dbUser.Username,
				Email:    dbUser.Email,
				CreateAt: dbUser.CreateAt,
				UpdateAt: dbUser.UpdateAt,
				DeleteAt: dbUser.DeleteAt,
			}, nil
		}
		// User not found in database, return ErrNotFound
		return nil, model.NewErrNotFound("user ID=" + userID)
	}

	return nil, model.NewErrNotFound("user ID=" + userID)
}

func (t *testServicesAPIForUnitTests) GetUserByEmail(email string) (*mmModel.User, error) {
	for _, user := range t.users {
		if user.Email == email {
			return &mmModel.User{
				Id:       user.ID,
				Username: user.Username,
				Email:    user.Email,
				CreateAt: user.CreateAt,
				UpdateAt: user.UpdateAt,
				DeleteAt: user.DeleteAt,
			}, nil
		}
	}

	// Try to query the database if user not in map
	if t.db != nil {
		var dbUser struct {
			ID       string
			Username string
			Email    string
			CreateAt int64
			UpdateAt int64
			DeleteAt int64
		}

		var query string
		var args []interface{}
		if t.dbType == model.PostgresDBType {
			query = "SELECT id, username, email, createat, updateat, deleteat FROM users WHERE email = $1"
			args = []interface{}{email}
		} else {
			query = "SELECT Id, Username, Email, CreateAt, UpdateAt, DeleteAt FROM Users WHERE Email = ?"
			args = []interface{}{email}
		}

		err := t.db.QueryRow(query, args...).Scan(
			&dbUser.ID, &dbUser.Username, &dbUser.Email, &dbUser.CreateAt, &dbUser.UpdateAt, &dbUser.DeleteAt)
		if err == nil {
			return &mmModel.User{
				Id:       dbUser.ID,
				Username: dbUser.Username,
				Email:    dbUser.Email,
				CreateAt: dbUser.CreateAt,
				UpdateAt: dbUser.UpdateAt,
				DeleteAt: dbUser.DeleteAt,
			}, nil
		}
	}

	return nil, model.NewErrNotFound("user email=" + email)
}

func (t *testServicesAPIForUnitTests) GetUserByUsername(username string) (*mmModel.User, error) {
	for _, user := range t.users {
		if user.Username == username {
			return &mmModel.User{
				Id:       user.ID,
				Username: user.Username,
				Email:    user.Email,
				CreateAt: user.CreateAt,
				UpdateAt: user.UpdateAt,
				DeleteAt: user.DeleteAt,
			}, nil
		}
	}

	// Try to query the database if user not in map
	if t.db != nil {
		var dbUser struct {
			ID       string
			Username string
			Email    string
			CreateAt int64
			UpdateAt int64
			DeleteAt int64
		}

		var query string
		var args []interface{}
		if t.dbType == model.PostgresDBType {
			query = "SELECT id, username, email, createat, updateat, deleteat FROM users WHERE username = $1"
			args = []interface{}{username}
		} else {
			query = "SELECT Id, Username, Email, CreateAt, UpdateAt, DeleteAt FROM Users WHERE Username = ?"
			args = []interface{}{username}
		}

		err := t.db.QueryRow(query, args...).Scan(
			&dbUser.ID, &dbUser.Username, &dbUser.Email, &dbUser.CreateAt, &dbUser.UpdateAt, &dbUser.DeleteAt)
		if err == nil {
			return &mmModel.User{
				Id:       dbUser.ID,
				Username: dbUser.Username,
				Email:    dbUser.Email,
				CreateAt: dbUser.CreateAt,
				UpdateAt: dbUser.UpdateAt,
				DeleteAt: dbUser.DeleteAt,
			}, nil
		}
	}

	return nil, model.NewErrNotFound("user username=" + username)
}

func (t *testServicesAPIForUnitTests) UpdateUser(user *mmModel.User) (*mmModel.User, error) {
	return user, nil
}

func (t *testServicesAPIForUnitTests) GetChannelByID(channelID string) (*mmModel.Channel, error) {
	return &mmModel.Channel{Id: channelID, Type: mmModel.ChannelTypeDirect, CreatorId: "test-user"}, nil
}

func (t *testServicesAPIForUnitTests) GetDirectChannel(userID1, userID2 string) (*mmModel.Channel, error) {
	return &mmModel.Channel{Id: mmModel.NewId(), Type: mmModel.ChannelTypeDirect, CreatorId: userID1}, nil
}

func (t *testServicesAPIForUnitTests) GetChannelMember(channelID string, userID string) (*mmModel.ChannelMember, error) {
	if _, exists := t.users[userID]; !exists {
		return nil, mmModel.NewAppError("GetChannelMember", "app.channel.get_member.missing.app_error", nil, "", http.StatusNotFound)
	}
	return &mmModel.ChannelMember{ChannelId: channelID, UserId: userID}, nil
}

func (t *testServicesAPIForUnitTests) GetChannelsForTeamForUser(teamID string, userID string, includeDeleted bool) (mmModel.ChannelList, error) {
	return mmModel.ChannelList{}, nil
}

func (t *testServicesAPIForUnitTests) GetLicense() *mmModel.License {
	return nil
}

func (t *testServicesAPIForUnitTests) GetFileInfo(fileID string) (*mmModel.FileInfo, error) {
	// Query the FileInfo table (Mattermost's table) to retrieve saved file info
	// This matches what the real Mattermost servicesAPI would do
	query := `SELECT Id, CreateAt, UpdateAt, DeleteAt, Path, ThumbnailPath, PreviewPath, Name, Extension, Size, MimeType, Width, Height, HasPreviewImage, MiniPreview, Content, RemoteId, CreatorId, PostId FROM FileInfo WHERE Id = $1`

	var fileInfo mmModel.FileInfo
	err := t.db.QueryRow(query, fileID).Scan(
		&fileInfo.Id,
		&fileInfo.CreateAt,
		&fileInfo.UpdateAt,
		&fileInfo.DeleteAt,
		&fileInfo.Path,
		&fileInfo.ThumbnailPath,
		&fileInfo.PreviewPath,
		&fileInfo.Name,
		&fileInfo.Extension,
		&fileInfo.Size,
		&fileInfo.MimeType,
		&fileInfo.Width,
		&fileInfo.Height,
		&fileInfo.HasPreviewImage,
		&fileInfo.MiniPreview,
		&fileInfo.Content,
		&fileInfo.RemoteId,
		&fileInfo.CreatorId,
		&fileInfo.PostId,
	)

	if err == sql.ErrNoRows {
		return nil, mmModel.NewAppError("GetFileInfo", "app.file_info.get.app_error", nil, "", http.StatusNotFound)
	}
	if err != nil {
		return nil, err
	}

	return &fileInfo, nil
}

func (t *testServicesAPIForUnitTests) CreatePost(post *mmModel.Post) (*mmModel.Post, error) {
	return nil, fmt.Errorf("not implemented")
}

func (t *testServicesAPIForUnitTests) EnsureBot(bot *mmModel.Bot) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (t *testServicesAPIForUnitTests) GetTeamMember(teamID string, userID string) (*mmModel.TeamMember, error) {
	if _, exists := t.users[userID]; !exists {
		return nil, mmModel.NewAppError("GetTeamMember", "app.team.get_member.missing.app_error", nil, "", http.StatusNotFound)
	}
	return &mmModel.TeamMember{TeamId: teamID, UserId: userID}, nil
}

func (t *testServicesAPIForUnitTests) GetPreferencesForUser(userID string) (mmModel.Preferences, error) {
	return nil, nil
}

func (t *testServicesAPIForUnitTests) DeletePreferencesForUser(userID string, preferences mmModel.Preferences) error {
	return nil
}

func (t *testServicesAPIForUnitTests) UpdatePreferencesForUser(userID string, preferences mmModel.Preferences) error {
	return nil
}

func SetupTests(t *testing.T) (store.Store, func()) {
	origUnitTesting := os.Getenv("FOCALBOARD_UNIT_TESTING")
	os.Setenv("FOCALBOARD_UNIT_TESTING", "1")

	// Skip SQLite in plugin mode (not supported)
	driverName := strings.TrimSpace(os.Getenv("TEST_DATABASE_DRIVERNAME"))
	if driverName == "" || driverName == model.SqliteDBType {
		driverName = "postgres" // Default to postgres
	}

	// Use storetest.MakeSqlSettings to get proper test database setup (handles SSL, etc.)
	sqlSettings := storetest.MakeSqlSettings(driverName, false)
	dbType := *sqlSettings.DriverName
	connectionString := *sqlSettings.DataSource

	logger, _ := mlog.NewLogger()

	sqlDB, err := sql.Open(dbType, connectionString)
	require.NoError(t, err)
	err = sqlDB.Ping()
	require.NoError(t, err)

	// Create Mattermost tables needed for migrations (Playbooks approach)
	setupMattermostTables(t, sqlDB, dbType)

	// Create a minimal testServicesAPI for unit tests
	// Include common test user IDs used by storetests
	testUsers := map[string]*model.User{
		"test-user": {
			ID:       "test-user",
			Username: "test-user",
			Email:    "test-user@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"user-id": { // Used by storetests
			ID:       "user-id",
			Username: "user-id",
			Email:    "user-id@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
	}
	testServicesAPI := &testServicesAPIForUnitTests{users: testUsers, db: sqlDB, dbType: dbType}

	storeParams := Params{
		DBType:           dbType,
		ConnectionString: connectionString,
		DBPingAttempts:   5,
		TablePrefix:      "test_",
		Logger:           logger,
		DB:               sqlDB,
		ServicesAPI:      testServicesAPI,
		NewMutexFn: func(name string) (*cluster.Mutex, error) {
			return cluster.NewMutex(&testMutexAPI{}, name)
		},
	}
	store, err := New(storeParams)
	require.NoError(t, err)

	tearDown := func() {
		defer func() { _ = logger.Shutdown() }()
		err = store.Shutdown()
		require.Nil(t, err)
		if err = os.Remove(connectionString); err == nil {
			logger.Debug("Removed test database", mlog.String("file", connectionString))
		}
		os.Setenv("FOCALBOARD_UNIT_TESTING", origUnitTesting)
	}

	return store, tearDown
}
