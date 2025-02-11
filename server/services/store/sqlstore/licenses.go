// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	mmModel "github.com/mattermost/mattermost/server/public/model"
)

func (s *SQLStore) getLicense(_ sq.BaseRunner) *mmModel.License {
	return s.servicesAPI.GetLicense()
}
