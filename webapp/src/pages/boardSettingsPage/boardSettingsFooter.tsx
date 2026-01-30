// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback, useRef, useEffect} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Board} from '../../blocks/board'
import Button from '../../widgets/buttons/button'
import ConfirmationDialogBox from '../../components/confirmationDialogBox'

import './boardSettingsFooter.scss'

type Props = {
    board: Board
    isHidden: boolean
    onHideBoard: () => Promise<void>
    onShowBoard: () => Promise<void>
    onDeleteBoard: () => Promise<void>
    onCancel: () => void
    onSave: () => void
}

const BoardSettingsFooter = (props: Props): JSX.Element => {
    const {board, isHidden} = props
    const intl = useIntl()

    const [showHideConfirmation, setShowHideConfirmation] = useState(false)
    const [showDeleteConfirmation, setShowDeleteConfirmation] = useState(false)
    const [isProcessing, setIsProcessing] = useState(false)

    // Guard against state updates after unmount (hide/delete navigate away)
    const mountedRef = useRef(true)
    useEffect(() => {
        return () => {
            mountedRef.current = false
        }
    }, [])

    const handleHideBoard = useCallback(async () => {
        setIsProcessing(true)
        try {
            await props.onHideBoard()
            if (mountedRef.current) {
                setShowHideConfirmation(false)
            }
        } finally {
            if (mountedRef.current) {
                setIsProcessing(false)
            }
        }
    }, [props])

    const handleShowBoard = useCallback(async () => {
        setIsProcessing(true)
        try {
            await props.onShowBoard()
        } finally {
            if (mountedRef.current) {
                setIsProcessing(false)
            }
        }
    }, [props])

    const handleDeleteBoard = useCallback(async () => {
        setIsProcessing(true)
        try {
            await props.onDeleteBoard()
            if (mountedRef.current) {
                setShowDeleteConfirmation(false)
            }
        } finally {
            if (mountedRef.current) {
                setIsProcessing(false)
            }
        }
    }, [props])

    return (
        <>
            <div className='BoardSettingsFooter'>
                <div className='BoardSettingsFooter__actions'>
                    {/* Hide/Show Board */}
                    {isHidden ? (
                        <Button
                            emphasis='tertiary'
                            size='medium'
                            onClick={handleShowBoard}
                            disabled={isProcessing}
                        >
                            <FormattedMessage
                                id='BoardSettings.footer.show-board'
                                defaultMessage='Show Board'
                            />
                        </Button>
                    ) : (
                        <Button
                            emphasis='tertiary'
                            size='medium'
                            onClick={() => setShowHideConfirmation(true)}
                            disabled={isProcessing}
                        >
                            <FormattedMessage
                                id='BoardSettings.footer.hide-board'
                                defaultMessage='Hide Board'
                            />
                        </Button>
                    )}

                    {/* Delete Board */}
                    <Button
                        emphasis='tertiary'
                        size='medium'
                        danger={true}
                        onClick={() => setShowDeleteConfirmation(true)}
                        disabled={isProcessing}
                    >
                        <FormattedMessage
                            id='BoardSettings.footer.delete-board'
                            defaultMessage='Delete Board'
                        />
                    </Button>
                </div>

                <div className='BoardSettingsFooter__main-actions'>
                    <Button
                        emphasis='tertiary'
                        size='medium'
                        onClick={props.onCancel}
                        disabled={isProcessing}
                    >
                        <FormattedMessage
                            id='BoardSettings.cancel'
                            defaultMessage='Cancel'
                        />
                    </Button>
                    <Button
                        filled={true}
                        size='medium'
                        onClick={props.onSave}
                        disabled={isProcessing}
                    >
                        <FormattedMessage
                            id='BoardSettings.save'
                            defaultMessage='Save'
                        />
                    </Button>
                </div>
            </div>

            {/* Hide Board Confirmation Dialog */}
            {showHideConfirmation && (
                <ConfirmationDialogBox
                    dialogBox={{
                        heading: intl.formatMessage({
                            id: 'BoardSettings.hide-board-confirmation.heading',
                            defaultMessage: 'Hide board?',
                        }),
                        subText: intl.formatMessage({
                            id: 'BoardSettings.hide-board-confirmation.subtext',
                            defaultMessage: 'Are you sure you want to hide "{boardTitle}"? You can show it again from the sidebar.',
                        }, {boardTitle: board.title || intl.formatMessage({id: 'ViewTitle.untitled-board', defaultMessage: 'Untitled board'})}),
                        confirmButtonText: intl.formatMessage({
                            id: 'BoardSettings.hide-board-confirmation.confirm',
                            defaultMessage: 'Hide',
                        }),
                        onConfirm: handleHideBoard,
                        onClose: () => setShowHideConfirmation(false),
                    }}
                />
            )}

            {/* Delete Board Confirmation Dialog */}
            {showDeleteConfirmation && (
                <ConfirmationDialogBox
                    dialogBox={{
                        heading: intl.formatMessage({
                            id: 'BoardSettings.delete-board-confirmation.heading',
                            defaultMessage: 'Delete board?',
                        }),
                        subText: intl.formatMessage({
                            id: 'BoardSettings.delete-board-confirmation.subtext',
                            defaultMessage: 'Are you sure you want to delete "{boardTitle}"? This action cannot be undone.',
                        }, {boardTitle: board.title || intl.formatMessage({id: 'ViewTitle.untitled-board', defaultMessage: 'Untitled board'})}),
                        confirmButtonText: intl.formatMessage({
                            id: 'BoardSettings.delete-board-confirmation.confirm',
                            defaultMessage: 'Delete',
                        }),
                        destructive: true,
                        onConfirm: handleDeleteBoard,
                        onClose: () => setShowDeleteConfirmation(false),
                    }}
                />
            )}
        </>
    )
}

export default React.memo(BoardSettingsFooter)

