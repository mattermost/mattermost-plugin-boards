// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {History} from 'history'

import {Utils} from './utils'
import {SuiteWindow} from './types/index'

import {boardsRouteBase, customHistory, doBrowserHistoryPush, handleBrowserHistoryMessage, handleBrowserHistoryPush} from './desktopHistory'

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
})
