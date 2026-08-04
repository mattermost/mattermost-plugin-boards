// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createBrowserHistory, History} from 'history'

import {Utils} from './utils'
import {SuiteWindow} from './types/index'

const windowAny = (window as SuiteWindow)

// Boards is always mounted at this in-app route. Paths exchanged with the
// Desktop App must be relative to the server subpath (the history basename),
// exactly like the core web app. Prefixing the full frontendBaseURL here would
// leak the subpath and force the Desktop App to strip it (see MM-67542).
export const boardsRouteBase = '/boards'

export const doBrowserHistoryPush = (path: string): void => {
    if (windowAny.desktopAPI?.sendBrowserHistoryPush) {
        windowAny.desktopAPI.sendBrowserHistoryPush(path)
    } else {
        window.postMessage(
            {
                type: 'browser-history-push',
                message: {path},
            },
            window.location.origin,
        )
    }
}

export const handleBrowserHistoryPush = (pathName: string, history: History): void => {
    if (!pathName || !pathName.startsWith(boardsRouteBase)) {
        return
    }

    Utils.log(`Navigating Boards to ${pathName}`)
    history.replace(pathName.replace(boardsRouteBase, ''))
}

export function customHistory() {
    const history = createBrowserHistory({basename: Utils.getFrontendBaseURL()})

    if (Utils.isDesktop()) {
        if (windowAny.desktopAPI?.onBrowserHistoryPush) {
            windowAny.desktopAPI.onBrowserHistoryPush((pathName) => handleBrowserHistoryPush(pathName, history))
        } else {
            window.addEventListener('message', (event: MessageEvent) => {
                if (event.origin !== windowAny.location.origin) {
                    return
                }

                handleBrowserHistoryPush(event.data.message?.pathName, history)
            })
        }
    }

    return {
        ...history,
        push: (path: string, state?: unknown) => {
            if (Utils.isDesktop()) {
                doBrowserHistoryPush(`${boardsRouteBase}${path}`)
            } else {
                history.push(path, state as Record<string, never>)
            }
        },
    }
}
