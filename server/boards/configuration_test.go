// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package boards

import (
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-boards/server/integrationtests"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/server"
	"github.com/mattermost/mattermost-plugin-boards/server/ws"

	mockservicesapi "github.com/mattermost/mattermost-plugin-boards/server/model/mocks"

	serverModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/channels/store/storetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errPatchFailed  = errors.New("patch failed")
	errEnsureFailed = errors.New("ensure failed")
	errGetFailed    = errors.New("get failed")
)

type TestHelper struct {
	Server *server.Server
}

func SetupTestHelper(t *testing.T) (*TestHelper, func()) {
	th := &TestHelper{}
	th.Server = newTestServer()

	err := th.Server.Start()
	require.NoError(t, err, "Server start should not error")

	tearDown := func() {
		err := th.Server.Shutdown()
		require.NoError(t, err, "Server shutdown should not error")
	}
	return th, tearDown
}

func getEnvWithDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

func newTestServer() *server.Server {
	driverName := getEnvWithDefault("TEST_DATABASE_DRIVERNAME", "postgres")
	sqlSettings := storetest.MakeSqlSettings(driverName)
	return integrationtests.NewTestServerPluginMode(sqlSettings)
}
func TestConfigurationNullConfiguration(t *testing.T) {
	boardsApp := &BoardsApp{}
	assert.NotNil(t, boardsApp.getConfiguration())
}

func TestOnConfigurationChange(t *testing.T) {
	t.Cleanup(func() { model.SetBotLegacyMode(false) })
	stringRef := ""

	basePlugins := make(map[string]map[string]interface{})
	basePlugins[PluginName] = make(map[string]interface{})
	basePlugins[PluginName][SharedBoardsName] = true
	basePlugins[PluginName][LegacyModeName] = true
	basePlugins[PluginName][MovedBoardsReadOnlyName] = true

	basePluginSettings := &serverModel.PluginSettings{
		Directory: &stringRef,
		Plugins:   basePlugins,
	}
	intRef := 365
	baseDataRetentionSettings := &serverModel.DataRetentionSettings{
		BoardsRetentionDays: &intRef,
	}
	usernameRef := "username"
	baseTeamSettings := &serverModel.TeamSettings{
		TeammateNameDisplay: &usernameRef,
	}

	falseRef := false
	basePrivacySettings := &serverModel.PrivacySettings{
		ShowEmailAddress: &falseRef,
		ShowFullName:     &falseRef,
	}

	baseConfig := &serverModel.Config{
		PluginSettings:        *basePluginSettings,
		DataRetentionSettings: *baseDataRetentionSettings,
		TeamSettings:          *baseTeamSettings,
		PrivacySettings:       *basePrivacySettings,
	}

	t.Run("Test Load Plugin Success", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		ctrl := gomock.NewController(t)
		api := mockservicesapi.NewMockServicesAPI(ctrl)
		api.EXPECT().GetConfig().Return(baseConfig)
		api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{UserId: "bot-id", DisplayName: "Boards"}, nil)
		api.EXPECT().PatchBot("bot-id", gomock.Any()).DoAndReturn(
			func(_ string, patch *serverModel.BotPatch) (*serverModel.Bot, error) {
				require.NotNil(t, patch.DisplayName)
				assert.Equal(t, "Legacy Boards", *patch.DisplayName)
				return &serverModel.Bot{UserId: "bot-id", DisplayName: *patch.DisplayName}, nil
			})

		b := &BoardsApp{
			server:          th.Server,
			wsPluginAdapter: &FakePluginAdapter{},
			servicesAPI:     api,
			logger:          mlog.CreateConsoleTestLogger(&testing.T{}),
		}

		err := b.OnConfigurationChange()
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		// make sure both App and Server got updated
		assert.True(t, b.server.Config().EnablePublicSharedBoards)
		assert.True(t, b.server.App().GetClientConfig().EnablePublicSharedBoards)
		assert.True(t, b.getConfiguration().LegacyMode)
		assert.True(t, b.server.Config().LegacyMode)
		assert.True(t, b.server.App().GetClientConfig().LegacyMode)
		assert.True(t, b.getConfiguration().MovedBoardsReadOnly)
		assert.True(t, b.server.Config().MovedBoardsReadOnly)
		assert.True(t, b.server.App().GetClientConfig().MovedBoardsReadOnly)
	})

	movedBoardsReadOnlyConfig := func(value interface{}, set bool) *serverModel.Config {
		plugins := map[string]map[string]interface{}{PluginName: {
			SharedBoardsName: true,
			LegacyModeName:   true,
		}}
		if set {
			plugins[PluginName][MovedBoardsReadOnlyName] = value
		}
		cfg := *baseConfig
		cfg.PluginSettings = serverModel.PluginSettings{Directory: &stringRef, Plugins: plugins}
		return &cfg
	}

	for name, tc := range map[string]struct {
		value interface{}
		set   bool
		want  bool
	}{
		"absent setting reads as on":          {set: false, want: true},
		"explicit false stays off":            {value: false, set: true, want: false},
		"explicit true stays on":              {value: true, set: true, want: true},
		"setting of an unexpected type is on": {value: "nope", set: true, want: true},
	} {
		t.Run("MovedBoardsReadOnly: "+name, func(t *testing.T) {
			th, tearDown := SetupTestHelper(t)
			defer tearDown()

			api := mockservicesapi.NewMockServicesAPI(gomock.NewController(t))
			api.EXPECT().GetConfig().Return(movedBoardsReadOnlyConfig(tc.value, tc.set))
			api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("bot-id", nil)
			api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{UserId: "bot-id", DisplayName: "Legacy Boards"}, nil)

			b := &BoardsApp{
				server:          th.Server,
				wsPluginAdapter: &FakePluginAdapter{},
				servicesAPI:     api,
				logger:          mlog.CreateConsoleTestLogger(&testing.T{}),
			}

			require.NoError(t, b.OnConfigurationChange())
			assert.Equal(t, tc.want, b.getConfiguration().MovedBoardsReadOnly)
			assert.Equal(t, tc.want, b.server.Config().MovedBoardsReadOnly)
			assert.Equal(t, tc.want, b.server.App().GetClientConfig().MovedBoardsReadOnly)
		})
	}

	t.Run("Bot display name already up to date is not patched", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		ctrl := gomock.NewController(t)
		api := mockservicesapi.NewMockServicesAPI(ctrl)
		api.EXPECT().GetConfig().Return(baseConfig)
		api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{UserId: "bot-id", DisplayName: "Legacy Boards"}, nil)

		b := &BoardsApp{
			server:          th.Server,
			wsPluginAdapter: &FakePluginAdapter{},
			servicesAPI:     api,
			logger:          mlog.CreateConsoleTestLogger(&testing.T{}),
		}

		assert.NoError(t, b.OnConfigurationChange())
	})

	t.Run("Bot patch failure does not fail the configuration change", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		ctrl := gomock.NewController(t)
		api := mockservicesapi.NewMockServicesAPI(ctrl)
		api.EXPECT().GetConfig().Return(baseConfig)
		api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{UserId: "bot-id", DisplayName: "Boards"}, nil)
		api.EXPECT().PatchBot("bot-id", gomock.Any()).Return(nil, errPatchFailed)

		b := &BoardsApp{
			server:          th.Server,
			wsPluginAdapter: &FakePluginAdapter{},
			servicesAPI:     api,
			logger:          mlog.CreateConsoleTestLogger(&testing.T{}),
		}

		assert.NoError(t, b.OnConfigurationChange())
		assert.True(t, b.server.Config().LegacyMode)
	})
}

