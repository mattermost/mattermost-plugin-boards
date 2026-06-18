// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import {useAppSelector} from '../../store/hooks'
import {getLastCardContent} from '../../store/contents'
import {getLastCardComment} from '../../store/comments'
import {getLatestUpdatedBlock} from '../latestUpdatedBlock'
import Person from '../person/person'

import {PropertyProps} from '../types'

const LastModifiedBy = (props: PropertyProps): JSX.Element => {
    const lastContent = useAppSelector(getLastCardContent(props.card.id || ''))
    const lastComment = useAppSelector(getLastCardComment(props.card.id))

    const latestBlock = props.board ?
        getLatestUpdatedBlock(props.card, lastContent, lastComment) :
        props.card

    return (
        <Person
            {...props}
            propertyValue={latestBlock.modifiedBy}
            readOnly={true} // created by is an immutable property, so will always be readonly
        />
    )
}

export default LastModifiedBy
