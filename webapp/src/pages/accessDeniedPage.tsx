// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {useHistory, useLocation} from 'react-router-dom'
import {FormattedMessage, useIntl} from 'react-intl'

import Button from '../widgets/buttons/button'
import './accessDeniedPage.scss'

import {setGlobalError} from '../store/globalError'
import {useAppDispatch} from '../store/hooks'
import {UserSettings} from '../userSettings'
import AccessDeniedIllustration from '../svg/access-denied-illustation'

const clearLastBoardAndViewIds = () => {
    const lastTeamId = UserSettings.lastTeamId
    if (lastTeamId) {
        const lastBoardId = UserSettings.lastBoardId[lastTeamId]
        if (lastBoardId) {
            UserSettings.setLastViewId(lastBoardId, null)
        }
        UserSettings.setLastBoardID(lastTeamId, null)
    }
}

const AccessDeniedPage = () => {
    const history = useHistory()
    const location = useLocation()
    const intl = useIntl()
    const dispatch = useAppDispatch()

    const handleBackToHome = useCallback(() => {
        // Clear any error state before navigating
        dispatch(setGlobalError(''))

        // Clear last board/view IDs as a precaution to prevent redirect loops.
        const referrerPath = new URLSearchParams(location.search).get('r')
        if (referrerPath) {
            try {
                const pathParts = decodeURIComponent(referrerPath).split('/').filter(Boolean)
                if (pathParts[0] === 'team' && pathParts.length >= 3) {
                    const teamId = pathParts[1]
                    const boardId = pathParts[2]
                    UserSettings.setLastBoardID(teamId, null)
                    UserSettings.setLastViewId(boardId, null)
                }
            } catch {
                clearLastBoardAndViewIds()
            }
        } else {
            clearLastBoardAndViewIds()
        }

        // Always navigate to home, not back to the board that caused the error
        history.push('/')
    }, [history, dispatch, location.search])

    return (
        <div className='AccessDeniedPage'>
            <div>
                <AccessDeniedIllustration/>
                <div className='title'>
                    <FormattedMessage
                        id='accessDenied.page.title'
                        defaultMessage={'You don’t have access to this board'}
                    />
                </div>
                <div className='subtitle'>
                    <FormattedMessage
                        id='accessDenied.page.subtitle'
                        defaultMessage={'This board is private or has restricted permissions.'}
                    />
                </div>
                <br/>
                <Button
                    filled={true}
                    size='large'
                    onClick={handleBackToHome}
                >
                    {intl.formatMessage({id: 'accessDenied.back-to-home', defaultMessage: 'Back to your boards'})}
                </Button>
            </div>
        </div>
    )
}

export default React.memo(AccessDeniedPage)
