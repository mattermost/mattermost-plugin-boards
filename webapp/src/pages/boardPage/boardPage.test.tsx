// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {createMemoryHistory} from 'history'
import {Route, Router} from 'react-router-dom'

import {render, waitFor} from '@testing-library/react'

import {Provider as ReduxProvider} from 'react-redux'

import configureStore from 'redux-mock-store'
import {thunk} from 'redux-thunk'

import {mocked} from 'jest-mock'

import '@testing-library/jest-dom'

import {wrapIntl} from '../../testUtils'

import octoClient from '../../octoClient'
import {UserSettings} from '../../userSettings'
import {ErrorId} from '../../errors'
import {BoardMember} from '../../blocks/board'

import BoardPage from './boardPage'

// Prevent real WebSocket connections
jest.mock('../../hooks/websockets', () => ({
    useWebsockets: jest.fn(),
}))

jest.mock('../../octoClient')
const mockedOctoClient = mocked(octoClient)

// Mock loadBoardData and initialLoad thunks so we control what dispatch returns
const mockLoadBoardDataFn = jest.fn()
const mockInitialLoadFn = jest.fn()
jest.mock('../../store/initialLoad', () => {
    const loadBoardData = (...args: any[]) => mockLoadBoardDataFn(...args)
    loadBoardData.rejected = {type: 'initialLoad/loadBoardData/rejected'}
    const initialLoad = (...args: any[]) => mockInitialLoadFn(...args)
    const initialReadOnlyLoad = jest.fn(() => async () => ({}))
    return {loadBoardData, initialLoad, initialReadOnlyLoad}
})

// Stub heavy sub-components that would need their own deps
jest.mock('./setWindowTitleAndIcon', () => ({__esModule: true, default: () => null}))
jest.mock('./teamToBoardAndViewRedirect', () => ({__esModule: true, default: () => null}))
jest.mock('./undoRedoHotKeys', () => ({__esModule: true, default: () => null}))
jest.mock('./backwardCompatibilityQueryParamsRedirect', () => ({__esModule: true, default: () => null}))
jest.mock('./websocketConnection', () => ({__esModule: true, default: () => null}))
jest.mock('../../components/workspace', () => ({__esModule: true, default: () => null}))
jest.mock('../../components/messages/versionMessage', () => ({__esModule: true, default: () => null}))

// Stub fetchBoardMembers so it doesn't fire real API calls
jest.mock('../../store/boards', () => {
    const actual = jest.requireActual('../../store/boards')
    const fetchBoardMembers = jest.fn(() => async () => ({payload: []}))
    return {...actual, fetchBoardMembers}
})

