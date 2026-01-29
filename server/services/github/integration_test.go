// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package github

import (
	"net/http"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/stretchr/testify/assert"
)

// mockModelServicesAPI implements model.ServicesAPI for testing
type mockModelServicesAPI struct {
	model.ServicesAPI
	pluginHTTPFunc func(req *http.Request) *http.Response
	logger         mlog.LoggerIFace
}

func (m *mockModelServicesAPI) PluginHTTP(req *http.Request) *http.Response {
	if m.pluginHTTPFunc != nil {
		return m.pluginHTTPFunc(req)
	}
	return nil
}

func (m *mockModelServicesAPI) GetLogger() mlog.LoggerIFace {
	if m.logger != nil {
		return m.logger
	}
	logger, _ := mlog.NewLogger()
	return logger
}

// TestServiceWithModelServicesAPI verifies that the GitHub service can be
// instantiated with model.ServicesAPI interface
func TestServiceWithModelServicesAPI(t *testing.T) {
	mockAPI := &mockModelServicesAPI{
		pluginHTTPFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}
		},
	}

	// This should compile and work with model.ServicesAPI
	service := New(mockAPI)
	assert.NotNil(t, service)
	assert.NotNil(t, service.api)
}

