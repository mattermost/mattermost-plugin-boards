// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState} from '@mattermost/types/store'
import {getCurrentChannel} from 'mattermost-redux/selectors/entities/channels'

import {Channel} from './store/channels'

// getCurrentMattermostChannel returns the active Mattermost channel for the
// Boards store. For group message channels the server stores display_name
// truncated to 64 characters, which drops members from large GMs. The
// mattermost-redux selector rebuilds the full member list from the loaded user
// profiles so the Boards RHS header lists every participant.
export function getCurrentMattermostChannel(state: GlobalState): Channel | undefined {
    return getCurrentChannel(state) as Channel | undefined
}

// createCurrentChannelSync builds a function that pushes the active Mattermost
// channel into the Boards store. Group message names are completed from member
// profiles that load asynchronously after the channel becomes active, so it
// re-emits whenever the resolved id or display name changes rather than only on
// channel switch.
export function createCurrentChannelSync(
    getState: () => GlobalState,
    onChannelChange: (channel: Channel) => void,
): () => void {
    let lastChannelId: string | undefined
    let lastChannelDisplayName: string | undefined

    return () => {
        const channel = getCurrentMattermostChannel(getState())
        if (channel && (channel.id !== lastChannelId || channel.display_name !== lastChannelDisplayName)) {
            lastChannelId = channel.id
            lastChannelDisplayName = channel.display_name
            onChannelChange(channel)
        }
    }
}
