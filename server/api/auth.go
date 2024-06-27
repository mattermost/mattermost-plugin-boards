package api

import (
	"context"
	"net/http"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/auth"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (a *API) sessionRequired(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return a.attachSession(handler, true)
}

func (a *API) attachSession(handler func(w http.ResponseWriter, r *http.Request), required bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := auth.ParseAuthTokenFromRequest(r)

		a.logger.Debug(`attachSession`, mlog.Bool("single_user", len(a.singleUserToken) > 0))

		if a.MattermostAuth && r.Header.Get("Mattermost-User-Id") != "" {
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

		session, err := a.app.GetSession(token)
		if err != nil {
			if required {
				a.errorResponse(w, r, model.NewErrUnauthorized(err.Error()))
				return
			}

			handler(w, r)
			return
		}

		authService := session.AuthService
		if authService != a.authService {
			msg := `Session authService mismatch`
			a.logger.Error(msg,
				mlog.String("sessionID", session.ID),
				mlog.String("want", a.authService),
				mlog.String("got", authService),
			)
			a.errorResponse(w, r, model.NewErrUnauthorized(msg))
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		handler(w, r.WithContext(ctx))
	}
}
