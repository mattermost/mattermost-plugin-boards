// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {useIntl} from 'react-intl'

import AttachmentElement from '../../components/content/attachmentElement'
import {AttachmentBlock} from '../../blocks/attachmentBlock'

import './attachment.scss'
import {Block} from '../../blocks/block'
import CompassIcon from '../../widgets/icons/compassIcon'
import BoardPermissionGate from '../../components/permissions/boardPermissionGate'
import {Permission} from '../../constants'

type Props = {
    attachments: AttachmentBlock[]
    onDelete: (block: Block) => void
    addAttachment: () => void
}

const AttachmentList = (props: Props): JSX.Element => {
    const {attachments, onDelete, addAttachment} = props
    const intl = useIntl()

    return (
        <div className='Attachment'>
            <div className='attachment-header'>
                <div className='attachment-title mb-2'>{intl.formatMessage({id: 'Attachment.Attachment-title', defaultMessage: 'Attachment'})} {`(${attachments.length})`}</div>
                <BoardPermissionGate permissions={[Permission.ManageBoardCards]}>
                    <div
                        className='attachment-plus-btn'
                        onClick={addAttachment}
                    >
                        <CompassIcon
                            icon='plus'
                            className='attachment-plus-icon'
                        />
                    </div>
                </BoardPermissionGate>
            </div>
            <div className='attachment-content'>
                {attachments.map((block: AttachmentBlock) => {
                    return (
                        <div key={block.id}>
                            <AttachmentElement
                                block={block}
                                onDelete={onDelete}
                            />
                        </div>)
                })
                }
            </div>
        </div>
    )
}

export default AttachmentList
