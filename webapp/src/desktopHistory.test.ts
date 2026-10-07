// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {History} from 'history'

import {Utils} from './utils'
import {SuiteWindow} from './types/index'

import {boardsRouteBase, customHistory, doBrowserHistoryPush, handleBrowserHistoryMessage, handleBrowserHistoryPush, syncHistoryWithBrowserLocation} from './desktopHistory'

const windowAny = (window as SuiteWindow)

describe('desktopHistory', () => {
    const originalFrontendBaseURL = windowAny.frontendBaseURL

    beforeEach(() => {
        jest.restoreAllMocks()
        jest.spyOn(Utils, 'log').mockImplementation(() => {})
        delete windowAny.desktopAPI
        windowAny.frontendBaseURL = '/boards'
    })

    afterAll(() => {
        windowAny.frontendBaseURL = originalFrontendBaseURL
    })

    describe('handleBrowserHistoryPush', () => {
        const makeHistory = () => ({replace: jest.fn()} as unknown as History)

        test('ignores an empty path', () => {
            const history = makeHistory()
            handleBrowserHistoryPush('', history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('ignores a path that still carries the server subpath', () => {
            const history = makeHistory()
            handleBrowserHistoryPush('/company/boards/team/team-id', history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('strips the boards route base before navigating', () => {
            const history = makeHistory()
            handleBrowserHistoryPush(`${boardsRouteBase}/team/team-id`, history)
            expect(history.replace).toHaveBeenCalledWith('/team/team-id')
        })

        test('navigates to the root for the bare boards route', () => {
            const history = makeHistory()
            handleBrowserHistoryPush(boardsRouteBase, history)
            expect(history.replace).toHaveBeenCalledWith('/')
        })

        test('ignores a route that only shares the boards prefix', () => {
            const history = makeHistory()
            handleBrowserHistoryPush('/boards-legacy/team/team-id', history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('rejects path traversal that escapes the boards route', () => {
            const history = makeHistory()
            handleBrowserHistoryPush('/boards/../admin', history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('rejects percent-encoded path traversal', () => {
            const history = makeHistory()
            handleBrowserHistoryPush('/boards/%2e%2e/admin', history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('preserves the query string and hash', () => {
            const history = makeHistory()
            handleBrowserHistoryPush('/boards/team/team-id?view=1#card', history)
            expect(history.replace).toHaveBeenCalledWith('/team/team-id?view=1#card')
        })
    })

    describe('handleBrowserHistoryMessage', () => {
        const makeHistory = () => ({replace: jest.fn()} as unknown as History)
        const sameOrigin = window.location.origin

        test('forwards a valid same-origin boards path', () => {
            const history = makeHistory()
            const event = {origin: sameOrigin, data: {message: {pathName: '/boards/team/team-id'}}} as MessageEvent
            handleBrowserHistoryMessage(event, history)
            expect(history.replace).toHaveBeenCalledWith('/team/team-id')
        })

        test('ignores messages from a different origin', () => {
            const history = makeHistory()
            const event = {origin: 'https://evil.example', data: {message: {pathName: '/boards/team/team-id'}}} as MessageEvent
            handleBrowserHistoryMessage(event, history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('ignores a null payload without throwing', () => {
            const history = makeHistory()
            const event = {origin: sameOrigin, data: null} as MessageEvent
            expect(() => handleBrowserHistoryMessage(event, history)).not.toThrow()
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('ignores a non-string pathName', () => {
            const history = makeHistory()
            const event = {origin: sameOrigin, data: {message: {pathName: 42}}} as unknown as MessageEvent
            handleBrowserHistoryMessage(event, history)
            expect(history.replace).not.toHaveBeenCalled()
        })
    })

    describe('doBrowserHistoryPush', () => {
        test('uses the desktop API when available', () => {
            const sendBrowserHistoryPush = jest.fn()
            windowAny.desktopAPI = {sendBrowserHistoryPush}

            doBrowserHistoryPush('/boards/team/team-id')

            expect(sendBrowserHistoryPush).toHaveBeenCalledWith('/boards/team/team-id')
        })

        test('falls back to postMessage when the desktop API is missing', () => {
            const postMessage = jest.spyOn(window, 'postMessage').mockImplementation(() => {})

            doBrowserHistoryPush('/boards/team/team-id')

            expect(postMessage).toHaveBeenCalledWith(
                {type: 'browser-history-push', message: {path: '/boards/team/team-id'}},
                window.location.origin,
            )
        })
    })

    describe('customHistory push on desktop', () => {
        test('sends a subpath-relative path so the subpath is never leaked (MM-67542)', () => {
            jest.spyOn(Utils, 'isDesktop').mockReturnValue(true)
            const sendBrowserHistoryPush = jest.fn()
            windowAny.desktopAPI = {sendBrowserHistoryPush}

            // Simulate a subpath deployment: the frontend base URL includes the subpath.
            windowAny.frontendBaseURL = '/company/boards'

            const history = customHistory()
            history.push('/team/team-id')

            expect(sendBrowserHistoryPush).toHaveBeenCalledWith('/boards/team/team-id')
            expect(sendBrowserHistoryPush).not.toHaveBeenCalledWith('/company/boards/team/team-id')
        })

        test('does not notify the desktop app when not running in desktop', () => {
            jest.spyOn(Utils, 'isDesktop').mockReturnValue(false)
            const sendBrowserHistoryPush = jest.fn()
            windowAny.desktopAPI = {sendBrowserHistoryPush}

            const history = customHistory()
            history.push('/team/team-id')

            expect(sendBrowserHistoryPush).not.toHaveBeenCalled()
        })
    })

    describe('customHistory live location (MM-68337)', () => {
        beforeEach(() => {
            window.history.pushState({}, '', '/')
        })

        test('keeps location current after navigating, instead of freezing it at init', () => {
            jest.spyOn(Utils, 'isDesktop').mockReturnValue(false)
            window.history.pushState({}, '', '/boards/team/team-a/board-a')

            const history = customHistory()
            expect(history.location.pathname).toBe('/team/team-a/board-a')

            // A shallow-copied history would keep the stale location here.
            history.push('/team/team-a/board-a/view-2')
            expect(history.location.pathname).toBe('/team/team-a/board-a/view-2')
        })

        test('keeps location live on desktop via the browser-history round-trip', () => {
            jest.spyOn(Utils, 'isDesktop').mockReturnValue(true)
            let pushListener: ((pathName: string) => void) | undefined
            const sendBrowserHistoryPush = jest.fn()
            windowAny.desktopAPI = {
                sendBrowserHistoryPush,
                onBrowserHistoryPush: (listener: (pathName: string) => void) => {
                    pushListener = listener
                    return () => {}
                },
            }

            window.history.pushState({}, '', '/boards/team/team-a/board-a')
            const history = customHistory()
            expect(history.location.pathname).toBe('/team/team-a/board-a')

            // On desktop, push routes through the desktop app instead of mutating
            // the local history directly.
            history.push('/team/team-a/board-a/view-2')
            expect(sendBrowserHistoryPush).toHaveBeenCalledWith('/boards/team/team-a/board-a/view-2')
            expect(history.location.pathname).toBe('/team/team-a/board-a')

            // The desktop app echoes the navigation back, which must advance the
            // live location so the router can react to it.
            pushListener?.('/boards/team/team-a/board-a/view-2')
            expect(history.location.pathname).toBe('/team/team-a/board-a/view-2')
        })
    })

    describe('syncHistoryWithBrowserLocation', () => {
        const makeHistory = (pathname: string, search = '', hash = '') => ({
            location: {pathname, search, hash},
            replace: jest.fn(),
        } as unknown as History)

        beforeEach(() => {
            window.history.pushState({}, '', '/')
            windowAny.frontendBaseURL = '/boards'
        })

        test('adopts the current browser URL, stripping the basename', () => {
            window.history.pushState({}, '', '/boards/team/team-a/board-a')
            const history = makeHistory('/')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).toHaveBeenCalledWith('/team/team-a/board-a')
        })

        test('preserves the query string and hash', () => {
            window.history.pushState({}, '', '/boards/team/team-a/board-a?view=1#card')
            const history = makeHistory('/')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).toHaveBeenCalledWith('/team/team-a/board-a?view=1#card')
        })

        test('preserves an error-page query string so the error id survives (MM-69658)', () => {
            window.history.pushState({}, '', '/boards/error?id=not-logged-in&r=%2Fteam%2Ft1%2F')
            const history = makeHistory('/')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).toHaveBeenCalledWith('/error?id=not-logged-in&r=%2Fteam%2Ft1%2F')
        })

        test('maps the bare boards route to the root', () => {
            window.history.pushState({}, '', '/boards')
            const history = makeHistory('/team/old-team/old-board')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).toHaveBeenCalledWith('/')
        })

        test('does nothing when the browser is not on a boards route', () => {
            window.history.pushState({}, '', '/admin_console/user_management')
            const history = makeHistory('/team/team-a/board-a')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('ignores a route that only shares the boards prefix', () => {
            window.history.pushState({}, '', '/boards-legacy/team/team-a')
            const history = makeHistory('/')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('does not replace when the history is already in sync', () => {
            window.history.pushState({}, '', '/boards/team/team-a/board-a')
            const history = makeHistory('/team/team-a/board-a')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).not.toHaveBeenCalled()
        })

        test('strips a subpath basename on subpath deployments', () => {
            windowAny.frontendBaseURL = '/company/boards'
            window.history.pushState({}, '', '/company/boards/team/team-a/board-a')
            const history = makeHistory('/')
            syncHistoryWithBrowserLocation(history)
            expect(history.replace).toHaveBeenCalledWith('/team/team-a/board-a')
        })
    })
})
