package sqlstore

import (
	"database/sql"
	"fmt"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	mmModel "github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost/server/public/shared/mlog"

	sq "github.com/Masterminds/squirrel"
)

type UserNotFoundError struct {
	id string
}

func (unf UserNotFoundError) Error() string {
	return fmt.Sprintf("user not found (%s)", unf.id)
}

func (s *SQLStore) baseUserQuery(showEmail, showName bool) sq.SelectBuilder {
	emailField := "''"
	if showEmail {
		emailField = "u.email"
	}
	firstNameField := "''"
	lastNameField := "''"
	if showName {
		firstNameField = "u.firstname"
		lastNameField = "u.lastname"
	}

	return s.getQueryBuilder(s.db).
		Select(
			"u.id",
			"u.username",
			emailField,
			"u.nickname",
			firstNameField,
			lastNameField,
			"u.CreateAt as create_at",
			"u.UpdateAt as update_at",
			"u.DeleteAt as delete_at",
			"b.UserId IS NOT NULL AS is_bot",
			"u.roles = 'system_guest' as is_guest",
		).
		From("Users as u").
		LeftJoin("Bots b ON ( b.UserID = u.id )")
}

func (s *SQLStore) usersFromRows(rows *sql.Rows) ([]*model.User, error) {
	users := []*model.User{}

	for rows.Next() {
		var user model.User

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Nickname,
			&user.FirstName,
			&user.LastName,
			&user.CreateAt,
			&user.UpdateAt,
			&user.DeleteAt,
			&user.IsBot,
			&user.IsGuest,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	return users, nil
}

func mmUserToFbUser(mmUser *mmModel.User) model.User {
	authData := ""
	if mmUser.AuthData != nil {
		authData = *mmUser.AuthData
	}
	return model.User{
		ID:          mmUser.Id,
		Username:    mmUser.Username,
		Email:       mmUser.Email,
		Password:    mmUser.Password,
		Nickname:    mmUser.Nickname,
		FirstName:   mmUser.FirstName,
		LastName:    mmUser.LastName,
		MfaSecret:   mmUser.MfaSecret,
		AuthService: mmUser.AuthService,
		AuthData:    authData,
		CreateAt:    mmUser.CreateAt,
		UpdateAt:    mmUser.UpdateAt,
		DeleteAt:    mmUser.DeleteAt,
		IsBot:       mmUser.IsBot,
		IsGuest:     mmUser.IsGuest(),
		Roles:       mmUser.Roles,
	}
}

func (s *SQLStore) GetRegisteredUserCount() (int, error) {
	query := s.getQueryBuilder(s.db).
		Select("count(*)").
		From("Users").
		Where(sq.Eq{"deleteAt": 0})
	row := query.QueryRow()

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *SQLStore) GetUserByID(userID string) (*model.User, error) {
	mmuser, err := s.servicesAPI.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	user := mmUserToFbUser(mmuser)
	return &user, nil
}

func (s *SQLStore) GetUserByEmail(email string) (*model.User, error) {
	mmuser, err := s.servicesAPI.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	user := mmUserToFbUser(mmuser)
	return &user, nil
}

func (s *SQLStore) GetUserByUsername(username string) (*model.User, error) {
	mmuser, err := s.servicesAPI.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	user := mmUserToFbUser(mmuser)
	return &user, nil
}

func (s *SQLStore) PatchUserPreferences(userID string, patch model.UserPreferencesPatch) (mmModel.Preferences, error) {
	preferences, err := s.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}

	if len(patch.UpdatedFields) > 0 {
		updatedPreferences := mmModel.Preferences{}
		for key, value := range patch.UpdatedFields {
			preference := mmModel.Preference{
				UserId:   userID,
				Category: model.PreferencesCategoryFocalboard,
				Name:     key,
				Value:    value,
			}

			updatedPreferences = append(updatedPreferences, preference)
		}

		if err := s.servicesAPI.UpdatePreferencesForUser(userID, updatedPreferences); err != nil {
			s.logger.Error("failed to update user preferences", mlog.String("user_id", userID), mlog.Err(err))
			return nil, err
		}

		// we update the preferences list replacing or adding those
		// that were updated
		newPreferences := mmModel.Preferences{}
		for _, existingPreference := range preferences {
			hasBeenUpdated := false
			for _, updatedPreference := range updatedPreferences {
				if updatedPreference.Name == existingPreference.Name {
					hasBeenUpdated = true
					break
				}
			}

			if !hasBeenUpdated {
				newPreferences = append(newPreferences, existingPreference)
			}
		}
		newPreferences = append(newPreferences, updatedPreferences...)
		preferences = newPreferences
	}

	if len(patch.DeletedFields) > 0 {
		deletedPreferences := mmModel.Preferences{}
		for _, key := range patch.DeletedFields {
			preference := mmModel.Preference{
				UserId:   userID,
				Category: model.PreferencesCategoryFocalboard,
				Name:     key,
			}

			deletedPreferences = append(deletedPreferences, preference)
		}

		if err := s.servicesAPI.DeletePreferencesForUser(userID, deletedPreferences); err != nil {
			s.logger.Error("failed to delete user preferences", mlog.String("user_id", userID), mlog.Err(err))
			return nil, err
		}

		// we update the preferences removing those that have been
		// deleted
		newPreferences := mmModel.Preferences{}
		for _, existingPreference := range preferences {
			hasBeenDeleted := false
			for _, deletedPreference := range deletedPreferences {
				if deletedPreference.Name == existingPreference.Name {
					hasBeenDeleted = true
					break
				}
			}

			if !hasBeenDeleted {
				newPreferences = append(newPreferences, existingPreference)
			}
		}
		preferences = newPreferences
	}

	return preferences, nil
}

