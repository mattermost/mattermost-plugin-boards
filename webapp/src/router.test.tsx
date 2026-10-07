// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {render} from '@testing-library/react'
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
jest.mock('./pages/errorPage', () => ({__esModule: true, default: () => null}))
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
            value: {featureFlags: {}},
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
})
