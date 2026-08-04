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
    // Only navigate for the boards root or a path under it, so a route like
    // `/boards-legacy/...` is not mistaken for a boards path.
    if (!pathName || (pathName !== boardsRouteBase && !pathName.startsWith(`${boardsRouteBase}/`))) {
        return
    }

    // Resolve dot segments so traversal like `/boards/../admin` can't escape the
    // boards route, then re-check the boundary against the canonical path.
    const {pathname, search, hash} = new URL(pathName, window.location.origin)
    if (pathname !== boardsRouteBase && !pathname.startsWith(`${boardsRouteBase}/`)) {
        return
    }

    const relativePath = pathname === boardsRouteBase ? '/' : pathname.slice(boardsRouteBase.length)
    Utils.log(`Navigating Boards to ${pathName}`)
    history.replace(`${relativePath}${search}${hash}`)
}

export const handleBrowserHistoryMessage = (event: MessageEvent, history: History): void => {
    if (event.origin !== windowAny.location.origin) {
        return
    }

    const pathName = event.data?.message?.pathName
    if (typeof pathName === 'string') {
        handleBrowserHistoryPush(pathName, history)
    }
}

export function customHistory() {
    const history = createBrowserHistory({basename: Utils.getFrontendBaseURL()})

    if (Utils.isDesktop()) {
        if (windowAny.desktopAPI?.onBrowserHistoryPush) {
            windowAny.desktopAPI.onBrowserHistoryPush((pathName) => handleBrowserHistoryPush(pathName, history))
        } else {
            window.addEventListener('message', (event: MessageEvent) => handleBrowserHistoryMessage(event, history))
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
