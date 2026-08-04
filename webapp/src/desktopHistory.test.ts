// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {History} from 'history'

import {Utils} from './utils'
import {SuiteWindow} from './types/index'

import {boardsRouteBase, customHistory, doBrowserHistoryPush, handleBrowserHistoryPush} from './desktopHistory'

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
