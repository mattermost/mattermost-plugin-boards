// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {render, screen} from '@testing-library/react'
import {Provider as ReduxProvider} from 'react-redux'
import configureStore from 'redux-mock-store'
import {thunk} from 'redux-thunk'

import {wrapIntl} from './testUtils'
import {customHistory} from './desktopHistory'
import {SuiteWindow} from './types/index'

import FocalboardRouter from './router'

// BoardPage is replaced with a recorder that reports every board the router
// actually mounts. Mounting the wrong board for even a single frame is recorded,
// so the test fails on a stale render and not merely on the final destination.
const mockRenderedBoards: Array<{teamId?: string, boardId?: string, viewId?: string}> = []
jest.mock('./pages/boardPage/boardPage', () => {
    // eslint-disable-next-line @typescript-eslint/no-var-requires, global-require
    const {useParams} = require('react-router-dom')
    const BoardPageRecorder = () => {
        const params = useParams() as {teamId?: string, boardId?: string, viewId?: string}
        mockRenderedBoards.push({teamId: params.teamId, boardId: params.boardId, viewId: params.viewId})
        return null
    }
    return {__esModule: true, default: BoardPageRecorder}
})

jest.mock('./pages/welcome/welcomePage', () => ({__esModule: true, default: () => null}))
jest.mock('./pages/accessDeniedPage', () => ({__esModule: true, default: () => null}))

const windowAny = (window as SuiteWindow)

describe('router', () => {
    const state = {
        users: {
            me: {id: 'user-1', props: {}, is_guest: false},
            loggedIn: true,
            myConfig: {},
        },
        clientConfig: {
            // disableTour keeps FBRoute from redirecting to /welcome when plugin
            // mode is on (needed by the MM-69658 error-page assertions).
            value: {featureFlags: {disableTour: 'true'}},
        },
        globalError: {value: ''},
    }

    const renderRouter = (history: ReturnType<typeof customHistory>) => {
        const store = configureStore([thunk])(state)
        return render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FocalboardRouter history={history}/>
                </ReduxProvider>,
            ),
        )
    }

    beforeEach(() => {
        mockRenderedBoards.length = 0
        window.history.pushState({}, '', '/')
        windowAny.frontendBaseURL = '/boards'

        // isFocalboardPlugin gates the welcome-page redirect; keep it off so the
        // router renders the board route directly.
        windowAny.isFocalboardPlugin = false
    })

    test('mounts the board the browser URL points to, never the stale last-visited board (MM-68337)', () => {
        // The Boards history instance is created while the browser is on a board
        // in another team (the previously visited board).
        window.history.pushState({}, '', '/boards/team/team-b/board-b/view-b')
        const history = customHistory()

        // Mattermost then navigates its own history to the clicked share link
        // without notifying the Boards history instance — this is the real flow
        // that leaves the Boards history stale.
        window.history.pushState({}, '', '/boards/team/team-a/board-a/view-a')

        renderRouter(history)

        const stale = mockRenderedBoards.find((board) => board.boardId === 'board-b')
        expect(stale).toBeUndefined()

        expect(mockRenderedBoards.length).toBeGreaterThan(0)
        expect(mockRenderedBoards[mockRenderedBoards.length - 1]).toMatchObject({
            teamId: 'team-a',
            boardId: 'board-a',
            viewId: 'view-a',
        })
    })

    test('opens the linked shared board, not the stale one, for a cross-team share link (MM-68337)', () => {
        // Share links are the reported trigger and match the readonly /shared/ route.
        window.history.pushState({}, '', '/boards/team/team-b/board-b/view-b')
        const history = customHistory()

        window.history.pushState({}, '', '/boards/team/team-a/shared/board-a/view-a')

        renderRouter(history)

        const stale = mockRenderedBoards.find((board) => board.boardId === 'board-b')
        expect(stale).toBeUndefined()

        expect(mockRenderedBoards.length).toBeGreaterThan(0)
        expect(mockRenderedBoards[mockRenderedBoards.length - 1]).toMatchObject({
            teamId: 'team-a',
            boardId: 'board-a',
        })
    })

    test('renders the board already in the URL when history is in sync', () => {
        window.history.pushState({}, '', '/boards/team/team-a/board-a/view-a')
        const history = customHistory()

        renderRouter(history)

        expect(mockRenderedBoards.find((board) => board.boardId !== 'board-a')).toBeUndefined()
        expect(mockRenderedBoards[mockRenderedBoards.length - 1]).toMatchObject({
            teamId: 'team-a',
            boardId: 'board-a',
        })
    })

    describe('FocalboardRouter initial location sync (MM-69658)', () => {
        beforeEach(() => {
            // ErrorPage only renders its message (instead of auto-redirecting)
            // in plugin mode.
            windowAny.isFocalboardPlugin = true
        })

        it('preserves the error id query string so a specific error is shown', () => {
            window.history.pushState({}, '', '/boards/team/t1/b1')
            const history = customHistory()
            window.history.pushState({}, '', '/boards/error?id=not-logged-in&r=%2Fteam%2Ft1%2F')

            renderRouter(history)

            expect(history.location.pathname).toBe('/error')
            expect(history.location.search).toContain('id=not-logged-in')
            expect(history.location.search).toContain('r=')

            screen.getByText(/session may have expired/)
            screen.getByRole('button', {name: 'Log in'})
            expect(screen.queryByText('An error occurred.')).toBeNull()
        })

        it('preserves the query string and hash together on the initial sync', () => {
            window.history.pushState({}, '', '/boards/team/t1/b1')
            const history = customHistory()
            window.history.pushState({}, '', '/boards/team/t1/b1?foo=bar#section')

            renderRouter(history)

            expect(history.location.pathname).toBe('/team/t1/b1')
            expect(history.location.search).toBe('?foo=bar')
            expect(history.location.hash).toBe('#section')
        })

        it('leaves a plain path without query or hash untouched', () => {
            window.history.pushState({}, '', '/boards/team/t1/b1')
            const history = customHistory()

            renderRouter(history)

            expect(history.location.pathname).toBe('/team/t1/b1')
            expect(history.location.search).toBe('')
            expect(history.location.hash).toBe('')
        })
    })
})
