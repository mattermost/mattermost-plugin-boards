// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {createMemoryHistory} from 'history'
import {Route, Router} from 'react-router-dom'
import {act, render} from '@testing-library/react'
import {Provider as ReduxProvider} from 'react-redux'
import configureStore from 'redux-mock-store'

import {TestBlockFactory} from '../../test/testBlockFactory'
import {UserSettings} from '../../userSettings'

import TeamToBoardAndViewRedirect from './teamToBoardAndViewRedirect'

describe('pages/boardPage/teamToBoardAndViewRedirect', () => {
    const board = TestBlockFactory.createBoard()
    board.id = 'board-1'
    const firstView = TestBlockFactory.createBoardView(board)
    firstView.id = 'view-1'
    firstView.title = 'A'
    const lastView = TestBlockFactory.createBoardView(board)
    lastView.id = 'view-2'
    lastView.title = 'B'

    const state = {
        teams: {current: {id: 'team-id'}},
        boards: {current: board.id, boards: {[board.id]: board}, loadingBoard: false},
        views: {current: '', views: {[firstView.id]: firstView, [lastView.id]: lastView}},
        sidebar: {categoryAttributes: [], hiddenBoardIDs: []},
    }

    const renderAt = (path: string, storeState: Record<string, unknown> | (() => Record<string, unknown>) = state) => {
        const history = createMemoryHistory({initialEntries: [path]})
        const store = configureStore([])(storeState)
        render(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Route path='/team/:teamId/:boardId?/:viewId?/:cardId?'>
                        <TeamToBoardAndViewRedirect/>
                    </Route>
                </Router>
            </ReduxProvider>,
        )
        return {history, store}
    }

    beforeEach(() => {
        localStorage.clear()
    })

    test('replaces a deleted view in the URL with the last viewed one', () => {
        UserSettings.setLastViewId(board.id, lastView.id)

        const {history, store} = renderAt(`/team/team-id/${board.id}/deleted-view?q=1`)

        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/${lastView.id}`)
        expect(history.location.search).toBe('?q=1')
        expect(store.getActions()).toContainEqual({type: 'views/setCurrent', payload: lastView.id})
    })

    test('uses the last viewed view from before BoardPage stored the deleted view from the URL', () => {
        UserSettings.setLastViewId(board.id, lastView.id)
        let loadingBoard = true
        const {history, store} = renderAt(
            `/team/team-id/${board.id}/deleted-view`,
            () => ({...state, boards: {...state.boards, loadingBoard}}),
        )

        UserSettings.setLastViewId(board.id, 'deleted-view')
        act(() => {
            loadingBoard = false
            store.dispatch({type: 'boards/loaded'})
        })

        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/${lastView.id}`)
        expect(UserSettings.lastViewId[board.id]).toBe(lastView.id)
    })

    test('opens the view selected after the board opened when navigating to the board without a view', () => {
        const {history} = renderAt(`/team/team-id/${board.id}/${firstView.id}`)

        act(() => {
            UserSettings.setLastViewId(board.id, lastView.id)
            history.push(`/team/team-id/${board.id}/${lastView.id}`)
        })
        act(() => {
            history.push(`/team/team-id/${board.id}`)
        })

        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/${lastView.id}`)
    })

    test('falls back to the first view when the stored last view no longer exists', () => {
        UserSettings.setLastViewId(board.id, 'deleted-view')

        const {history} = renderAt(`/team/team-id/${board.id}`)

        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/${firstView.id}`)
        expect(UserSettings.lastViewId[board.id]).toBe(firstView.id)
    })

    test('keeps a valid view and does not bounce off a view created after the board opened', () => {
        const {history} = renderAt(`/team/team-id/${board.id}/${lastView.id}`)
        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/${lastView.id}`)

        act(() => {
            history.push(`/team/team-id/${board.id}/new-view`)
        })

        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/new-view`)
    })

    test('waits for the board to finish loading before checking the view', () => {
        const {history} = renderAt(
            `/team/team-id/${board.id}/deleted-view`,
            {...state, boards: {...state.boards, loadingBoard: true}},
        )

        expect(history.location.pathname).toBe(`/team/team-id/${board.id}/deleted-view`)
    })
})
