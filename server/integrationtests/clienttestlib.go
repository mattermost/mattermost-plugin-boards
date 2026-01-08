// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-boards/server/client"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/server"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/permissions/mmpermissions"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/sqlstore"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/channels/store/storetest"

	"github.com/stretchr/testify/require"
)

const (
	user1Username = "user1"
	user2Username = "user2"
	password      = "Pa$$word"
	testTeamID    = "team-id"
)

const (
	userAnon         string = "anon"
	userNoTeamMember string = "no-team-member"
	userTeamMember   string = "team-member"
	userViewer       string = "viewer"
	userCommenter    string = "commenter"
	userEditor       string = "editor"
	userAdmin        string = "admin"
	userGuest        string = "guest"
)

var (
	userAnonID         = userAnon
	userNoTeamMemberID = userNoTeamMember
	userTeamMemberID   = userTeamMember
	userViewerID       = userViewer
	userCommenterID    = userCommenter
	userEditorID       = userEditor
	userAdminID        = userAdmin
	userGuestID        = userGuest
)

type Clients struct {
	Anon         *client.Client
	NoTeamMember *client.Client
	TeamMember   *client.Client
	Viewer       *client.Client
	Commenter    *client.Client
	Editor       *client.Client
	Admin        *client.Client
	Guest        *client.Client
}

func setupClients(th *TestHelper) Clients {
	clients := Clients{
		Anon:         client.NewClient(th.Server.Config().ServerRoot, ""),
		NoTeamMember: client.NewClient(th.Server.Config().ServerRoot, ""),
		TeamMember:   client.NewClient(th.Server.Config().ServerRoot, ""),
		Viewer:       client.NewClient(th.Server.Config().ServerRoot, ""),
		Commenter:    client.NewClient(th.Server.Config().ServerRoot, ""),
		Editor:       client.NewClient(th.Server.Config().ServerRoot, ""),
		Admin:        client.NewClient(th.Server.Config().ServerRoot, ""),
		Guest:        client.NewClient(th.Server.Config().ServerRoot, ""),
	}

	clients.NoTeamMember.HTTPHeader["Mattermost-User-Id"] = userNoTeamMember
	clients.TeamMember.HTTPHeader["Mattermost-User-Id"] = userTeamMember
	clients.Viewer.HTTPHeader["Mattermost-User-Id"] = userViewer
	clients.Commenter.HTTPHeader["Mattermost-User-Id"] = userCommenter
	clients.Editor.HTTPHeader["Mattermost-User-Id"] = userEditor
	clients.Admin.HTTPHeader["Mattermost-User-Id"] = userAdmin
	clients.Guest.HTTPHeader["Mattermost-User-Id"] = userGuest

	// For plugin tests, the userID = username
	userAnonID = userAnon
	userNoTeamMemberID = userNoTeamMember
	userTeamMemberID = userTeamMember
	userViewerID = userViewer
	userCommenterID = userCommenter
	userEditorID = userEditor
	userAdminID = userAdmin
	userGuestID = userGuest

	return clients
}

type LicenseType int

const (
	LicenseNone         LicenseType = iota // 0
	LicenseProfessional                    // 1
	LicenseEnterprise                      // 2
)

type TestHelper struct {
	T       *testing.T
	Server  *server.Server
	Client  *client.Client
	Client2 *client.Client

	origEnvUnitTesting string
	sqlSettings        *mmModel.SqlSettings
	cleanupDone        bool
}

type FakePermissionPluginAPI struct{}

func (*FakePermissionPluginAPI) HasPermissionTo(userID string, permission *mmModel.Permission) bool {
	return userID == userAdmin
}

func (*FakePermissionPluginAPI) HasPermissionToTeam(userID string, teamID string, permission *mmModel.Permission) bool {
	if permission.Id == model.PermissionManageTeam.Id {
		return false
	}
	if userID == userNoTeamMember {
		return false
	}
	if teamID == "empty-team" {
		return false
	}
	return true
}

