// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
import React, {FC} from 'react'
import {useIntl} from 'react-intl'

import {getChannelsNameMapInTeam} from 'mattermost-redux/selectors/entities/channels'

import {Provider} from 'react-redux'

import {Team} from '@mattermost/types/teams'

import {Channel} from '@mattermost/types/channels'

import {useFormatTextToComponent, useWebAppSelector, useWebAppStore} from '@hmhealey/plugin-support'

import {Block} from '../../blocks/block'
import mutator from '../../mutator'
import {Utils} from '../../utils'
import IconButton from '../../widgets/buttons/iconButton'
import DeleteIcon from '../../widgets/icons/delete'
import OptionsIcon from '../../widgets/icons/options'
import Menu from '../../widgets/menu'
import MenuWrapper from '../../widgets/menuWrapper'
import {getUser} from '../../store/users'
import {useAppSelector} from '../../store/hooks'
import Tooltip from '../../widgets/tooltip'
import GuestBadge from '../../widgets/guestBadge'

import './comment.scss'
import {getCurrentTeam} from '../../store/teams'


type Props = {
    comment: Block
    userId: string
    userImageUrl: string
    readonly: boolean
}

const Comment: FC<Props> = (props: Props) => {
    const {comment, userId, userImageUrl} = props
    const intl = useIntl()
    const user = useAppSelector(getUser(userId))
    const date = new Date(comment.createAt)

    const selectedTeam = useAppSelector(getCurrentTeam)

    /*
     * TODO This doesn't currently work because Boards paves over the web app's Redux context. It seems like
     * React Redux has some way to apply a custom context to Boards' selectors to stop that from happening,
     * but if I can't get that working, I'll need to add the web app's React Redux context to the PluginContext.
     *
     * Also, if I'm going to keep pushing people to using the old version of mattermost-redux, we need to find
     * a way to have the plugin support package override its version of GlobalState...
     */

    const channelNamesMap = useWebAppSelector((state) => getChannelsNameMapInTeam(state, selectedTeam!.id))

    const formatTextToComponent = useFormatTextToComponent({
        atMentions: true,
        mentionHighlight: false,
        team: (selectedTeam || undefined) as Team | undefined,
        channelNamesMap: channelNamesMap as unknown as Record<string, Channel>,
        fetchMissingUsers: true,
    })
    const formattedText = (
        <Provider store={useWebAppStore()}>
            {formatTextToComponent(comment.title)}
        </Provider>
    )

    return (
        <div
            key={comment.id}
            className='Comment comment'
        >
            <div className='comment-header'>
                <img
                    className='comment-avatar'
                    src={userImageUrl}
                />
                <div className='comment-username'>{user?.username}</div>
                <GuestBadge show={user?.is_guest}/>

                <Tooltip title={Utils.displayDateTime(date, intl)}>
                    <div className='comment-date'>
                        {Utils.relativeDisplayDateTime(date, intl)}
                    </div>
                </Tooltip>

                {!props.readonly && (
                    <MenuWrapper>
                        <IconButton icon={<OptionsIcon/>}/>
                        <Menu position='left'>
                            <Menu.Text
                                icon={<DeleteIcon/>}
                                id='delete'
                                name={intl.formatMessage({id: 'Comment.delete', defaultMessage: 'Delete'})}
                                onClick={() => mutator.deleteBlock(comment)}
                            />
                        </Menu>
                    </MenuWrapper>
                )}
            </div>
            {formattedText}
        </div>
    )
}

export default Comment
