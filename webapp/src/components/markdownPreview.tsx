// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useMemo} from 'react'

import {Channel} from '@mattermost/types/channels'
import {getChannelsNameMapInTeam} from 'mattermost-redux/selectors/entities/channels'

import {Provider} from 'react-redux'

import {formatText, messageHtmlToComponent} from '../webapp_globals'
import {getCurrentTeam} from '../store/teams'
import {useAppSelector} from '../store/hooks'
import {Utils} from '../utils'

const EMPTY_CHANNEL_NAMES_MAP: Record<string, Channel> = {}

type Props = {
    text: string
}

// Renders markdown through the Mattermost webapp renderer when running inside
// Mattermost (where window.store and window.PostUtils are available) and falls
// back to the Boards-native renderer for standalone/published boards, where
// neither exists. Wrapping the webapp renderer in <Provider store={undefined}>
// would throw during render and blank the whole shared-board page.
const MarkdownPreview = (props: Props): React.JSX.Element => {
    const {text} = props
    const selectedTeam = useAppSelector(getCurrentTeam)
    const windowStore = (window as any).store

    const channelNamesMap = useMemo(() => {
        if (!selectedTeam || !windowStore) {
            return EMPTY_CHANNEL_NAMES_MAP
        }
        return getChannelsNameMapInTeam(windowStore.getState(), selectedTeam.id)
    }, [selectedTeam?.id, windowStore])

    if (!windowStore) {
        return (
            <div dangerouslySetInnerHTML={{__html: Utils.htmlFromMarkdown(text)}}/>
        )
    }

    return (
        <Provider store={windowStore}>
            {messageHtmlToComponent(formatText(text, {
                atMentions: true,
                team: selectedTeam,
                channelNamesMap,
            }), {
                fetchMissingUsers: true,
            })}
        </Provider>
    )
}

export {MarkdownPreview}
export default MarkdownPreview