func (s *SQLStore) GetUserPreferences(userID string) (mmModel.Preferences, error) {
	return s.servicesAPI.GetPreferencesForUser(userID)
}

// GetActiveUserCount returns the number of users with active sessions within N seconds ago.
func (s *SQLStore) GetActiveUserCount(updatedSecondsAgo int64) (int, error) {
	query := s.getQueryBuilder(s.db).
		Select("count(distinct userId)").
		From("Sessions").
		Where(sq.Gt{"LastActivityAt": utils.GetMillis() - utils.SecondsToMillis(updatedSecondsAgo)})

	row := query.QueryRow()

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *SQLStore) GetUsersByTeam(teamID string, asGuestID string, showEmail, showName bool) ([]*model.User, error) {
	query := s.baseUserQuery(showEmail, showName).
		Where(sq.Eq{"u.deleteAt": 0})

	if asGuestID == "" {
		query = query.
			Join("TeamMembers as tm ON tm.UserID = u.id").
			Where(sq.Eq{"tm.TeamId": teamID})
	} else {
		boards, err := s.GetBoardsForUserAndTeam(asGuestID, teamID, false)
		if err != nil {
			return nil, err
		}

		boardsIDs := []string{}
		for _, board := range boards {
			boardsIDs = append(boardsIDs, board.ID)
		}
		query = query.
			Join(s.tablePrefix + "board_members as bm ON bm.UserID = u.ID").
			Where(sq.Eq{"bm.BoardId": boardsIDs})
	}

	rows, err := query.Query()
	if err != nil {
		return nil, err
	}
	defer s.CloseRows(rows)

	users, err := s.usersFromRows(rows)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *SQLStore) GetUsersList(userIDs []string, showEmail, showName bool) ([]*model.User, error) {
	query := s.baseUserQuery(showEmail, showName).
		Where(sq.Eq{"u.id": userIDs})

	rows, err := query.Query()
	if err != nil {
		return nil, err
	}
	defer s.CloseRows(rows)

	users, err := s.usersFromRows(rows)
	if err != nil {
		return nil, err
	}

	if len(users) != len(userIDs) {
		return users, model.NewErrNotAllFound("user", userIDs)
	}

	return users, nil
}

func (s *SQLStore) SearchUsersByTeam(teamID string, searchQuery string, asGuestID string, excludeBots, showEmail, showName bool) ([]*model.User, error) {
	query := s.baseUserQuery(showEmail, showName).
		Where(sq.Eq{"u.deleteAt": 0}).
		Where(sq.Or{
			sq.Like{"u.username": "%" + searchQuery + "%"},
			sq.Like{"u.nickname": "%" + searchQuery + "%"},
			sq.Like{"u.firstname": "%" + searchQuery + "%"},
			sq.Like{"u.lastname": "%" + searchQuery + "%"},
		}).
		OrderBy("u.username").
		Limit(10)

	if excludeBots {
		query = query.
			Where(sq.Eq{"b.UserId IS NOT NULL": false})
	}

	if asGuestID == "" {
		query = query.
			Join("TeamMembers as tm ON tm.UserID = u.id").
			Where(sq.Eq{"tm.TeamId": teamID})
	} else {
		boards, err := s.GetBoardsForUserAndTeam(asGuestID, teamID, false)
		if err != nil {
			return nil, err
		}
		boardsIDs := []string{}
		for _, board := range boards {
			boardsIDs = append(boardsIDs, board.ID)
		}
		query = query.
			Join(s.tablePrefix + "board_members as bm ON bm.user_id = u.ID").
			Where(sq.Eq{"bm.board_id": boardsIDs})
	}

	rows, err := query.Query()
	if err != nil {
		return nil, err
	}
	defer s.CloseRows(rows)

	users, err := s.usersFromRows(rows)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *SQLStore) GetUserTimezone(userID string) (string, error) {
	user, err := s.servicesAPI.GetUserByID(userID)
	if err != nil {
		return "", err
	}
	timezone := user.Timezone
	return mmModel.GetPreferredTimezone(timezone), nil
}

func (s *SQLStore) CanSeeUser(seerID string, seenID string) (bool, error) {
	mmuser, appErr := s.servicesAPI.GetUserByID(seerID)
	if appErr != nil {
		return false, appErr
	}
	if !mmuser.IsGuest() {
		return true, nil
	}

	query := s.getQueryBuilder(s.db).
		Select("1").
		From(s.tablePrefix + "board_members AS bm1").
		Join(s.tablePrefix + "board_members AS bm2 ON bm1.board_id=bm2.board_id").
		Where(sq.Or{
			sq.And{
				sq.Eq{"bm1.user_id": seerID},
				sq.Eq{"bm2.user_id": seenID},
			},
			sq.And{
				sq.Eq{"bm1.user_id": seenID},
				sq.Eq{"bm2.user_id": seerID},
			},
		}).Limit(1)

	rows, err := query.Query()
	if err != nil {
		return false, err
	}
	defer s.CloseRows(rows)

	for rows.Next() {
		return true, err
	}

	query = s.getQueryBuilder(s.db).
		Select("1").
		From("channelmembers AS cm1").
		Join("channelmembers AS cm2 ON cm1.channelid=cm2.channelid").
		Where(sq.Or{
			sq.And{
				sq.Eq{"cm1.userid": seerID},
				sq.Eq{"cm2.userid": seenID},
			},
			sq.And{
				sq.Eq{"cm1.userid": seenID},
				sq.Eq{"cm2.userid": seerID},
			},
		}).Limit(1)

	rows, err = query.Query()
	if err != nil {
		return false, err
	}
	defer s.CloseRows(rows)

	for rows.Next() {
		return true, err
	}

	return false, nil
}
