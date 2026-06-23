// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import {useIntl} from 'react-intl'

import {Utils} from '../../utils'
import {useAppSelector} from '../../store/hooks'
import {getLastCardContent} from '../../store/contents'
import {getLastCardComment} from '../../store/comments'
import {getLatestUpdatedBlock} from '../latestUpdatedBlock'
import './updatedTime.scss'

import {PropertyProps} from '../types'

const UpdatedTime = (props: PropertyProps): JSX.Element => {
    const intl = useIntl()
    const lastContent = useAppSelector(getLastCardContent(props.card.id || ''))
    const lastComment = useAppSelector(getLastCardComment(props.card.id))

    const latestBlock = props.card ?
        getLatestUpdatedBlock(props.card, lastContent, lastComment) :
        props.card

    return (
        <div className={`UpdatedTime ${props.property.valueClassName(true)}`}>
            {Utils.displayDateTime(new Date(latestBlock.updateAt), intl)}
        </div>
    )
}

export default UpdatedTime
