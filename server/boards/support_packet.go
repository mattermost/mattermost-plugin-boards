// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package boards

import (
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type SupportPacket struct {
	Version string `yaml:"version"`
	// The total number of boards.
	TotalBoards int64 `yaml:"total_boards"`
	// The number of active boards.
	ActiveBoards int64 `yaml:"active_boards"`
	// The total number of cards.
	TotalCards int64 `yaml:"total_cards"`
	// The number of active cards.
	ActiveCards int64 `yaml:"active_cards"`
}

func (b *BoardsApp) GenerateSupportData(_ *plugin.Context) ([]*model.FileData, error) {
	var result *multierror.Error

	boardCount, err := b.server.Store().GetBoardCount(true)
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "failed to get total board count"))
	}
	activeBoardCount, err := b.server.Store().GetBoardCount(false)
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "failed to get active board count"))
	}

	cardsCount, err := b.server.Store().GetUsedCardsCount()
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "failed to get total cards count"))
	}
	usedCardsCount, err := b.server.Store().GetUsedCardsCount()
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "failed to get active cards count"))
	}

	diagnostics := SupportPacket{
		Version:      b.manifest.Version,
		TotalBoards:  boardCount,
		ActiveBoards: activeBoardCount,
		TotalCards:   cardsCount,
		ActiveCards:  usedCardsCount,
	}
	body, err := yaml.Marshal(diagnostics)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal diagnostics")
	}

	return []*model.FileData{{
		Filename: filepath.Join(b.manifest.Id, "diagnostics.yaml"),
		Body:     body,
	}}, result.ErrorOrNil()
}