func (*FakePermissionPluginAPI) HasPermissionToChannel(userID string, channelID string, permission *mmModel.Permission) bool {
	// Accept any valid Mattermost ID format (26 characters)
	// This allows tests to use dynamically generated channel IDs
	return mmModel.IsValidId(channelID)
}

// testMutexAPI provides a no-op mutex for tests
type testMutexAPI struct{}

func (f *testMutexAPI) KVSetWithOptions(key string, value []byte, options mmModel.PluginKVSetOptions) (bool, *mmModel.AppError) {
	// Return true to simulate successful set (mutex acquired)
	return true, nil
}

func (f *testMutexAPI) LogError(msg string, keyValuePairs ...interface{}) {
	// No-op for tests
}

func getEnvWithDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

// getAvailablePort finds an available port by listening on port 0 and then closing the listener
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func getTestConfig(sqlSettings *mmModel.SqlSettings) (*config.Configuration, error) {
	// Get an available port dynamically to avoid conflicts when tests run in parallel
	port, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get available port: %w", err)
	}

	logging := `
	{
		"testing": {
			"type": "console",
			"options": {
				"out": "stdout"
			},
			"format": "plain",
			"format_options": {
				"delim": "  "
			},
			"levels": [
				{"id": 5, "name": "debug"},
				{"id": 4, "name": "info"},
				{"id": 3, "name": "warn"},
				{"id": 2, "name": "error", "stacktrace": true},
				{"id": 1, "name": "fatal", "stacktrace": true},
				{"id": 0, "name": "panic", "stacktrace": true}
			]
		}
	}`

	return &config.Configuration{
		ServerRoot:        fmt.Sprintf("http://localhost:%d", port),
		Port:              port,
		DBType:            *sqlSettings.DriverName,
		DBConfigString:    *sqlSettings.DataSource,
		DBTablePrefix:     "test_",
		WebPath:           "./pack",
		FilesDriver:       "local",
		FilesPath:         "./files",
		LoggingCfgJSON:    logging,
		SessionExpireTime: int64(30 * time.Second),
		AuthMode:          "mattermost",
	}, nil
}

// testServicesAPI provides a servicesAPI implementation for tests
type testServicesAPI struct {
	users  map[string]*model.User
	db     *sql.DB
	dbType string
}

func (t *testServicesAPI) GetUserByID(userID string) (*mmModel.User, error) {
	user := t.users[userID]
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	// Convert Boards model.User to Mattermost model.User
	return &mmModel.User{
		Id:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		CreateAt: user.CreateAt,
		UpdateAt: user.UpdateAt,
		DeleteAt: user.DeleteAt,
	}, nil
}

func (t *testServicesAPI) GetUserByEmail(email string) (*mmModel.User, error) {
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
	return nil, fmt.Errorf("user not found: %s", email)
}

func (t *testServicesAPI) GetUserByUsername(username string) (*mmModel.User, error) {
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
	return nil, fmt.Errorf("user not found: %s", username)
}

func (t *testServicesAPI) UpdateUser(user *mmModel.User) (*mmModel.User, error) {
	return user, nil
}

func (t *testServicesAPI) GetChannelByID(channelID string) (*mmModel.Channel, error) {
	// Return a mock channel for tests - this is used in migrations
	return &mmModel.Channel{
		Id:        channelID,
		Type:      mmModel.ChannelTypeDirect,
		CreatorId: "team-member", // Default creator
	}, nil
}

func (t *testServicesAPI) GetDirectChannel(userID1, userID2 string) (*mmModel.Channel, error) {
	// Return a mock direct channel
	channelID := mmModel.NewId()
	return &mmModel.Channel{
		Id:        channelID,
		Type:      mmModel.ChannelTypeDirect,
		CreatorId: userID1,
	}, nil
}

