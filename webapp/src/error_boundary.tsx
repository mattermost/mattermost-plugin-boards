// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import {Utils} from './utils'
import {Constants} from './constants'

type State = {
    hasError: boolean
}

type Props = {
    children: React.ReactNode
}

export default class ErrorBoundary extends React.Component<Props, State> {
    state = {hasError: false}
    msg = 'Redirecting to error page...'

    handleError = (): void => {
        if (window.location.pathname.endsWith('/error')) {
            return
        }

        // Loop detection: prevent infinite error redirects
        const now = Date.now()
        const lastRedirectTime = parseInt(sessionStorage.getItem(Constants.sessionStorageErrorRedirectTimeKey) || '0', 10)
        const redirectCount = parseInt(sessionStorage.getItem(Constants.sessionStorageErrorRedirectCountKey) || '0', 10)

        if (now - lastRedirectTime > 10000) {
            sessionStorage.setItem(Constants.sessionStorageErrorRedirectCountKey, '1')
            sessionStorage.setItem(Constants.sessionStorageErrorRedirectTimeKey, now.toString())
        } else {
            const newCount = redirectCount + 1
            sessionStorage.setItem(Constants.sessionStorageErrorRedirectCountKey, newCount.toString())
            sessionStorage.setItem(Constants.sessionStorageErrorRedirectTimeKey, now.toString())

            // If redirected more than 3 times in 10 seconds, go to boards root
            if (newCount > 3) {
                sessionStorage.removeItem(Constants.sessionStorageErrorRedirectCountKey)
                sessionStorage.removeItem(Constants.sessionStorageErrorRedirectTimeKey)
                window.location.replace(Utils.getFrontendBaseURL(true))
                return
            }
        }

        const url = Utils.getFrontendBaseURL(true) + '/error?id=unknown'
        window.location.replace(url)
    }

    static getDerivedStateFromError(/*error: Error*/): State {
        return {hasError: true}
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
        Utils.logError(error + ': ' + errorInfo)
    }

    shouldComponentUpdate(): boolean {
        return true
    }

    render(): React.ReactNode {
        if (this.state.hasError) {
            this.handleError()
            return <span>{this.msg}</span>
        }
        return this.props.children
    }
}