describe('pages/boardPage', () => {
    const baseState = {
        users: {
            me: {
                id: 'user_id_1',
                permissions: [],
                props: {},
            },
            myConfig: {},
        },
        boards: {
            current: '',
            boards: {},
            templates: {},
            myBoardMemberships: {},
        },
        views: {
            current: '',
            views: {},
        },
        teams: {
            current: {id: 'team-id'},
        },
        sidebar: {
            categoryAttributes: [],
            hiddenBoardIDs: [],
        },
        globalError: {value: ''},
    }

    beforeEach(() => {
        jest.clearAllMocks()

        // Default: initialLoad is a no-op thunk
        mockInitialLoadFn.mockReturnValue(async () => ({}))

        // Default: loadBoardData returns empty blocks (simulates a missing/deleted board)
        mockLoadBoardDataFn.mockReturnValue(async () => ({
            type: 'initialLoad/loadBoardData/fulfilled',
            payload: {blocks: []},
        }))

        // Default: joinBoard/octoClient stubs
        mockedOctoClient.joinBoard.mockResolvedValue(undefined)
        mockedOctoClient.unhideBoard.mockResolvedValue({} as Response)
    })

    const renderBoardPage = (
        history: ReturnType<typeof createMemoryHistory>,
        state: Record<string, any> = baseState,
    ) => {
        const mockStore = configureStore([thunk])
        const store = mockStore(state)

        const rendered = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <Router history={history}>
                        <Route path='/team/:teamId/:boardId'>
                            <BoardPage/>
                        </Route>
                    </Router>
                </ReduxProvider>,
            ),
        )
        return {store, ...rendered}
    }

    test('deleted board (404) navigates to the team page and does not attempt to join', async () => {
        const history = createMemoryHistory()
        history.push('/team/team-id/deleted-board-id')
        history.push = jest.fn()
        const setLastBoardID = jest.spyOn(UserSettings, 'setLastBoardID')

        // boardNotFound resolves true → the board was really deleted (server answered 404)
        mockedOctoClient.boardNotFound.mockResolvedValue(true)

        const {store} = renderBoardPage(history)

        await waitFor(() => {
            expect(history.push).toHaveBeenCalledWith('/team/team-id')
        })

        expect(mockedOctoClient.boardNotFound).toHaveBeenCalledWith('deleted-board-id')
        expect(mockedOctoClient.joinBoard).not.toHaveBeenCalled()
        expect(setLastBoardID).toHaveBeenCalledWith('team-id', null)

        // A deleted board must never surface the access-denied page.
        expect(store.getActions()).not.toContainEqual(
            {type: 'globalError/setGlobalError', payload: ErrorId.AccessDenied},
        )
    })

    test('private board the user cannot read (403) is not treated as deleted and surfaces access denied', async () => {
        const history = createMemoryHistory()
        history.push('/team/team-id/private-board-id')
        history.push = jest.fn()
        const setLastBoardID = jest.spyOn(UserSettings, 'setLastBoardID')
        const setLastViewId = jest.spyOn(UserSettings, 'setLastViewId')

        // boardNotFound resolves false → the board exists (server answered 403), so we must try to join it
        mockedOctoClient.boardNotFound.mockResolvedValue(false)
        const accessDenied = Object.assign(new Error('access-denied'), {status: 403})
        mockedOctoClient.joinBoard.mockRejectedValue(accessDenied)

        const {store} = renderBoardPage(history)

        await waitFor(() => {
            expect(mockedOctoClient.joinBoard).toHaveBeenCalledWith('private-board-id', false)
        })

        // The user is never bounced to the team page the way a deleted board would be...
        expect(history.push).not.toHaveBeenCalledWith('/team/team-id')

        // ...and a regular user is shown the access-denied page.
        await waitFor(() => {
            expect(store.getActions()).toEqual(
                expect.arrayContaining([
                    {type: 'globalError/setGlobalError', payload: ErrorId.AccessDenied},
                ]),
            )
        })

        // The stored last board/view are cleared so the home redirect can't bounce the user back in.
        expect(setLastBoardID).toHaveBeenCalledWith('team-id', null)
        expect(setLastViewId).toHaveBeenCalledWith('private-board-id', null)
    })

    test('admin opening an unreadable private board gets the join-private-board dialog, not access denied', async () => {
        const adminState = {
            ...baseState,
            users: {
                ...baseState.users,
                me: {...baseState.users.me, permissions: ['manage_team']},
            },
        }
        const history = createMemoryHistory()
        history.push('/team/team-id/private-board-id')

        mockedOctoClient.boardNotFound.mockResolvedValue(false)
        const accessDenied = Object.assign(new Error('access-denied'), {status: 403})
        mockedOctoClient.joinBoard.mockRejectedValue(accessDenied)

        const {store, findByText} = renderBoardPage(history, adminState)

        expect(await findByText('Join private board')).toBeInTheDocument()
        expect(store.getActions()).not.toContainEqual(
            {type: 'globalError/setGlobalError', payload: ErrorId.AccessDenied},
        )
    })

    test('access denied raised while loading board data shows the access-denied page for a regular user', async () => {
        // Some boards reject the blocks request outright; loadBoardData then rejects with AccessDenied.
        mockLoadBoardDataFn.mockReturnValue(async () => ({
            type: 'initialLoad/loadBoardData/rejected',
            error: {message: ErrorId.AccessDenied},
        }))
        const history = createMemoryHistory()
        history.push('/team/team-id/private-board-id')
        const setLastBoardID = jest.spyOn(UserSettings, 'setLastBoardID')

        const {store} = renderBoardPage(history)

        await waitFor(() => {
            expect(store.getActions()).toContainEqual(
                {type: 'globalError/setGlobalError', payload: ErrorId.AccessDenied},
            )
        })

        // The load path short-circuits before the deleted-board probe or a join attempt.
        expect(mockedOctoClient.boardNotFound).not.toHaveBeenCalled()
        expect(mockedOctoClient.joinBoard).not.toHaveBeenCalled()
        expect(setLastBoardID).toHaveBeenCalledWith('team-id', null)
    })

    test('access denied raised while loading board data shows the join dialog for an admin', async () => {
        mockLoadBoardDataFn.mockReturnValue(async () => ({
            type: 'initialLoad/loadBoardData/rejected',
            error: {message: ErrorId.AccessDenied},
        }))
        const adminState = {
            ...baseState,
            users: {
                ...baseState.users,
                me: {...baseState.users.me, permissions: ['manage_system']},
            },
        }
        const history = createMemoryHistory()
        history.push('/team/team-id/private-board-id')

        const {store, findByText} = renderBoardPage(history, adminState)

        expect(await findByText('Join private board')).toBeInTheDocument()
        expect(store.getActions()).not.toContainEqual(
            {type: 'globalError/setGlobalError', payload: ErrorId.AccessDenied},
        )
    })

    test('a board that loads normally records the last board and never probes or joins', async () => {
        mockLoadBoardDataFn.mockReturnValue(async () => ({
            type: 'initialLoad/loadBoardData/fulfilled',
            payload: {blocks: [{id: 'block-1'}]},
        }))
        const history = createMemoryHistory()
        history.push('/team/team-id/readable-board-id')
        const setLastBoardID = jest.spyOn(UserSettings, 'setLastBoardID')

        renderBoardPage(history)

        await waitFor(() => {
            expect(setLastBoardID).toHaveBeenCalledWith('team-id', 'readable-board-id')
        })

        expect(mockedOctoClient.boardNotFound).not.toHaveBeenCalled()
        expect(mockedOctoClient.joinBoard).not.toHaveBeenCalled()
    })

    test('joining an existing board the user may access loads its data and records the last board', async () => {
        mockedOctoClient.boardNotFound.mockResolvedValue(false)
        mockedOctoClient.joinBoard.mockResolvedValue({boardId: 'joinable-board-id'} as BoardMember)
        // After joining, loadBoardData is dispatched again and returns real blocks.
        mockLoadBoardDataFn.
            mockReturnValueOnce(async () => ({
                type: 'initialLoad/loadBoardData/fulfilled',
                payload: {blocks: []},
            })).
            mockReturnValue(async () => ({
                type: 'initialLoad/loadBoardData/fulfilled',
                payload: {blocks: [{id: 'block-1'}]},
            }))
        const history = createMemoryHistory()
        history.push('/team/team-id/joinable-board-id')
        history.push = jest.fn()
        const setLastBoardID = jest.spyOn(UserSettings, 'setLastBoardID')

        renderBoardPage(history)

        await waitFor(() => {
            expect(mockedOctoClient.joinBoard).toHaveBeenCalledWith('joinable-board-id', false)
        })

        await waitFor(() => {
            expect(setLastBoardID).toHaveBeenCalledWith('team-id', 'joinable-board-id')
        })
        expect(history.push).not.toHaveBeenCalledWith('/team/team-id')
    })

    test('a join that silently fails for a regular user shows the board-not-found error, not access denied', async () => {
        mockedOctoClient.boardNotFound.mockResolvedValue(false)
        // joinBoard resolves undefined (non-throwing failure) rather than rejecting with 403.
        mockedOctoClient.joinBoard.mockResolvedValue(undefined)
        const history = createMemoryHistory()
        history.push('/team/team-id/private-board-id')

        const {store} = renderBoardPage(history)

        await waitFor(() => {
            expect(store.getActions()).toContainEqual(
                {type: 'globalError/setGlobalError', payload: ErrorId.BoardNotFound},
            )
        })
        expect(store.getActions()).not.toContainEqual(
            {type: 'globalError/setGlobalError', payload: ErrorId.AccessDenied},
        )
    })
})
