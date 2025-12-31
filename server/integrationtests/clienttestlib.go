// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"database/sql"
	"errors"
	"fmt"
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
	return channelID == "valid-channel-id" || channelID == "valid-channel-id-2"
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

func getTestConfig() (*config.Configuration, error) {
	driverName := getEnvWithDefault("TEST_DATABASE_DRIVERNAME", "postgres")
	sqlSettings := storetest.MakeSqlSettings(driverName, false)

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
		ServerRoot:        "http://localhost:8888",
		Port:              8888,
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

func NewTestServerPluginMode() *server.Server {
	cfg, err := getTestConfig()
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

	storeParams := sqlstore.Params{
		DBType:           cfg.DBType,
		DBPingAttempts:   cfg.DBPingAttempts,
		ConnectionString: cfg.DBConfigString,
		TablePrefix:      cfg.DBTablePrefix,
		Logger:           logger,
		DB:               sqlDB,
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

	th := &TestHelper{
		T:                  t,
		origEnvUnitTesting: origUnitTesting,
	}

	th.Server = NewTestServerPluginMode()
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
func (th *TestHelper) InitBasic() *TestHelper {
	th.Start()

	// get token
	team, resp := th.Client.GetTeam(model.GlobalTeamID)
	th.CheckOK(resp)
	require.NotNil(th.T, team)
	require.NotNil(th.T, team.SignupToken)

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
