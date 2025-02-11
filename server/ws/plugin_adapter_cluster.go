// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ws

import (
	"encoding/json"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

type ClusterMessage struct {
	TeamID      string
	BoardID     string
	UserID      string
	Payload     map[string]interface{}
	EnsureUsers []string
}

func (pa *PluginAdapter) sendMessageToCluster(clusterMessage *ClusterMessage) {
	const id = "websocket_message"
	b, err := json.Marshal(clusterMessage)
	if err != nil {
		pa.logger.Error("couldn't get JSON bytes from cluster message",
			mlog.String("id", id),
			mlog.Err(err),
		)
		return
	}

	event := mmModel.PluginClusterEvent{Id: id, Data: b}
	opts := mmModel.PluginClusterEventSendOptions{
		SendType: mmModel.PluginClusterEventSendTypeReliable,
	}

	if err := pa.api.PublishPluginClusterEvent(event, opts); err != nil {
		pa.logger.Error("error publishing cluster event",
			mlog.String("id", id),
			mlog.Err(err),
		)
	}
}

func (pa *PluginAdapter) HandleClusterEvent(ev mmModel.PluginClusterEvent) {
	pa.logger.Debug("received cluster event", mlog.String("id", ev.Id))

	var clusterMessage ClusterMessage
	if err := json.Unmarshal(ev.Data, &clusterMessage); err != nil {
		pa.logger.Error("cannot unmarshal cluster message data",
			mlog.String("id", ev.Id),
			mlog.Err(err),
		)
		return
	}

	if clusterMessage.BoardID != "" {
		pa.sendBoardMessageSkipCluster(clusterMessage.TeamID, clusterMessage.BoardID, clusterMessage.Payload, clusterMessage.EnsureUsers...)
		return
	}

	var action string
	if actionRaw, ok := clusterMessage.Payload["action"]; ok {
		if s, ok := actionRaw.(string); ok {
			action = s
		}
	}
	if action == "" {
		// no action was specified in the event; assume block change and warn.
		pa.logger.Warn("cannot determine action from cluster message data",
			mlog.String("id", ev.Id),
			mlog.Map("payload", clusterMessage.Payload),
		)
		return
	}

	if clusterMessage.UserID != "" {
		pa.sendUserMessageSkipCluster(action, clusterMessage.Payload, clusterMessage.UserID)
		return
	}

	pa.sendTeamMessageSkipCluster(action, clusterMessage.TeamID, clusterMessage.Payload)
}
