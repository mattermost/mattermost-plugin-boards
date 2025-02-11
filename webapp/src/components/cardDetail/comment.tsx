// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {FC} from 'react'
import {useIntl} from 'react-intl'

import {getChannelsNameMapInTeam} from 'mattermost-redux/selectors/entities/channels'

import {Provider} from 'react-redux'

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
import {formatText, messageHtmlToComponent} from '../../webapp_globals'
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
    const channelNamesMap =  getChannelsNameMapInTeam((window as any).store.getState(), selectedTeam!.id)

    const formattedText = 
    <Provider store={(window as any).store}>
        {messageHtmlToComponent(formatText(comment.title, {
            atMentions: true,
            team: selectedTeam,
            channelNamesMap,
        }), {
            fetchMissingUsers: true, 
        })}
    </Provider>

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
            <div className='comment-markdown'>
                {formattedText}
            </div>
        </div>
    )
}

export default Comment
