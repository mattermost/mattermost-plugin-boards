// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {useHistory} from 'react-router-dom'
import {FormattedMessage, useIntl} from 'react-intl'

import ErrorIllustration from '../svg/error-illustration'

import Button from '../widgets/buttons/button'
import './accessDeniedPage.scss'

import {setGlobalError} from '../store/globalError'
import {useAppDispatch} from '../store/hooks'

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
                <div className='title'>
                    <FormattedMessage
                        id='accessDenied.page.title'
                        defaultMessage={'Access Denied'}
                    />
                </div>
                <div className='subtitle'>
                    <FormattedMessage
                        id='accessDenied.page.subtitle'
                        defaultMessage={'You don\'t have access to this board. Please contact the board owner if you believe this is an error.'}
                    />
                </div>
                <ErrorIllustration/>
                <br/>
                <Button
                    filled={true}
                    size='large'
                    onClick={handleBackToHome}
                >
                    {intl.formatMessage({id: 'accessDenied.back-to-home', defaultMessage: 'Back to Home'})}
                </Button>
            </div>
        </div>
    )
}

export default React.memo(AccessDeniedPage)
