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

export function customHistory(): History {
    const history = createBrowserHistory({basename: Utils.getFrontendBaseURL()})

    if (Utils.isDesktop()) {
        if (windowAny.desktopAPI?.onBrowserHistoryPush) {
            windowAny.desktopAPI.onBrowserHistoryPush((pathName) => handleBrowserHistoryPush(pathName, history))
        } else {
            window.addEventListener('message', (event: MessageEvent) => handleBrowserHistoryMessage(event, history))
        }
    }

    // Patch push on the real history object rather than returning a spread
    // copy. history v4 keeps `location`/`action` current by reassigning them on
    // its own object, so a shallow copy would freeze `location` at init time and
    // cause <Router> to render a stale board (MM-68337).
    const originalPush = history.push.bind(history)
    history.push = (path: string, state?: unknown) => {
        if (Utils.isDesktop()) {
            doBrowserHistoryPush(`${boardsRouteBase}${path}`)
        } else {
            originalPush(path, state as Record<string, never>)
        }
    }

    return history
}

// Mattermost owns the browser URL and navigates through its own history
// instance, so the Boards history never hears about a core navigation. Adopt the
// current browser location before the first render so the router mounts the board
// the URL actually points to instead of the previously visited one (MM-68337).
export function syncHistoryWithBrowserLocation(history: History): void {
    // Same source as the history basename so the two can never drift.
    const base = Utils.getFrontendBaseURL()
    const prefix = base.startsWith('/') ? base : `/${base}`

    const {pathname, search, hash} = window.location
    if (pathname !== prefix && !pathname.startsWith(`${prefix}/`)) {
        return
    }

    const relativePath = pathname === prefix ? '/' : pathname.slice(prefix.length)
    const target = `${relativePath}${search}${hash}`

    const current = `${history.location.pathname}${history.location.search}${history.location.hash}`
    if (current !== target) {
        history.replace(target)
    }
}