func (t *testServicesAPI) GetChannelMember(channelID string, userID string) (*mmModel.ChannelMember, error) {
	// Return NotFound error if user doesn't exist (matches Mattermost behavior)
	if _, exists := t.users[userID]; !exists {
		return nil, mmModel.NewAppError("GetChannelMember", "app.channel.get_member.missing.app_error", nil, "", http.StatusNotFound)
	}
	// Return a mock channel member
	return &mmModel.ChannelMember{
		ChannelId: channelID,
		UserId:    userID,
	}, nil
}

func (t *testServicesAPI) GetChannelsForTeamForUser(teamID string, userID string, includeDeleted bool) (mmModel.ChannelList, error) {
	// Return empty list for tests - this is used for channel search
	return mmModel.ChannelList{}, nil
}

func (t *testServicesAPI) GetLicense() *mmModel.License {
	return nil
}

func (t *testServicesAPI) GetFileInfo(fileID string) (*mmModel.FileInfo, error) {
	// Query the FileInfo table (Mattermost's table) to retrieve saved file info
	// This matches what the real Mattermost servicesAPI would do
	var query string
	if t.dbType == model.PostgresDBType {
		query = `SELECT Id, CreateAt, UpdateAt, DeleteAt, Path, ThumbnailPath, PreviewPath, Name, Extension, Size, MimeType, Width, Height, HasPreviewImage, MiniPreview, Content, RemoteId, CreatorId, PostId FROM FileInfo WHERE Id = $1`
	} else {
		query = `SELECT Id, CreateAt, UpdateAt, DeleteAt, Path, ThumbnailPath, PreviewPath, Name, Extension, Size, MimeType, Width, Height, HasPreviewImage, MiniPreview, Content, RemoteId, CreatorId, PostId FROM FileInfo WHERE Id = ?`
	}

	var fileInfo mmModel.FileInfo
	var err error
	if t.dbType == model.PostgresDBType {
		err = t.db.QueryRow(query, fileID).Scan(
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
	} else {
		err = t.db.QueryRow(query, fileID).Scan(
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
	}

	if err == sql.ErrNoRows {
		return nil, mmModel.NewAppError("GetFileInfo", "app.file_info.get.app_error", nil, "", http.StatusNotFound)
	}
	if err != nil {
		return nil, err
	}

	return &fileInfo, nil
}

func (t *testServicesAPI) CreatePost(post *mmModel.Post) (*mmModel.Post, error) {
	return nil, fmt.Errorf("not implemented")
}

func (t *testServicesAPI) EnsureBot(bot *mmModel.Bot) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (t *testServicesAPI) GetTeamMember(teamID string, userID string) (*mmModel.TeamMember, error) {
	// Return NotFound error if user doesn't exist (matches Mattermost behavior)
	if _, exists := t.users[userID]; !exists {
		return nil, mmModel.NewAppError("GetTeamMember", "app.team.get_member.missing.app_error", nil, "", http.StatusNotFound)
	}
	// Return a mock team member
	return &mmModel.TeamMember{
		TeamId: teamID,
		UserId: userID,
	}, nil
}

func (t *testServicesAPI) GetPreferencesForUser(userID string) (mmModel.Preferences, error) {
	return nil, nil
}

func (t *testServicesAPI) DeletePreferencesForUser(userID string, preferences mmModel.Preferences) error {
	return nil
}

func (t *testServicesAPI) UpdatePreferencesForUser(userID string, preferences mmModel.Preferences) error {
	return nil
}

func NewTestServerPluginMode(sqlSettings *mmModel.SqlSettings) *server.Server {
	cfg, err := getTestConfig(sqlSettings)
	if err != nil {
		panic(err)
	}
	cfg.EnablePublicSharedBoards = true

	logger, _ := mlog.NewLogger()
	if err = logger.Configure("", cfg.LoggingCfgJSON, nil); err != nil {
		panic(err)
	}

	sqlDB, err := sql.Open(cfg.DBType, cfg.DBConfigString)
	if err != nil {
		panic(fmt.Errorf("connectDatabase failed: %w", err))
	}

	err = sqlDB.Ping()
	if err != nil {
		panic(fmt.Errorf("Database Ping failed: %w", err))
	}

	// Create Mattermost tables needed for migrations (Playbooks approach)
	if err := sqlstore.SetupMattermostTablesForIntegration(sqlDB, cfg.DBType); err != nil {
		panic(fmt.Errorf("failed to setup Mattermost tables: %w", err))
	}

	// Create test users map for servicesAPI
	testUsers := map[string]*model.User{
		"no-team-member": {
			ID:       "no-team-member",
			Username: "no-team-member",
			Email:    "no-team-member@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"team-member": {
			ID:       "team-member",
			Username: "team-member",
			Email:    "team-member@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"viewer": {
			ID:       "viewer",
			Username: "viewer",
			Email:    "viewer@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"commenter": {
			ID:       "commenter",
			Username: "commenter",
			Email:    "commenter@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"editor": {
			ID:       "editor",
			Username: "editor",
			Email:    "editor@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"admin": {
			ID:       "admin",
			Username: "admin",
			Email:    "admin@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
		},
		"guest": {
			ID:       "guest",
			Username: "guest",
			Email:    "guest@sample.com",
			CreateAt: model.GetMillis(),
			UpdateAt: model.GetMillis(),
			IsGuest:  true,
		},
	}

	storeParams := sqlstore.Params{
		DBType:           cfg.DBType,
		DBPingAttempts:   cfg.DBPingAttempts,
		ConnectionString: cfg.DBConfigString,
		TablePrefix:      cfg.DBTablePrefix,
		Logger:           logger,
		DB:               sqlDB,
		ServicesAPI:      &testServicesAPI{users: testUsers, db: sqlDB, dbType: cfg.DBType},
		NewMutexFn: func(name string) (*cluster.Mutex, error) {
			// Create a mutex using a test API that does nothing
			// This allows migrations to run without actual cluster coordination
			return cluster.NewMutex(&testMutexAPI{}, name)
		},
	}

	innerStore, err := sqlstore.New(storeParams)
	if err != nil {
		panic(err)
	}

	db := NewPluginTestStore(innerStore)

	permissionsService := mmpermissions.New(db, &FakePermissionPluginAPI{}, logger)

	params := server.Params{
		Cfg:                cfg,
		DBStore:            db,
		Logger:             logger,
		PermissionsService: permissionsService,
	}

	srv, err := server.New(params)
	if err != nil {
		panic(err)
	}

	return srv
}

func SetupTestHelper(t *testing.T) *TestHelper {
	return SetupTestHelperPluginMode(t)
}

func SetupTestHelperPluginMode(t *testing.T) *TestHelper {
	origUnitTesting := os.Getenv("FOCALBOARD_UNIT_TESTING")
	os.Setenv("FOCALBOARD_UNIT_TESTING", "1")

	// Create sqlSettings for test database
	driverName := getEnvWithDefault("TEST_DATABASE_DRIVERNAME", "postgres")
	sqlSettings := storetest.MakeSqlSettings(driverName, false)

	th := &TestHelper{
		T:                  t,
		origEnvUnitTesting: origUnitTesting,
		sqlSettings:        sqlSettings,
		cleanupDone:        false,
	}

	// Ensure cleanup runs even if test panics
	t.Cleanup(func() {
		if th.sqlSettings != nil && !th.cleanupDone {
			// Use recover to handle errors gracefully (e.g., database already dropped)
			defer func() {
				if r := recover(); r != nil {
					// Ignore panics from CleanupSqlSettings - database may already be cleaned up
					// This prevents double-cleanup issues when both TearDown() and t.Cleanup() run
				}
			}()
			storetest.CleanupSqlSettings(th.sqlSettings)
			th.cleanupDone = true
		}
	})

	th.Server = NewTestServerPluginMode(sqlSettings)
	th.Start()
	return th
}

// Start starts the test server and ensures that it's correctly
// responding to requests before returning.
func (th *TestHelper) Start() *TestHelper {
	go func() {
		if err := th.Server.Start(); err != nil {
			panic(err)
		}
	}()

	for {
		URL := th.Server.Config().ServerRoot
		th.Server.Logger().Info("Polling server", mlog.String("url", URL))
		resp, err := http.Get(URL) //nolint:gosec
		if err != nil {
			th.Server.Logger().Error("Polling failed", mlog.Err(err))
			time.Sleep(100 * time.Millisecond)
			continue
		}
		resp.Body.Close()

		// Currently returns 404
		// if resp.StatusCode != http.StatusOK {
		// 	th.Server.Logger().Error("Not OK", mlog.Int("statusCode", resp.StatusCode))
		// 	continue
		// }

		// Reached this point: server is up and running!
		th.Server.Logger().Info("Server ping OK", mlog.Int("statusCode", resp.StatusCode))

		break
	}

	return th
}

// InitBasic starts the test server and initializes the clients of the
// helper, registering them and logging them into the system.
// For plugin mode, this initializes clients using setupClients.
// Note: SetupTestHelperPluginMode already calls Start(), so we don't need to call it again here.
func (th *TestHelper) InitBasic() *TestHelper {
	// Initialize clients for plugin mode
	clients := setupClients(th)
	th.Client = clients.TeamMember
	th.Client2 = clients.Viewer

	return th
}

var ErrRegisterFail = errors.New("register failed")

func (th *TestHelper) TearDown() {
	os.Setenv("FOCALBOARD_UNIT_TESTING", th.origEnvUnitTesting)

	logger := th.Server.Logger()

	if l, ok := logger.(*mlog.Logger); ok {
		defer func() { _ = l.Shutdown() }()
	}

	err := th.Server.Shutdown()
	if err != nil {
		panic(err)
	}

	os.RemoveAll(th.Server.Config().FilesPath)

	// Cleanup database using storetest.CleanupSqlSettings
	// This handles both SQLite files and PostgreSQL/MySQL databases
	// Note: t.Cleanup() will also try to clean up, so we mark it as done to prevent double cleanup
	if th.sqlSettings != nil && !th.cleanupDone {
		storetest.CleanupSqlSettings(th.sqlSettings)
		th.cleanupDone = true
	}

	// Fallback: Try to remove SQLite file if it exists (for backward compatibility)
	if err := os.Remove(th.Server.Config().DBConfigString); err == nil {
		logger.Debug("Removed test database", mlog.String("file", th.Server.Config().DBConfigString))
	}
}

func (th *TestHelper) Me(client *client.Client) *model.User {
	user, resp := client.GetMe()
	th.CheckOK(resp)
	require.NotNil(th.T, user)
	return user
}

func (th *TestHelper) CreateBoard(teamID string, boardType model.BoardType) *model.Board {
	newBoard := &model.Board{
		TeamID: teamID,
		Type:   boardType,
	}
	board, resp := th.Client.CreateBoard(newBoard)
	th.CheckOK(resp)
	return board
}

func (th *TestHelper) CreateBoards(teamID string, boardType model.BoardType, count int) []*model.Board {
	boards := make([]*model.Board, 0, count)

	for i := 0; i < count; i++ {
		board := th.CreateBoard(teamID, boardType)
		boards = append(boards, board)
	}
	return boards
}

func (th *TestHelper) CreateCategory(category model.Category) *model.Category {
	cat, resp := th.Client.CreateCategory(category)
	th.CheckOK(resp)
	return cat
}

func (th *TestHelper) UpdateCategoryBoard(teamID, categoryID, boardID string) {
	response := th.Client.UpdateCategoryBoard(teamID, categoryID, boardID)
	th.CheckOK(response)
}

func (th *TestHelper) CreateBoardAndCards(teamdID string, boardType model.BoardType, numCards int) (*model.Board, []*model.Card) {
	board := th.CreateBoard(teamdID, boardType)
	cards := make([]*model.Card, 0, numCards)
	for i := 0; i < numCards; i++ {
		card := &model.Card{
			Title:        fmt.Sprintf("test card %d", i+1),
			ContentOrder: []string{utils.NewID(utils.IDTypeBlock), utils.NewID(utils.IDTypeBlock), utils.NewID(utils.IDTypeBlock)},
			Icon:         "😱",
			Properties:   th.MakeCardProps(5),
		}
		newCard, resp := th.Client.CreateCard(board.ID, card, true)
		th.CheckOK(resp)
		cards = append(cards, newCard)
	}
	return board, cards
}

func (th *TestHelper) MakeCardProps(count int) map[string]any {
	props := make(map[string]any)
	for i := 0; i < count; i++ {
		props[utils.NewID(utils.IDTypeBlock)] = utils.NewID(utils.IDTypeBlock)
	}
	return props
}

func (th *TestHelper) GetUserCategoryBoards(teamID string) []model.CategoryBoards {
	categoryBoards, response := th.Client.GetUserCategoryBoards(teamID)
	th.CheckOK(response)
	return categoryBoards
}

func (th *TestHelper) DeleteCategory(teamID, categoryID string) {
	response := th.Client.DeleteCategory(teamID, categoryID)
	th.CheckOK(response)
}

func (th *TestHelper) GetUser1() *model.User {
	return th.Me(th.Client)
}

func (th *TestHelper) GetUser2() *model.User {
	return th.Me(th.Client2)
}

func (th *TestHelper) CheckOK(r *client.Response) {
	require.Equal(th.T, http.StatusOK, r.StatusCode)
	require.NoError(th.T, r.Error)
}

func (th *TestHelper) CheckBadRequest(r *client.Response) {
	require.Equal(th.T, http.StatusBadRequest, r.StatusCode)
	require.Error(th.T, r.Error)
}

func (th *TestHelper) CheckNotFound(r *client.Response) {
	require.Equal(th.T, http.StatusNotFound, r.StatusCode)
	require.Error(th.T, r.Error)
}

func (th *TestHelper) CheckUnauthorized(r *client.Response) {
	require.Equal(th.T, http.StatusUnauthorized, r.StatusCode)
	require.Error(th.T, r.Error)
}

func (th *TestHelper) CheckForbidden(r *client.Response) {
	require.Equal(th.T, http.StatusForbidden, r.StatusCode)
	require.Error(th.T, r.Error)
}

func (th *TestHelper) CheckRequestEntityTooLarge(r *client.Response) {
	require.Equal(th.T, http.StatusRequestEntityTooLarge, r.StatusCode)
	require.Error(th.T, r.Error)
}

func (th *TestHelper) CheckNotImplemented(r *client.Response) {
	require.Equal(th.T, http.StatusNotImplemented, r.StatusCode)
	require.Error(th.T, r.Error)
}

// AddUserToTeamMembers inserts a user into the TeamMembers table for the given team.
// This is useful for tests that need to ensure a user is a team member so that
// SearchBoardsForTeam can find public boards associated with that team.
func (th *TestHelper) AddUserToTeamMembers(teamID, userID string) error {
	dbType := th.Server.Config().DBType
	connectionString := th.Server.Config().DBConfigString
	db, err := sql.Open(dbType, connectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	if dbType == model.PostgresDBType {
		// PostgreSQL uses lowercase table and column names
		// Check if record exists first to avoid constraint errors
		var count int
		checkSQL := `SELECT COUNT(*) FROM teammembers WHERE teamid = $1 AND userid = $2`
		err := db.QueryRow(checkSQL, teamID, userID).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			insertTeamMemberSQL := `INSERT INTO teammembers (teamid, userid, roles, deleteat) VALUES ($1, $2, $3, $4)`
			_, err = db.Exec(insertTeamMemberSQL, teamID, userID, "member", 0)
			if err != nil {
				return err
			}
		}
	} else {
		// MySQL uses camel case
		insertTeamMemberSQL := `INSERT IGNORE INTO TeamMembers (TeamId, UserId, Roles, DeleteAt) VALUES (?, ?, ?, ?)`
		_, err = db.Exec(insertTeamMemberSQL, teamID, userID, "member", 0)
		if err != nil {
			return err
		}
	}
	return nil
}
