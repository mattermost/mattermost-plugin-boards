//go:generate mockgen -destination=mocks/mockauth_interface.go -package mocks . AuthInterface
package auth

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/config"
	"github.com/mattermost/mattermost-plugin-boards/server/services/permissions"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/pkg/errors"
)

type AuthInterface interface {
	IsValidReadToken(boardID string, readToken string) (bool, error)
	DoesUserHaveTeamAccess(userID string, teamID string) bool
}

// Auth authenticates sessions.
type Auth struct {
	config      *config.Configuration
	store       store.Store
	permissions permissions.PermissionsService
}

// New returns a new Auth.
func New(config *config.Configuration, store store.Store, permissions permissions.PermissionsService) *Auth {
	return &Auth{config: config, store: store, permissions: permissions}
}

// IsValidReadToken validates the read token for a board.
func (a *Auth) IsValidReadToken(boardID string, readToken string) (bool, error) {
	sharing, err := a.store.GetSharing(boardID)
	if model.IsErrNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !a.config.EnablePublicSharedBoards {
		return false, errors.New("public shared boards disabled")
	}

	if sharing != nil && (sharing.ID == boardID && sharing.Enabled && sharing.Token == readToken) {
		return true, nil
	}

	return false, nil
}

func (a *Auth) DoesUserHaveTeamAccess(userID string, teamID string) bool {
	return a.permissions.HasPermissionToTeam(userID, teamID, model.PermissionViewTeam)
}
