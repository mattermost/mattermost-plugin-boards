package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	mmModel "github.com/mattermost/mattermost/server/public/model"
)

func (s *SQLStore) getLicense(_ sq.BaseRunner) *mmModel.License {
	return s.servicesAPI.GetLicense()
}
