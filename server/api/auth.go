// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"context"
	"net/http"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

// requireUserID writes an unauthorized error response and returns true when
// userID is empty, signaling the caller to abort request handling.
func (a *API) requireUserID(w http.ResponseWriter, r *http.Request, userID, msg string) bool {
	if userID == "" {
		a.errorResponse(w, r, model.NewErrUnauthorized(msg))
		return true
	}
	return false
}

func (a *API) sessionRequired(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return a.attachSession(handler)
}

func (a *API) attachSession(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Mattermost-User-Id") != "" {
			userID := r.Header.Get("Mattermost-User-Id")
			now := utils.GetMillis()
			session := &model.Session{
				ID:          userID,
				Token:       userID,
				UserID:      userID,
				AuthService: a.authService,
				Props:       map[string]interface{}{},
				CreateAt:    now,
				UpdateAt:    now,
			}

			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			handler(w, r.WithContext(ctx))
			return
		}

		handler(w, r)
	}
}
