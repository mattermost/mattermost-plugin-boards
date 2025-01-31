// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (s *SQLStore) getBoardsBotID() (string, error) {
	var boardsBotID string
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

func (s *SQLStore) sendMessage(db sq.BaseRunner, message, postType string, receipts []string) error {
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

		if err := s.postMessage(db, message, postType, channel.Id); err != nil {
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

func (s *SQLStore) postMessage(_ sq.BaseRunner, message, postType, channelID string) error {
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
