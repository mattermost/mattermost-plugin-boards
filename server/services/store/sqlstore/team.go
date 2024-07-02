package sqlstore

import (
	"database/sql"
	"encoding/json"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/mattermost/mattermost/server/public/shared/mlog"

	sq "github.com/Masterminds/squirrel"
)

var (
	teamFields = []string{
		"id",
		"signup_token",
		"COALESCE(settings, '{}')",
		"modified_by",
		"update_at",
	}
)

func (s *SQLStore) upsertTeamSignupToken(db sq.BaseRunner, team model.Team) error {
	now := utils.GetMillis()

	query := s.getQueryBuilder(db).
		Insert(s.tablePrefix+"teams").
		Columns(
			"id",
			"signup_token",
			"modified_by",
			"update_at",
		).
		Values(
			team.ID,
			team.SignupToken,
			team.ModifiedBy,
			now,
		)
	if s.dbType == model.MysqlDBType {
		query = query.Suffix("ON DUPLICATE KEY UPDATE signup_token = ?, modified_by = ?, update_at = ?",
			team.SignupToken, team.ModifiedBy, now)
	} else {
		query = query.Suffix(
			`ON CONFLICT (id)
			 DO UPDATE SET signup_token = EXCLUDED.signup_token, modified_by = EXCLUDED.modified_by, update_at = EXCLUDED.update_at`,
		)
	}

	_, err := query.Exec()
	return err
}

func (s *SQLStore) upsertTeamSettings(db sq.BaseRunner, team model.Team) error {
	now := utils.GetMillis()
	signupToken := utils.NewID(utils.IDTypeToken)

	settingsJSON, err := json.Marshal(team.Settings)
	if err != nil {
		return err
	}

	query := s.getQueryBuilder(db).
		Insert(s.tablePrefix+"teams").
		Columns(
			"id",
			"signup_token",
			"settings",
			"modified_by",
			"update_at",
		).
		Values(
			team.ID,
			signupToken,
			settingsJSON,
			team.ModifiedBy,
			now,
		)
	if s.dbType == model.MysqlDBType {
		query = query.Suffix("ON DUPLICATE KEY UPDATE settings = ?, modified_by = ?, update_at = ?", settingsJSON, team.ModifiedBy, now)
	} else {
		query = query.Suffix(
			`ON CONFLICT (id)
			 DO UPDATE SET settings = EXCLUDED.settings, modified_by = EXCLUDED.modified_by, update_at = EXCLUDED.update_at`,
		)
	}

	_, err = query.Exec()
	return err
}

func (s *SQLStore) getTeamCount(db sq.BaseRunner) (int64, error) {
	query := s.getQueryBuilder(db).
		Select(
			"COUNT(*) AS count",
		).
		From(s.tablePrefix + "teams")

	rows, err := query.Query()
	if err != nil {
		s.logger.Error("ERROR GetTeamCount", mlog.Err(err))
		return 0, err
	}
	defer s.CloseRows(rows)

	var count int64

	rows.Next()
	err = rows.Scan(&count)
	if err != nil {
		s.logger.Error("Failed to fetch team count", mlog.Err(err))
		return 0, err
	}
	return count, nil
}

func (s *SQLStore) teamsFromRows(rows *sql.Rows) ([]*model.Team, error) {
	teams := []*model.Team{}

	for rows.Next() {
		var team model.Team
		var settingsBytes []byte

		err := rows.Scan(
			&team.ID,
			&team.SignupToken,
			&settingsBytes,
			&team.ModifiedBy,
			&team.UpdateAt,
		)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(settingsBytes, &team.Settings)
		if err != nil {
			return nil, err
		}

		teams = append(teams, &team)
	}

	return teams, nil
}

func (s *SQLStore) getAllTeams(db sq.BaseRunner) ([]*model.Team, error) {
	query := s.getQueryBuilder(db).
		Select(teamFields...).
		From(s.tablePrefix + "teams")
	rows, err := query.Query()
	if err != nil {
		s.logger.Error("ERROR GetAllTeams", mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	teams, err := s.teamsFromRows(rows)
	if err != nil {
		return nil, err
	}

	return teams, nil
}
