// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {createMemoryHistory} from 'history'
import {Route, Router} from 'react-router-dom'

import {render, waitFor} from '@testing-library/react'

import {Provider as ReduxProvider} from 'react-redux'

import configureStore from 'redux-mock-store'
import thunk from 'redux-thunk'

import {mocked} from 'jest-mock'

import {wrapIntl} from '../../testUtils'

import octoClient from '../../octoClient'

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
        mockedOctoClient.joinBoard = jest.fn().mockResolvedValue(null)
        mockedOctoClient.unhideBoard = jest.fn().mockResolvedValue(undefined)
    })

    const renderBoardPage = (history: ReturnType<typeof createMemoryHistory>) => {
        const mockStore = configureStore([thunk])
        const store = mockStore(baseState)

        return render(
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
    }

    test('navigates to team page when loadBoardData returns empty blocks and board does not exist', async () => {
        const history = createMemoryHistory()
        history.push('/team/team-id/deleted-board-id')
        history.push = jest.fn()

        // getBoard returns undefined/null → board was deleted
        mockedOctoClient.getBoard = jest.fn().mockResolvedValue(undefined)

        renderBoardPage(history)

        await waitFor(() => {
            expect(history.push).toHaveBeenCalledWith('/team/team-id')
        })

        expect(mockedOctoClient.joinBoard).not.toHaveBeenCalled()
    })

    test('calls joinBoard when loadBoardData returns empty blocks and board still exists', async () => {
        const history = createMemoryHistory()
        history.push('/team/team-id/private-board-id')
        history.push = jest.fn()

        // getBoard returns a board object → board exists (private, not yet joined)
        mockedOctoClient.getBoard = jest.fn().mockResolvedValue({id: 'private-board-id', title: 'Private Board'})

        renderBoardPage(history)

        await waitFor(() => {
            expect(mockedOctoClient.joinBoard).toHaveBeenCalledWith('private-board-id', false)
        })

        expect(history.push).not.toHaveBeenCalledWith('/team/team-id')
    })
})
