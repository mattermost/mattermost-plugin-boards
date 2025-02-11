// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import {useIntl} from 'react-intl'

import {Block} from '../../blocks/block'
import {Utils} from '../../utils'
import {useAppSelector} from '../../store/hooks'
import {getLastCardContent} from '../../store/contents'
import {getLastCardComment} from '../../store/comments'
import './updatedTime.scss'

import {PropertyProps} from '../types'

const UpdatedTime = (props: PropertyProps): JSX.Element => {
    const intl = useIntl()
    const lastContent = useAppSelector(getLastCardContent(props.card.id || '')) as Block
    const lastComment = useAppSelector(getLastCardComment(props.card.id)) as Block

    let latestBlock: Block = props.card
    if (props.card) {
        const allBlocks = [props.card, lastContent, lastComment]
        const sortedBlocks = allBlocks.sort((a, b) => b.updateAt - a.updateAt)

        latestBlock = sortedBlocks.length > 0 ? sortedBlocks[0] : latestBlock
    }

    return (
        <div className={`UpdatedTime ${props.property.valueClassName(true)}`}>
            {Utils.displayDateTime(new Date(latestBlock.updateAt), intl)}
        </div>
    )
}

export default UpdatedTime
