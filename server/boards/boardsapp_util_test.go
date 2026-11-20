// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package boards

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mm_model "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
)

func newFilesTestConfig(fileSettings mm_model.FileSettings) mm_model.Config {
	cfg := mm_model.Config{}
	cfg.SetDefaults()
	cfg.FileSettings = fileSettings
	cfg.FileSettings.SetDefaults(false)
	return cfg
}

func TestCreateFilesBackendSettings(t *testing.T) {
	t.Run("azureblob with default credential on the government cloud", func(t *testing.T) {
		cfg := newFilesTestConfig(mm_model.FileSettings{
			DriverName:                      mm_model.NewPointer(mm_model.ImageDriverAzure),
			AzureStorageAccount:             mm_model.NewPointer("boardstestaccount"),
			AzureAuthMode:                   mm_model.NewPointer(mm_model.AzureAuthModeDefaultCredential),
			AzureContainer:                  mm_model.NewPointer("boards-test"),
			AzurePathPrefix:                 mm_model.NewPointer("prefix"),
			AzureCloud:                      mm_model.NewPointer(mm_model.AzureCloudGovernment),
			AzureSSL:                        mm_model.NewPointer(true),
			AzureRequestTimeoutMilliseconds: mm_model.NewPointer(int64(45000)),
		})

		settings := createFilesBackendSettings(cfg)
		assert.Equal(t, mm_model.ImageDriverAzure, settings.DriverName)
		assert.Equal(t, "boardstestaccount", settings.AzureStorageAccount)
		assert.Equal(t, mm_model.AzureAuthModeDefaultCredential, settings.AzureAuthMode)
		assert.Equal(t, "boards-test", settings.AzureContainer)
		assert.Equal(t, "prefix", settings.AzurePathPrefix)
		assert.Equal(t, mm_model.AzureCloudGovernment, settings.AzureCloud)
		assert.True(t, settings.AzureSSL)
		assert.Equal(t, int64(45000), settings.AzureRequestTimeoutMilliseconds)

		// The settings must reach the Boards server config unchanged.
		boardsCfg := createBoardsConfig(cfg, "", "")
		assert.Equal(t, mm_model.ImageDriverAzure, boardsCfg.FilesDriver)
		assert.Equal(t, settings, boardsCfg.FilesBackendSettings)

		// Building the client does not contact Azure: DefaultAzureCredential
		// only acquires tokens on the first request.
		backend, err := filestore.NewFileBackend(boardsCfg.FilesBackendSettings)
		require.NoError(t, err)
		assert.Equal(t, mm_model.ImageDriverAzure, backend.DriverName())
	})

	t.Run("azureblob with shared key on a custom endpoint", func(t *testing.T) {
		cfg := newFilesTestConfig(mm_model.FileSettings{
			DriverName:          mm_model.NewPointer(mm_model.ImageDriverAzure),
			AzureStorageAccount: mm_model.NewPointer("boardstestaccount"),
			AzureAuthMode:       mm_model.NewPointer(mm_model.AzureAuthModeSharedKey),
			AzureAccessKey:      mm_model.NewPointer(base64.StdEncoding.EncodeToString([]byte("not-a-real-key"))),
			AzureContainer:      mm_model.NewPointer("boards-test"),
			AzureCloud:          mm_model.NewPointer(mm_model.AzureCloudCustom),
			AzureEndpoint:       mm_model.NewPointer("http://127.0.0.1:10000/boardstestaccount"),
			AzureSSL:            mm_model.NewPointer(false),
		})
		cfg.ServiceSettings.AllowedUntrustedInternalConnections = mm_model.NewPointer("127.0.0.1")

		settings := createFilesBackendSettings(cfg)
		assert.Equal(t, "127.0.0.1", settings.AllowedUntrustedInternalConnections)
		assert.Equal(t, mm_model.AzureAuthModeSharedKey, settings.AzureAuthMode)
		assert.NotEmpty(t, settings.AzureAccessKey)
		assert.Equal(t, mm_model.AzureCloudCustom, settings.AzureCloud)
		assert.Equal(t, "http://127.0.0.1:10000/boardstestaccount", settings.AzureEndpoint)
		assert.False(t, settings.AzureSSL)

		backend, err := filestore.NewFileBackend(settings)
		require.NoError(t, err)
		assert.Equal(t, mm_model.ImageDriverAzure, backend.DriverName())
	})

	t.Run("amazons3 keeps the Boards defaults", func(t *testing.T) {
		cfg := newFilesTestConfig(mm_model.FileSettings{
			DriverName:                         mm_model.NewPointer(mm_model.ImageDriverS3),
			AmazonS3Bucket:                     mm_model.NewPointer("boards-test"),
			AmazonS3SSE:                        mm_model.NewPointer(true),
			AmazonS3RequestTimeoutMilliseconds: mm_model.NewPointer(int64(0)),
		})

		settings := createFilesBackendSettings(cfg)
		assert.Equal(t, mm_model.ImageDriverS3, settings.DriverName)
		assert.Equal(t, "boards-test", settings.AmazonS3Bucket)
		assert.True(t, settings.AmazonS3SSE, "SSE is honored as configured, as before")
		assert.Equal(t, int64(defaultS3Timeout), settings.AmazonS3RequestTimeoutMilliseconds)
	})

	t.Run("local", func(t *testing.T) {
		cfg := newFilesTestConfig(mm_model.FileSettings{
			DriverName: mm_model.NewPointer(mm_model.ImageDriverLocal),
			Directory:  mm_model.NewPointer("./data/"),
		})

		settings := createFilesBackendSettings(cfg)
		assert.Equal(t, filestore.FileBackendSettings{DriverName: mm_model.ImageDriverLocal, Directory: "./data/"}, settings)
	})

	t.Run("insecure outgoing connections set SkipVerify", func(t *testing.T) {
		cfg := newFilesTestConfig(mm_model.FileSettings{
			DriverName: mm_model.NewPointer(mm_model.ImageDriverAzure),
		})
		cfg.ServiceSettings.EnableInsecureOutgoingConnections = mm_model.NewPointer(true)

		settings := createFilesBackendSettings(cfg)
		assert.True(t, settings.SkipVerify)
	})

	t.Run("unset FileSettings fields do not panic", func(t *testing.T) {
		cfg := mm_model.Config{}
		cfg.FileSettings.DriverName = mm_model.NewPointer(mm_model.ImageDriverAzure)

		require.NotPanics(t, func() {
			settings := createFilesBackendSettings(cfg)
			assert.Equal(t, mm_model.ImageDriverAzure, settings.DriverName)
		})
		assert.Nil(t, cfg.FileSettings.AzureContainer, "caller's config is not modified")
	})
}
