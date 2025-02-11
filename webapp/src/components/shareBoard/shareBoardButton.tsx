// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React, {useState} from 'react'
import {FormattedMessage} from 'react-intl'

import Button from '../../widgets/buttons/button'
import TelemetryClient, {TelemetryActions, TelemetryCategory} from '../../telemetry/telemetryClient'
import {useAppSelector} from '../../store/hooks'
import {getCurrentBoard} from '../../store/boards'
import Globe from '../../widgets/icons/globe'
import LockOutline from '../../widgets/icons/lockOutline'
import {BoardTypeOpen} from '../../blocks/board'

import './shareBoardButton.scss'

import ShareBoardDialog from './shareBoard'

type Props = {
    enableSharedBoards: boolean
}
const ShareBoardButton = (props: Props) => {
    const [showShareDialog, setShowShareDialog] = useState(false)
    const board = useAppSelector(getCurrentBoard)

    const iconForBoardType = () => {
        if (board.type === BoardTypeOpen) {
            return <Globe/>
        }
        return <LockOutline/>
    }

    return (
        <div className='ShareBoardButton'>
            <Button
                title='Share board'
                size='medium'
                emphasis='primary'
                icon={iconForBoardType()}
                onClick={() => {
                    TelemetryClient.trackEvent(TelemetryCategory, TelemetryActions.ShareBoardOpenModal, {board: board.id})
                    setShowShareDialog(!showShareDialog)
                }}
            >
                <FormattedMessage
                    id='CenterPanel.Share'
                    defaultMessage='Share'
                />
            </Button>
            {showShareDialog &&
                <ShareBoardDialog
                    onClose={() => setShowShareDialog(false)}
                    enableSharedBoards={props.enableSharedBoards}
                />}
        </div>
    )
}

export default React.memo(ShareBoardButton)
