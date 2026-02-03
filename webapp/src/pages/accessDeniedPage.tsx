// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {useHistory} from 'react-router-dom'
import {FormattedMessage, useIntl} from 'react-intl'


import Button from '../widgets/buttons/button'
import './accessDeniedPage.scss'

import {setGlobalError} from '../store/globalError'
import {useAppDispatch} from '../store/hooks'
import AccessDeniedIllustration from '../svg/access-denied-illustation'

const AccessDeniedPage = () => {
    const history = useHistory()
    const intl = useIntl()
    const dispatch = useAppDispatch()

    const handleBackToHome = useCallback(() => {
        // Clear any error state before navigating
        dispatch(setGlobalError(''))
        // Always navigate to home, not back to the board that caused the error
        history.push('/')
    }, [history, dispatch])

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
