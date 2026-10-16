// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState} from '@mattermost/types/store'
import {Channel} from '@mattermost/types/channels'
import {makeGetChannel} from 'mattermost-redux/selectors/entities/channels'

// Module-level so Plugin.initialize's mmStore.subscribe keeps one memoized selector.
const getChannel = makeGetChannel()

// Prefer the member-derived DM/GM display name over the server-truncated stored value.
export function resolveChannelForBoards(mmState: GlobalState, channelId: string | undefined): Channel | undefined {
    if (!channelId) {
        return undefined
    }
    return getChannel(mmState, channelId) || mmState.entities.channels.channels[channelId]
}
