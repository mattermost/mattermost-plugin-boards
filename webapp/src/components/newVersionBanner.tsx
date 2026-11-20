// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react'
import {FormattedMessage} from 'react-intl'

import wsClient from '../wsclient'
import {useLegacyMode} from '../hooks/useProductName'

import './newVersionBanner.scss'

const NewVersionBanner = () => {
    const [appVersionChanged, setAppVersionChanged] = useState(false)
    const legacyMode = useLegacyMode()
    useEffect(() => {
        wsClient.onAppVersionChangeHandler = setAppVersionChanged
    }, [])

    if (!appVersionChanged) {
        return null
    }

    const newVersionReload = (e: any) => {
        e.preventDefault()
        location.reload()
    }

    return (
        <div className='NewVersionBanner'>
            <a
                target='_blank'
                rel='noreferrer'
                onClick={newVersionReload}
            >
                {legacyMode ? (
                    <FormattedMessage
                        id='BoardPage.newVersion.legacy'
                        defaultMessage='A new version of Legacy Boards is available, click here to reload.'
                    />
                ) : (
                    <FormattedMessage
                        id='BoardPage.newVersion'
                        defaultMessage='A new version of Boards is available, click here to reload.'
                    />
                )}
            </a>
        </div>
    )
}

export default NewVersionBanner
