// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {FormattedMessage} from 'react-intl'

import Button from '../widgets/buttons/button'

import Dialog from './dialog'
import './viewVisibilityDialog.scss'

type ViewVisibilityDialogProps = {
    onPublic: () => void
    onPersonal: () => void
    onClose: () => void
}

type Props = {
    dialogBox: ViewVisibilityDialogProps
}

export const ViewVisibilityDialog = (props: Props) => {
    const {onClose, onPublic, onPersonal} = props.dialogBox

    return (
        <Dialog
            size='small'
            className='view-visibility-dialog'
            onClose={onClose}
        >
            <div
                className='box-area'
                title='View Visibility Dialog'
            >
                <h3 className='text-heading5'>
                    <FormattedMessage
                        id='ViewVisibilityDialog.heading'
                        defaultMessage='Choose view visibility'
                    />
                </h3>
                <div className='sub-text'>
                    <FormattedMessage
                        id='ViewVisibilityDialog.subText'
                        defaultMessage='Public views are visible to all team members. Personal views are only visible to you.'
                    />
                </div>

                <div className='action-buttons'>
                    <Button
                        title='Cancel'
                        size='medium'
                        emphasis='tertiary'
                        onClick={onClose}
                    >
                        <FormattedMessage
                            id='ViewVisibilityDialog.cancel-action'
                            defaultMessage='Cancel'
                        />
                    </Button>
                    <Button
                        title='Personal'
                        size='medium'
                        onClick={onPersonal}
                        filled={false}
                    >
                        <FormattedMessage
                            id='ViewVisibilityDialog.personal-action'
                            defaultMessage='Personal'
                        />
                    </Button>
                    <Button
                        title='Public'
                        size='medium'
                        onClick={onPublic}
                        filled={true}
                    >
                        <FormattedMessage
                            id='ViewVisibilityDialog.public-action'
                            defaultMessage='Public'
                        />
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default ViewVisibilityDialog
export {ViewVisibilityDialogProps}

