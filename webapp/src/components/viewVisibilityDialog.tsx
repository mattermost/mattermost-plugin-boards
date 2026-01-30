// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
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
    const handleOnClose = useCallback(props.dialogBox.onClose, [])
    const handleOnPublic = useCallback(props.dialogBox.onPublic, [])
    const handleOnPersonal = useCallback(props.dialogBox.onPersonal, [])

    return (
        <Dialog
            size='small'
            className='view-visibility-dialog'
            onClose={handleOnClose}
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
                        onClick={handleOnClose}
                    >
                        <FormattedMessage
                            id='ViewVisibilityDialog.cancel-action'
                            defaultMessage='Cancel'
                        />
                    </Button>
                    <Button
                        title='Personal'
                        size='medium'
                        onClick={handleOnPersonal}
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
                        onClick={handleOnPublic}
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

