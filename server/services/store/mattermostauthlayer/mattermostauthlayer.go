package mattermostauthlayer

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	mmModel "github.com/mattermost/mattermost/server/public/model"

	sq "github.com/Masterminds/squirrel"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

var boardsBotID string

// servicesAPI is the interface required my the MattermostAuthLayer to interact with
// the mattermost-server. You can use plugin-api or product-api adapter implementations.
type servicesAPI interface {
	GetDirectChannel(userID1, userID2 string) (*mmModel.Channel, error)
	GetChannelByID(channelID string) (*mmModel.Channel, error)
	GetChannelMember(channelID string, userID string) (*mmModel.ChannelMember, error)
	GetChannelsForTeamForUser(teamID string, userID string, includeDeleted bool) (mmModel.ChannelList, error)
	GetUserByID(userID string) (*mmModel.User, error)
	UpdateUser(user *mmModel.User) (*mmModel.User, error)
	GetUserByEmail(email string) (*mmModel.User, error)
	GetUserByUsername(username string) (*mmModel.User, error)
	GetLicense() *mmModel.License
	GetFileInfo(fileID string) (*mmModel.FileInfo, error)
	EnsureBot(bot *mmModel.Bot) (string, error)
	CreatePost(post *mmModel.Post) (*mmModel.Post, error)
	GetTeamMember(teamID string, userID string) (*mmModel.TeamMember, error)
	GetPreferencesForUser(userID string) (mmModel.Preferences, error)
	DeletePreferencesForUser(userID string, preferences mmModel.Preferences) error
	UpdatePreferencesForUser(userID string, preferences mmModel.Preferences) error
}

// Store represents the abstraction of the data storage.
type MattermostAuthLayer struct {
	store.Store
	dbType      string
	mmDB        *sql.DB
	logger      mlog.LoggerIFace
	servicesAPI servicesAPI
	tablePrefix string
}

// New creates a new SQL implementation of the store.
func New(dbType string, db *sql.DB, store store.Store, logger mlog.LoggerIFace, api servicesAPI, tablePrefix string) (*MattermostAuthLayer, error) {
	layer := &MattermostAuthLayer{
		Store:       store,
		dbType:      dbType,
		mmDB:        db,
		logger:      logger,
		servicesAPI: api,
		tablePrefix: tablePrefix,
	}

	return layer, nil
}

// Shutdown close the connection with the store.
func (s *MattermostAuthLayer) Shutdown() error {
	return s.Store.Shutdown()
}

