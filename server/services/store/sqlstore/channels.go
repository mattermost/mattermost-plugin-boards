package sqlstore

import (
	"strings"

	mmModel "github.com/mattermost/mattermost/server/public/model"
)

func (s *SQLStore) SearchUserChannels(teamID, userID, query string) ([]*mmModel.Channel, error) {
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

func (s *SQLStore) GetChannel(teamID, channelID string) (*mmModel.Channel, error) {
	channel, err := s.servicesAPI.GetChannelByID(channelID)
	if err != nil {
		return nil, err
	}
	return channel, nil
}
