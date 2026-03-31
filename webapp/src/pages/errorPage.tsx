// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {useLocation} from 'react-router-dom'
import {FormattedMessage} from 'react-intl'

import ErrorIllustration from '../svg/error-illustration'

import Button from '../widgets/buttons/button'
import './errorPage.scss'

import {errorDefFromId, ErrorId} from '../errors'
import {Utils} from '../utils'
import {UserSettingKey} from '../userSettings'
import {Constants} from '../constants'

const ErrorPage = () => {
    const queryParams = new URLSearchParams(useLocation().search)
    const errid = queryParams.get('id')
    const errorDef = errorDefFromId(errid as ErrorId)

    const handleButtonClick = useCallback((path: string | ((params: URLSearchParams) => string), clearHistory: boolean) => {
        let url = '/'
        if (typeof path === 'function') {
            url = path(queryParams)
        } else if (path) {
            url = path as string
        }

        // Clear stored board/view IDs and set flag to skip all auto-redirects
        if (clearHistory) {
            localStorage.removeItem(UserSettingKey.LastBoardId)
            localStorage.removeItem(UserSettingKey.LastViewId)
            sessionStorage.setItem(Constants.sessionStorageSkipBoardRedirectKey, 'true')
        }

        const finalUrl = clearHistory ? Utils.getFrontendBaseURL(true) : url
        window.location.href = finalUrl
    }, [queryParams])

    const makeButton = ((path: string | ((params: URLSearchParams) => string), txt: string, fill: boolean, clearHistory: boolean) => {
        return (
            <Button
                filled={fill}
                size='large'
                onClick={async () => {
                    handleButtonClick(path, clearHistory)
                }}
            >
                {txt}
            </Button>
        )
    })

    if (!Utils.isFocalboardPlugin() && errid === ErrorId.NotLoggedIn) {
        handleButtonClick(errorDef.button1Redirect, errorDef.button1ClearHistory)
    }

    return (
        <div className='ErrorPage'>
            <div>
                <div className='title'>
                    <FormattedMessage
                        id='error.page.title'
                        defaultMessage={'Sorry, something went wrong'}
                    />
                </div>
                <div className='subtitle'>
                    {errorDef.title}
                </div>
                <ErrorIllustration/>
                <br/>
                {
                    (errorDef.button1Enabled ? makeButton(errorDef.button1Redirect, errorDef.button1Text, errorDef.button1Fill, errorDef.button1ClearHistory) : null)
                }
                {
                    (errorDef.button2Enabled ? makeButton(errorDef.button2Redirect, errorDef.button2Text, errorDef.button2Fill, errorDef.button2ClearHistory) : null)
                }
            </div>
        </div>
    )
}

export default React.memo(ErrorPage)
