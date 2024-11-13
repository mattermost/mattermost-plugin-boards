// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package boards

import (
	"path"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type SupportPacket struct {
	Version string `yaml:"version"`
	// The total number of active boards.
	ActiveBoards int64 `yaml:"active_boards"`
	// The total number of active cards.
	ActiveCards int64 `yaml:"active_cards"`
}

func (b *BoardsApp) GenerateSupportData(_ *plugin.Context) ([]*model.FileData, error) {
	var result *multierror.Error

	boardCount, err := b.server.Store().GetBoardCount()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get boards count")
	}
	usedCardsCount, err := b.server.Store().GetUsedCardsCount()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get used cards count")
	}

	diagnostics := SupportPacket{
		Version:      b.manifest.Version,
		ActiveBoards: boardCount,
		ActiveCards:  int64(usedCardsCount),
	}
	data, err := yaml.Marshal(diagnostics)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal diagnostics")
	}

	return []*model.FileData{{
		Filename: path.Join(b.manifest.Id, "diagnostics.yaml"),
		Body:     data,
	}}, result.ErrorOrNil()
}
