// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import '@testing-library/jest-dom'

import {MockStoreEnhanced} from 'redux-mock-store'

import {Provider as ReduxProvider} from 'react-redux'

import {render, screen, act, fireEvent} from '@testing-library/react'
import {mocked} from 'jest-mock'
import userEvent from '@testing-library/user-event'

import {createMemoryHistory, History} from 'history'

import {Route, Router} from 'react-router-dom'

import octoClient from '../../octoClient'
import {Board, BoardTypeOpen, BoardTypePrivate, createBoard} from '../../blocks/board'
import {Team} from '../../store/teams'
import {TestBlockFactory} from '../../test/testBlockFactory'

import {mockStateStore, wrapDNDIntl} from '../../testUtils'

import BoardSwitcherDialog from './boardSwitcherDialog'

jest.mock('../../octoClient')
const mockedOctoClient = mocked(octoClient)

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

describe('component/BoardSwitcherDialog', () => {
    const team1: Team = {
        id: 'team-id-1',
        title: 'Dunder Mifflin',
        signupToken: '',
        updateAt: 0,
        modifiedBy: 'michael-scott',
    }

    const team2: Team = {
        id: 'team-id-2',
        title: 'Michael Scott Paper Company',
        signupToken: '',
        updateAt: 0,
        modifiedBy: 'michael-scott',
    }

    const me = TestBlockFactory.createUser()

    // A board in a team the user belongs to (present in the store).
    const boardInKnownTeam: Board = {
        ...createBoard(),
        id: 'board-known',
        teamId: team1.id,
        title: 'Design Sprint',
        type: BoardTypeOpen,
    }

    // A board returned by search whose team is NOT in the store. This happens
    // because the search endpoint also returns boards the user can reach via
    // board or channel membership, even in teams they have left (MM-69061).
    const boardInUnknownTeam: Board = {
        ...createBoard(),
        id: 'board-unknown',
        teamId: 'team-id-not-in-store',
        title: 'Design Review Archive',
        type: BoardTypePrivate,
    }

    const untitledBoard: Board = {
        ...createBoard(),
        id: 'board-untitled',
        teamId: team1.id,
        title: '',
        type: BoardTypeOpen,
    }

    const allBoards = [boardInKnownTeam, boardInUnknownTeam, untitledBoard]

    const baseState = {
        users: {
            me,
        },
        teams: {
            allTeams: [team1, team2],
            current: team1,
            currentId: team1.id,
        },
    }

    let store: MockStoreEnhanced<unknown, unknown>
    let history: History

    // Mirror the server: AND a case-insensitive match per whitespace word.
    const filterBoards = (query: string) => {
        const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean)
        if (words.length === 0) {
            return []
        }
        return allBoards.filter((board) =>
            words.every((word) => board.title.toLowerCase().includes(word)),
        )
    }

    beforeEach(() => {
        jest.clearAllMocks()

        // Utils.uuid() (used as the React key for each result row) relies on
        // URL.createObjectURL, which jsdom does not implement. Return a unique
        // value per call so result rows get stable, distinct keys.
        let uuidCounter = 0
        window.URL.createObjectURL = jest.fn(() => `blob:focalboard-test-00000000-0000-0000-0000-${uuidCounter++}`)

        store = mockStateStore([], baseState)
        history = createMemoryHistory({initialEntries: ['/team/team-id-1/board-initial']})

        mockedOctoClient.searchAll.mockImplementation(async (query: string) => filterBoards(query))
    })

    const renderDialog = (onClose: () => void = jest.fn(), overrideStore?: MockStoreEnhanced<unknown, unknown>) => render(wrapDNDIntl(
        <Router history={history}>
            <Route path='/team/:teamId/:boardId?/:viewId?/:cardId?'>
                <ReduxProvider store={overrideStore || store}>
                    <BoardSwitcherDialog onClose={onClose}/>
                </ReduxProvider>
            </Route>
        </Router>,
    ))

    const typeQuery = async (query: string) => {
        await act(async () => {
            const input = screen.getByPlaceholderText('Search for boards')
            await userEvent.type(input, query)
            await wait(300)
        })
    }

    test('shows the intro screen before any query is typed', () => {
        renderDialog()
        expect(screen.getByPlaceholderText('Search for boards')).toBeInTheDocument()
        expect(screen.getByRole('heading', {name: 'Search for boards'})).toBeInTheDocument()
        expect(mockedOctoClient.searchAll).not.toHaveBeenCalled()
    })

    test('renders boards from teams the user is not a member of without errors', async () => {
        const errorSpy = jest.spyOn(console, 'error').mockImplementation(() => {})
        renderDialog()
        await typeQuery('des')

        expect(screen.getByText('Design Sprint')).toBeInTheDocument()
        expect(screen.getByText('Design Review Archive')).toBeInTheDocument()
        expect(errorSpy).not.toHaveBeenCalled()
        errorSpy.mockRestore()
    })

    test('labels a board with its team when the team is in the store', async () => {
        const {container} = renderDialog()
        await typeQuery('design sprint')

        expect(screen.getByText('Design Sprint')).toBeInTheDocument()
        expect(screen.getByText('Dunder Mifflin')).toBeInTheDocument()
        expect(container.querySelectorAll('.teamTitle')).toHaveLength(1)
    })

    test('omits the team label for a board whose team is not in the store', async () => {
        const {container} = renderDialog()
        await typeQuery('des')

        // Both boards render, but only the known-team board carries a label.
        expect(container.querySelectorAll('.resultTitle')).toHaveLength(2)
        expect(container.querySelectorAll('.teamTitle')).toHaveLength(1)
        expect(screen.queryByText('team-id-not-in-store')).not.toBeInTheDocument()
    })

    test('renders the correct icon for open and private boards', async () => {
        const {container} = renderDialog()
        await typeQuery('des')

        expect(container.querySelectorAll('.GlobeIcon')).toHaveLength(1)
        expect(container.querySelectorAll('.LockOutlineIcon')).toHaveLength(1)
    })

    test('falls back to the untitled label for a board without a title', async () => {
        mockedOctoClient.searchAll.mockResolvedValue([untitledBoard])
        renderDialog()
        await typeQuery('untitled')

        expect(screen.getByText('Untitled board')).toBeInTheDocument()
    })

    test('navigates to the board when a result is clicked', async () => {
        const onClose = jest.fn()
        renderDialog(onClose)
        await typeQuery('design review archive')

        await act(async () => {
            fireEvent.click(screen.getByText('Design Review Archive'))
        })

        expect(history.location.pathname).toBe('/team/team-id-not-in-store/board-unknown')
        expect(onClose).toHaveBeenCalled()
    })

    test('selects a board with arrow keys and ENTER, including boards from unknown teams', async () => {
        const onClose = jest.fn()
        renderDialog(onClose)
        await typeQuery('des')

        // Down twice selects the second result (the unknown-team board). Each
        // press is dispatched in its own act() so the keydown listener closure
        // re-renders with the updated selection between presses.
        await act(async () => {
            fireEvent.keyDown(document, {key: 'ArrowDown', keyCode: 40})
        })
        await act(async () => {
            fireEvent.keyDown(document, {key: 'ArrowDown', keyCode: 40})
        })
        await act(async () => {
            fireEvent.keyDown(document, {key: 'Enter', keyCode: 13})
        })

        expect(history.location.pathname).toBe('/team/team-id-not-in-store/board-unknown')
        expect(onClose).toHaveBeenCalled()
    })

    test('returns to the intro screen when the query is cleared', async () => {
        renderDialog()
        await typeQuery('des')
        expect(screen.getByText('Design Sprint')).toBeInTheDocument()

        await act(async () => {
            const input = screen.getByPlaceholderText('Search for boards')
            await userEvent.clear(input)
            await wait(300)
        })

        expect(screen.queryByText('Design Sprint')).not.toBeInTheDocument()
        expect(screen.getByRole('heading', {name: 'Search for boards'})).toBeInTheDocument()
        expect(mockedOctoClient.searchAll).not.toHaveBeenCalledWith('')
    })

    test('shows the no-results message when nothing matches', async () => {
        renderDialog()
        await typeQuery('zzznomatch')

        expect(screen.getByText('No results for "zzznomatch"')).toBeInTheDocument()
    })

    test('does not navigate when there is no logged-in user', async () => {
        const onClose = jest.fn()
        const noMeStore = mockStateStore([], {...baseState, users: {}})
        renderDialog(onClose, noMeStore)
        await typeQuery('design sprint')

        await act(async () => {
            fireEvent.click(screen.getByText('Design Sprint'))
        })

        expect(history.location.pathname).toBe('/team/team-id-1/board-initial')
        expect(onClose).not.toHaveBeenCalled()
    })
})