func botTemplate(displayName string) *serverModel.Bot {
	bot := model.FocalboardBot()
	bot.DisplayName = displayName
	return bot
}

func TestSyncBotDisplayName(t *testing.T) {
	t.Cleanup(func() { model.SetBotLegacyMode(false) })

	newApp := func(t *testing.T) (*BoardsApp, *mockservicesapi.MockServicesAPI) {
		api := mockservicesapi.NewMockServicesAPI(gomock.NewController(t))
		return &BoardsApp{servicesAPI: api}, api
	}

	t.Run("other EnsureBot callers keep the legacy display name", func(t *testing.T) {
		b, api := newApp(t)
		api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{DisplayName: "Legacy Boards"}, nil)

		assert.NoError(t, b.syncBotDisplayName(true))
		assert.Equal(t, "Legacy Boards", model.FocalboardBot().DisplayName)
	})

	t.Run("patches to Boards when legacy mode is off", func(t *testing.T) {
		b, api := newApp(t)
		api.EXPECT().EnsureBot(botTemplate("Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{DisplayName: "Legacy Boards"}, nil)
		api.EXPECT().PatchBot("bot-id", gomock.Any()).DoAndReturn(
			func(_ string, patch *serverModel.BotPatch) (*serverModel.Bot, error) {
				require.NotNil(t, patch.DisplayName)
				assert.Equal(t, "Boards", *patch.DisplayName)
				return &serverModel.Bot{}, nil
			})

		assert.NoError(t, b.syncBotDisplayName(false))
	})

	t.Run("skips patch when name matches", func(t *testing.T) {
		b, api := newApp(t)
		api.EXPECT().EnsureBot(botTemplate("Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(&serverModel.Bot{DisplayName: "Boards"}, nil)

		assert.NoError(t, b.syncBotDisplayName(false))
	})

	t.Run("returns errors from the services API", func(t *testing.T) {
		b, api := newApp(t)
		api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("", errEnsureFailed)
		assert.ErrorIs(t, b.syncBotDisplayName(true), errEnsureFailed)

		b, api = newApp(t)
		api.EXPECT().EnsureBot(botTemplate("Legacy Boards")).Return("bot-id", nil)
		api.EXPECT().GetBot("bot-id", false).Return(nil, errGetFailed)
		assert.ErrorIs(t, b.syncBotDisplayName(true), errGetFailed)
	})
}

var count = 0

type FakePluginAdapter struct {
	ws.PluginAdapter
}

func (c *FakePluginAdapter) BroadcastConfigChange(clientConfig model.ClientConfig) {
	count++
}
