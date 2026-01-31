// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React, {ReactNode} from 'react'
import {FormattedMessage} from 'react-intl'

import Button from '../widgets/buttons/button'

import Dialog from './dialog'
import './confirmationDialogBox.scss'

type ConfirmationDialogBoxProps = {
    heading: string
    subText?: string | ReactNode
    confirmButtonText?: string
    destructive?: boolean
    onConfirm: () => void
    onClose: () => void
}

type Props = {
    dialogBox: ConfirmationDialogBoxProps
}

export const ConfirmationDialogBox = (props: Props) => {
    const {onClose, onConfirm} = props.dialogBox

    return (
        <Dialog
            size='small'
            className='confirmation-dialog-box'
            onClose={onClose}
        >
            <div
                className='box-area'
                title='Confirmation Dialog Box'
            >
                <h3 className='text-heading5'>{props.dialogBox.heading}</h3>
                <div className='sub-text'>{props.dialogBox.subText}</div>

                <div className='action-buttons'>
                    <Button
                        title='Cancel'
                        size='medium'
                        emphasis='tertiary'
                        onClick={onClose}
                    >
                        <FormattedMessage
                            id='ConfirmationDialog.cancel-action'
                            defaultMessage='Cancel'
                        />
                    </Button>
                    <Button
                        title={props.dialogBox.confirmButtonText || 'Confirm'}
                        size='medium'
                        submit={true}
                        danger={Boolean(props.dialogBox.destructive)}
                        onClick={onConfirm}
                        filled={true}
                    >
                        { props.dialogBox.confirmButtonText ||
                        <FormattedMessage
                            id='ConfirmationDialog.confirm-action'
                            defaultMessage='Confirm'
                        />
                        }
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default ConfirmationDialogBox
export {ConfirmationDialogBoxProps}