func (s *MattermostAuthLayer) GetRegisteredUserCount() (int, error) {
	query := s.getQueryBuilder().
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

func (s *MattermostAuthLayer) GetUserByID(userID string) (*model.User, error) {
	mmuser, err := s.servicesAPI.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	user := mmUserToFbUser(mmuser)
	return &user, nil
}

func (s *MattermostAuthLayer) GetUserByEmail(email string) (*model.User, error) {
	mmuser, err := s.servicesAPI.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	user := mmUserToFbUser(mmuser)
	return &user, nil
}

func (s *MattermostAuthLayer) GetUserByUsername(username string) (*model.User, error) {
	mmuser, err := s.servicesAPI.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	user := mmUserToFbUser(mmuser)
	return &user, nil
}

func (s *MattermostAuthLayer) PatchUserPreferences(userID string, patch model.UserPreferencesPatch) (mmModel.Preferences, error) {
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

func (s *MattermostAuthLayer) GetUserPreferences(userID string) (mmModel.Preferences, error) {
	return s.servicesAPI.GetPreferencesForUser(userID)
}

// GetActiveUserCount returns the number of users with active sessions within N seconds ago.
func (s *MattermostAuthLayer) GetActiveUserCount(updatedSecondsAgo int64) (int, error) {
	query := s.getQueryBuilder().
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

func (s *MattermostAuthLayer) GetTeam(id string) (*model.Team, error) {
	if id == "0" {
		team := model.Team{
			ID:    id,
			Title: "",
		}

		return &team, nil
	}

	query := s.getQueryBuilder().
		Select("DisplayName").
		From("Teams").
		Where(sq.Eq{"ID": id})

	row := query.QueryRow()
	var displayName string
	err := row.Scan(&displayName)
	if err != nil && !model.IsErrNotFound(err) {
		s.logger.Error("GetTeam scan error",
			mlog.String("team_id", id),
			mlog.Err(err),
		)
		return nil, err
	}

	return &model.Team{ID: id, Title: displayName}, nil
}

// GetTeamsForUser retrieves all the teams that the user is a member of.
func (s *MattermostAuthLayer) GetTeamsForUser(userID string) ([]*model.Team, error) {
	query := s.getQueryBuilder().
		Select("t.Id", "t.DisplayName").
		From("Teams as t").
		Join("TeamMembers as tm on t.Id=tm.TeamId").
		Where(sq.Eq{"tm.UserId": userID}).
		Where(sq.Eq{"tm.DeleteAt": 0})

	rows, err := query.Query()
	if err != nil {
		return nil, err
	}
	defer s.CloseRows(rows)

	teams := []*model.Team{}
	for rows.Next() {
		var team model.Team

		err := rows.Scan(
			&team.ID,
			&team.Title,
		)
		if err != nil {
			return nil, err
		}

		teams = append(teams, &team)
	}

	return teams, nil
}

func (s *MattermostAuthLayer) getQueryBuilder() sq.StatementBuilderType {
	builder := sq.StatementBuilder
	if s.dbType == model.PostgresDBType || s.dbType == model.SqliteDBType {
		builder = builder.PlaceholderFormat(sq.Dollar)
	}

	return builder.RunWith(s.mmDB)
}

func (s *MattermostAuthLayer) GetUsersByTeam(teamID string, asGuestID string, showEmail, showName bool) ([]*model.User, error) {
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

func (s *MattermostAuthLayer) GetUsersList(userIDs []string, showEmail, showName bool) ([]*model.User, error) {
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

func (s *MattermostAuthLayer) SearchUsersByTeam(teamID string, searchQuery string, asGuestID string, excludeBots, showEmail, showName bool) ([]*model.User, error) {
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

func (s *MattermostAuthLayer) usersFromRows(rows *sql.Rows) ([]*model.User, error) {
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

func (s *MattermostAuthLayer) CloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		s.logger.Error("error closing MattermostAuthLayer row set", mlog.Err(err))
	}
}

func (s *MattermostAuthLayer) CreatePrivateWorkspace(userID string) (string, error) {
	// we emulate a private workspace by creating
	// a DM channel from the user to themselves.
	channel, err := s.servicesAPI.GetDirectChannel(userID, userID)
	if err != nil {
		s.logger.Error("error fetching private workspace", mlog.String("userID", userID), mlog.Err(err))
		return "", err
	}

	return channel.Id, nil
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

func (s *MattermostAuthLayer) GetFileInfo(id string) (*mmModel.FileInfo, error) {
	fileInfo, err := s.servicesAPI.GetFileInfo(id)
	if err != nil {
		// Not finding fileinfo is fine because we don't have data for
		// any existing files already uploaded in Boards before this code
		// was deployed.
		var appErr *mmModel.AppError
		if errors.As(err, &appErr) {
			if appErr.StatusCode == http.StatusNotFound {
				return nil, model.NewErrNotFound("file info ID=" + id)
			}
		}

		s.logger.Error("error fetching fileinfo",
			mlog.String("id", id),
			mlog.Err(err),
		)
		return nil, err
	}

	return fileInfo, nil
}

func (s *MattermostAuthLayer) SaveFileInfo(fileInfo *mmModel.FileInfo) error {
	query := s.getQueryBuilder().
		Insert("FileInfo").
		Columns(
			"Id",
			"CreatorId",
			"PostId",
			"CreateAt",
			"UpdateAt",
			"DeleteAt",
			"Path",
			"ThumbnailPath",
			"PreviewPath",
			"Name",
			"Extension",
			"Size",
			"MimeType",
			"Width",
			"Height",
			"HasPreviewImage",
			"MiniPreview",
			"Content",
			"RemoteId",
			"Archived",
		).
		Values(
			fileInfo.Id,
			fileInfo.CreatorId,
			fileInfo.PostId,
			fileInfo.CreateAt,
			fileInfo.UpdateAt,
			fileInfo.DeleteAt,
			fileInfo.Path,
			fileInfo.ThumbnailPath,
			fileInfo.PreviewPath,
			fileInfo.Name,
			fileInfo.Extension,
			fileInfo.Size,
			fileInfo.MimeType,
			fileInfo.Width,
			fileInfo.Height,
			fileInfo.HasPreviewImage,
			fileInfo.MiniPreview,
			fileInfo.Content,
			fileInfo.RemoteId,
			false,
		)

	if _, err := query.Exec(); err != nil {
		s.logger.Error(
			"failed to save fileinfo",
			mlog.String("file_name", fileInfo.Name),
			mlog.Int("size", fileInfo.Size),
			mlog.Err(err),
		)
		return err
	}

	return nil
}

func (s *MattermostAuthLayer) GetLicense() *mmModel.License {
	return s.servicesAPI.GetLicense()
}

func boardFields(prefix string) []string { //nolint:unparam
	fields := []string{
		"id",
		"team_id",
		"COALESCE(channel_id, '')",
		"COALESCE(created_by, '')",
		"modified_by",
		"type",
		"minimum_role",
		"title",
		"description",
		"icon",
		"show_description",
		"is_template",
		"template_version",
		"COALESCE(properties, '{}')",
		"COALESCE(card_properties, '[]')",
		"create_at",
		"update_at",
		"delete_at",
	}

	if prefix == "" {
		return fields
	}

	prefixedFields := make([]string, len(fields))
	for i, field := range fields {
		if strings.HasPrefix(field, "COALESCE(") {
			prefixedFields[i] = strings.Replace(field, "COALESCE(", "COALESCE("+prefix, 1)
		} else {
			prefixedFields[i] = prefix + field
		}
	}
	return prefixedFields
}

func (s *MattermostAuthLayer) baseUserQuery(showEmail, showName bool) sq.SelectBuilder {
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

	return s.getQueryBuilder().
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

// SearchBoardsForUser returns all boards that match with the
// term that are either private and which the user is a member of, or
// they're open, regardless of the user membership.
// Search is case-insensitive.
func (s *MattermostAuthLayer) SearchBoardsForUser(term string, searchField model.BoardSearchField, userID string, includePublicBoards bool) ([]*model.Board, error) {
	// as we're joining three queries, we need to avoid numbered
	// placeholders until the join is done, so we use the default
	// question mark placeholder here
	builder := s.getQueryBuilder().PlaceholderFormat(sq.Question)

	boardMembersQ := builder.
		Select(boardFields("b.")...).
		From(s.tablePrefix + "boards as b").
		Join(s.tablePrefix + "board_members as bm on b.id=bm.board_id").
		Where(sq.Eq{
			"b.is_template": false,
			"bm.user_id":    userID,
		})

	teamMembersQ := builder.
		Select(boardFields("b.")...).
		From(s.tablePrefix + "boards as b").
		Join("TeamMembers as tm on tm.teamid=b.team_id").
		Where(sq.Eq{
			"b.is_template": false,
			"tm.userID":     userID,
			"tm.deleteAt":   0,
			"b.type":        model.BoardTypeOpen,
		})

	channelMembersQ := builder.
		Select(boardFields("b.")...).
		From(s.tablePrefix + "boards as b").
		Join("ChannelMembers as cm on cm.channelId=b.channel_id").
		Where(sq.Eq{
			"b.is_template": false,
			"cm.userId":     userID,
		})

	if term != "" {
		if searchField == model.BoardSearchFieldPropertyName {
			var where, whereTerm string
			switch s.dbType {
			case model.PostgresDBType:
				where = "b.properties->? is not null"
				whereTerm = term
			case model.MysqlDBType, model.SqliteDBType:
				where = "JSON_EXTRACT(b.properties, ?) IS NOT NULL"
				whereTerm = "$." + term
			default:
				where = "b.properties LIKE ?"
				whereTerm = "%\"" + term + "\"%"
			}
			boardMembersQ = boardMembersQ.Where(where, whereTerm)
			teamMembersQ = teamMembersQ.Where(where, whereTerm)
			channelMembersQ = channelMembersQ.Where(where, whereTerm)
		} else { // model.BoardSearchFieldTitle
			// break search query into space separated words
			// and search for all words.
			// This should later be upgraded to industrial-strength
			// word tokenizer, that uses much more than space
			// to break words.
			conditions := sq.And{}
			for _, word := range strings.Split(strings.TrimSpace(term), " ") {
				conditions = append(conditions, sq.Like{"lower(b.title)": "%" + strings.ToLower(word) + "%"})
			}

			boardMembersQ = boardMembersQ.Where(conditions)
			teamMembersQ = teamMembersQ.Where(conditions)
			channelMembersQ = channelMembersQ.Where(conditions)
		}
	}

	teamMembersSQL, teamMembersArgs, err := teamMembersQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SearchBoardsForUser error getting teamMembersSQL: %w", err)
	}

	channelMembersSQL, channelMembersArgs, err := channelMembersQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SearchBoardsForUser error getting channelMembersSQL: %w", err)
	}

	unionQ := boardMembersQ
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	// NOTE: theoretically, could do e.g. `isGuest := !includePublicBoards`
	// but that introduces some tight coupling + fragility
	if !user.IsGuest {
		unionQ = unionQ.
			Prefix("(").
			Suffix(") UNION ("+channelMembersSQL+")", channelMembersArgs...)
		if includePublicBoards {
			unionQ = unionQ.Suffix(" UNION ("+teamMembersSQL+")", teamMembersArgs...)
		}
	} else if includePublicBoards {
		unionQ = unionQ.
			Prefix("(").
			Suffix(") UNION ("+teamMembersSQL+")", teamMembersArgs...)
	}

	unionSQL, unionArgs, err := unionQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SearchBoardsForUser error getting unionSQL: %w", err)
	}

	// if we're using postgres or sqlite, we need to replace the
	// question mark placeholder with the numbered dollar one, now
	// that the full query is built
	if s.dbType == model.PostgresDBType || s.dbType == model.SqliteDBType {
		var rErr error
		unionSQL, rErr = sq.Dollar.ReplacePlaceholders(unionSQL)
		if rErr != nil {
			return nil, fmt.Errorf("SearchBoardsForUser unable to replace unionSQL placeholders: %w", rErr)
		}
	}

	rows, err := s.mmDB.Query(unionSQL, unionArgs...)
	if err != nil {
		s.logger.Error(`searchBoardsForUser ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	return s.boardsFromRows(rows, false)
}

// searchBoardsForUserInTeam returns all boards that match with the
// term that are either private and which the user is a member of, or
// they're open, regardless of the user membership.
// Search is case-insensitive.
func (s *MattermostAuthLayer) SearchBoardsForUserInTeam(teamID, term, userID string) ([]*model.Board, error) {
	// as we're joining three queries, we need to avoid numbered
	// placeholders until the join is done, so we use the default
	// question mark placeholder here
	builder := s.getQueryBuilder().PlaceholderFormat(sq.Question)

	openBoardsQ := builder.
		Select(boardFields("b.")...).
		From(s.tablePrefix + "boards as b").
		Where(sq.Eq{
			"b.is_template": false,
			"b.team_id":     teamID,
			"b.type":        model.BoardTypeOpen,
		})

	memberBoardsQ := builder.
		Select(boardFields("b.")...).
		From(s.tablePrefix + "boards AS b").
		Join(s.tablePrefix + "board_members AS bm on b.id = bm.board_id").
		Where(sq.Eq{
			"b.is_template": false,
			"b.team_id":     teamID,
			"bm.user_id":    userID,
		})

	channelMemberBoardsQ := builder.
		Select(boardFields("b.")...).
		From(s.tablePrefix + "boards AS b").
		Join("ChannelMembers AS cm on cm.channelId = b.channel_id").
		Where(sq.Eq{
			"b.is_template": false,
			"b.team_id":     teamID,
			"cm.userId":     userID,
		})

	if term != "" {
		// break search query into space separated words
		// and search for all words.
		// This should later be upgraded to industrial-strength
		// word tokenizer, that uses much more than space
		// to break words.

		conditions := sq.And{}

		for _, word := range strings.Split(strings.TrimSpace(term), " ") {
			conditions = append(conditions, sq.Like{"lower(b.title)": "%" + strings.ToLower(word) + "%"})
		}

		openBoardsQ = openBoardsQ.Where(conditions)
		memberBoardsQ = memberBoardsQ.Where(conditions)
		channelMemberBoardsQ = channelMemberBoardsQ.Where(conditions)
	}

	memberBoardsSQL, memberBoardsArgs, err := memberBoardsQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SearchBoardsForUserInTeam error getting memberBoardsSQL: %w", err)
	}

	channelMemberBoardsSQL, channelMemberBoardsArgs, err := channelMemberBoardsQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SearchBoardsForUserInTeam error getting channelMemberBoardsSQL: %w", err)
	}

	unionQ := openBoardsQ.
		Prefix("(").
		Suffix(") UNION ("+memberBoardsSQL, memberBoardsArgs...).
		Suffix(") UNION ("+channelMemberBoardsSQL+")", channelMemberBoardsArgs...)

	unionSQL, unionArgs, err := unionQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SearchBoardsForUserInTeam error getting unionSQL: %w", err)
	}

	// if we're using postgres or sqlite, we need to replace the
	// question mark placeholder with the numbered dollar one, now
	// that the full query is built
	if s.dbType == model.PostgresDBType || s.dbType == model.SqliteDBType {
		var rErr error
		unionSQL, rErr = sq.Dollar.ReplacePlaceholders(unionSQL)
		if rErr != nil {
			return nil, fmt.Errorf("SearchBoardsForUserInTeam unable to replace unionSQL placeholders: %w", rErr)
		}
	}

	rows, err := s.mmDB.Query(unionSQL, unionArgs...)
	if err != nil {
		s.logger.Error(`searchBoardsForUserInTeam ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	return s.boardsFromRows(rows, false)
}

func (s *MattermostAuthLayer) boardsFromRows(rows *sql.Rows, removeDuplicates bool) ([]*model.Board, error) {
	boards := []*model.Board{}
	idMap := make(map[string]struct{})

	for rows.Next() {
		var board model.Board
		var propertiesBytes []byte
		var cardPropertiesBytes []byte

		err := rows.Scan(
			&board.ID,
			&board.TeamID,
			&board.ChannelID,
			&board.CreatedBy,
			&board.ModifiedBy,
			&board.Type,
			&board.MinimumRole,
			&board.Title,
			&board.Description,
			&board.Icon,
			&board.ShowDescription,
			&board.IsTemplate,
			&board.TemplateVersion,
			&propertiesBytes,
			&cardPropertiesBytes,
			&board.CreateAt,
			&board.UpdateAt,
			&board.DeleteAt,
		)
		if err != nil {
			s.logger.Error("boardsFromRows scan error", mlog.Err(err))
			return nil, err
		}

		if removeDuplicates {
			if _, ok := idMap[board.ID]; ok {
				continue
			} else {
				idMap[board.ID] = struct{}{}
			}
		}

		err = json.Unmarshal(propertiesBytes, &board.Properties)
		if err != nil {
			s.logger.Error("board properties unmarshal error", mlog.Err(err))
			return nil, err
		}
		err = json.Unmarshal(cardPropertiesBytes, &board.CardProperties)
		if err != nil {
			s.logger.Error("board card properties unmarshal error", mlog.Err(err))
			return nil, err
		}

		boards = append(boards, &board)
	}

	return boards, nil
}

func (s *MattermostAuthLayer) implicitBoardMembershipsFromRows(rows *sql.Rows) ([]*model.BoardMember, error) {
	boardMembers := []*model.BoardMember{}

	for rows.Next() {
		var boardMember model.BoardMember

		err := rows.Scan(
			&boardMember.UserID,
			&boardMember.BoardID,
		)
		if err != nil {
			return nil, err
		}
		boardMember.Roles = "editor"
		boardMember.SchemeEditor = true
		boardMember.Synthetic = true

		boardMembers = append(boardMembers, &boardMember)
	}

	return boardMembers, nil
}

func (s *MattermostAuthLayer) GetMemberForBoard(boardID, userID string) (*model.BoardMember, error) {
	bm, originalErr := s.Store.GetMemberForBoard(boardID, userID)
	// Explicit membership not found
	if model.IsErrNotFound(originalErr) {
		if userID == model.SystemUserID {
			return nil, model.NewErrNotFound(userID)
		}
		var user *model.User
		// No synthetic memberships for guests
		user, err := s.GetUserByID(userID)
		if err != nil {
			return nil, err
		}
		if user.IsGuest {
			return nil, model.NewErrNotFound("user is a guest")
		}

		b, boardErr := s.Store.GetBoard(boardID)
		if boardErr != nil {
			return nil, boardErr
		}
		if b.ChannelID != "" {
			_, memberErr := s.servicesAPI.GetChannelMember(b.ChannelID, userID)
			if memberErr != nil {
				var appErr *mmModel.AppError
				if errors.As(memberErr, &appErr) && appErr.StatusCode == http.StatusNotFound {
					// Plugin API returns error if channel member doesn't exist.
					// We're fine if it doesn't exist, so its not an error for us.
					message := fmt.Sprintf("member BoardID=%s UserID=%s", boardID, userID)
					return nil, model.NewErrNotFound(message)
				}

				return nil, memberErr
			}

			return &model.BoardMember{
				BoardID:         boardID,
				UserID:          userID,
				Roles:           "editor",
				SchemeAdmin:     false,
				SchemeEditor:    true,
				SchemeCommenter: false,
				SchemeViewer:    false,
				Synthetic:       true,
			}, nil
		}
		if b.Type == model.BoardTypeOpen && b.IsTemplate {
			_, memberErr := s.servicesAPI.GetTeamMember(b.TeamID, userID)
			if memberErr != nil {
				var appErr *mmModel.AppError
				if errors.As(memberErr, &appErr) && appErr.StatusCode == http.StatusNotFound {
					return nil, model.NewErrNotFound(userID)
				}
				return nil, memberErr
			}

			return &model.BoardMember{
				BoardID:         boardID,
				UserID:          userID,
				Roles:           "viewer",
				SchemeAdmin:     false,
				SchemeEditor:    false,
				SchemeCommenter: false,
				SchemeViewer:    true,
				Synthetic:       true,
			}, nil
		}
	}
	if originalErr != nil {
		return nil, originalErr
	}
	return bm, nil
}

func (s *MattermostAuthLayer) GetMembersForUser(userID string) ([]*model.BoardMember, error) {
	explicitMembers, err := s.Store.GetMembersForUser(userID)
	if err != nil {
		s.logger.Error(`getMembersForUser ERROR`, mlog.Err(err))
		return nil, err
	}

	query := s.getQueryBuilder().
		Select("CM.userID, B.Id").
		From(s.tablePrefix + "boards AS B").
		Join("ChannelMembers AS CM ON B.channel_id=CM.channelId").
		Where(sq.Eq{"CM.userID": userID})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getMembersForUser ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	members := []*model.BoardMember{}
	existingMembers := map[string]bool{}
	for _, m := range explicitMembers {
		members = append(members, m)
		existingMembers[m.BoardID] = true
	}

	// No synthetic memberships for guests
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user.IsGuest {
		return members, nil
	}

	implicitMembers, err := s.implicitBoardMembershipsFromRows(rows)
	if err != nil {
		return nil, err
	}
	for _, m := range implicitMembers {
		if !existingMembers[m.BoardID] {
			members = append(members, m)
		}
	}

	return members, nil
}

func (s *MattermostAuthLayer) GetMembersForBoard(boardID string) ([]*model.BoardMember, error) {
	explicitMembers, err := s.Store.GetMembersForBoard(boardID)
	if err != nil {
		s.logger.Error(`getMembersForBoard ERROR`, mlog.Err(err))
		return nil, err
	}

	query := s.getQueryBuilder().
		Select("CM.userID, B.Id").
		From(s.tablePrefix + "boards AS B").
		Join("ChannelMembers AS CM ON B.channel_id=CM.channelId").
		Join("Users as U on CM.userID = U.id").
		LeftJoin("Bots as bo on U.id = bo.UserID").
		Where(sq.Eq{"B.id": boardID}).
		Where(sq.NotEq{"B.channel_id": ""}).
		// Filter out guests as they don't have synthetic membership
		Where(sq.NotEq{"U.roles": "system_guest"}).
		Where(sq.Eq{"bo.UserId IS NOT NULL": false})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getMembersForBoard ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	implicitMembers, err := s.implicitBoardMembershipsFromRows(rows)
	if err != nil {
		return nil, err
	}
	members := []*model.BoardMember{}
	existingMembers := map[string]bool{}
	for _, m := range explicitMembers {
		members = append(members, m)
		existingMembers[m.UserID] = true
	}
	for _, m := range implicitMembers {
		if !existingMembers[m.UserID] {
			members = append(members, m)
		}
	}

	return members, nil
}

func (s *MattermostAuthLayer) GetBoardsForUserAndTeam(userID, teamID string, includePublicBoards bool) ([]*model.Board, error) {
	if includePublicBoards {
		boards, err := s.SearchBoardsForUserInTeam(teamID, "", userID)
		if err != nil {
			return nil, err
		}
		return boards, nil
	}

	// retrieve only direct memberships for user
	// this is usually done for guests.
	members, err := s.GetMembersForUser(userID)
	if err != nil {
		return nil, err
	}
	boardIDs := []string{}
	for _, m := range members {
		boardIDs = append(boardIDs, m.BoardID)
	}

	boards, err := s.Store.GetBoardsInTeamByIds(boardIDs, teamID)
	if model.IsErrNotFound(err) {
		if boards == nil {
			boards = []*model.Board{}
		}
		return boards, nil
	}
	if err != nil {
		return nil, err
	}

	return boards, nil
}

func (s *MattermostAuthLayer) SearchUserChannels(teamID, userID, query string) ([]*mmModel.Channel, error) {
	channels, err := s.servicesAPI.GetChannelsForTeamForUser(teamID, userID, false)
	if err != nil {
		return nil, err
	}
	lowerQuery := strings.ToLower(query)

	result := []*mmModel.Channel{}
	count := 0
	for _, channel := range channels {
		if channel.Type != mmModel.ChannelTypeDirect &&
			channel.Type != mmModel.ChannelTypeGroup &&
			(strings.Contains(strings.ToLower(channel.Name), lowerQuery) || strings.Contains(strings.ToLower(channel.DisplayName), lowerQuery)) {
			result = append(result, channel)
			count++
			if count >= 10 {
				break
			}
		}
	}
	return result, nil
}

func (s *MattermostAuthLayer) GetChannel(teamID, channelID string) (*mmModel.Channel, error) {
	channel, err := s.servicesAPI.GetChannelByID(channelID)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func (s *MattermostAuthLayer) getBoardsBotID() (string, error) {
	if boardsBotID == "" {
		var err error
		boardsBotID, err = s.servicesAPI.EnsureBot(model.FocalboardBot)
		if err != nil {
			s.logger.Error("failed to ensure boards bot", mlog.Err(err))
			return "", err
		}
	}
	return boardsBotID, nil
}

func (s *MattermostAuthLayer) SendMessage(message, postType string, receipts []string) error {
	botID, err := s.getBoardsBotID()
	if err != nil {
		return err
	}

	for _, receipt := range receipts {
		channel, err := s.servicesAPI.GetDirectChannel(botID, receipt)
		if err != nil {
			s.logger.Error(
				"failed to get DM channel between system bot and user for receipt",
				mlog.String("receipt", receipt),
				mlog.String("user_id", receipt),
				mlog.Err(err),
			)
			continue
		}

		if err := s.PostMessage(message, postType, channel.Id); err != nil {
			s.logger.Error(
				"failed to send message to receipt from SendMessage",
				mlog.String("receipt", receipt),
				mlog.Err(err),
			)
			continue
		}
	}
	return nil
}

func (s *MattermostAuthLayer) PostMessage(message, postType, channelID string) error {
	botID, err := s.getBoardsBotID()
	if err != nil {
		return err
	}

	post := &mmModel.Post{
		Message:   message,
		UserId:    botID,
		ChannelId: channelID,
		Type:      postType,
	}

	if _, err := s.servicesAPI.CreatePost(post); err != nil {
		s.logger.Error(
			"failed to send message to receipt from PostMessage",
			mlog.Err(err),
		)
	}
	return nil
}

func (s *MattermostAuthLayer) GetUserTimezone(userID string) (string, error) {
	user, err := s.servicesAPI.GetUserByID(userID)
	if err != nil {
		return "", err
	}
	timezone := user.Timezone
	return mmModel.GetPreferredTimezone(timezone), nil
}

func (s *MattermostAuthLayer) CanSeeUser(seerID string, seenID string) (bool, error) {
	mmuser, appErr := s.servicesAPI.GetUserByID(seerID)
	if appErr != nil {
		return false, appErr
	}
	if !mmuser.IsGuest() {
		return true, nil
	}

	query := s.getQueryBuilder().
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

	query = s.getQueryBuilder().
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
