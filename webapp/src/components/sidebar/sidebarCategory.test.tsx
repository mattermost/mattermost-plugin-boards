// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {createMemoryHistory} from 'history'
import {Route, Router} from 'react-router-dom'

import {act, render, screen, waitFor} from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import {Provider as ReduxProvider} from 'react-redux'

import configureStore from 'redux-mock-store'

import {mocked} from 'jest-mock'

import {TestBlockFactory} from '../../test/testBlockFactory'

import {wrapIntl, wrapRBDNDDroppable} from '../../testUtils'

import mutator from '../../mutator'

import SidebarCategory from './sidebarCategory'

jest.mock('../../mutator')
const mockedMutator = mocked(mutator)

describe('components/sidebarCategory', () => {
    const board = TestBlockFactory.createBoard()
    board.id = 'board_id'

    const view = TestBlockFactory.createBoardView(board)
    view.fields.sortOptions = []
    const history = createMemoryHistory()

    const board1 = TestBlockFactory.createBoard()
    board1.id = 'board_1_id'

    const board2 = TestBlockFactory.createBoard()
    board2.id = 'board_2_id'

    const boards = [board1, board2]
    const categoryBoards1 = TestBlockFactory.createCategoryBoards()
    categoryBoards1.id = 'category_1_id'
    categoryBoards1.name = 'Category 1'
    categoryBoards1.boardMetadata = [{boardID: board1.id, hidden: false}, {boardID: board2.id, hidden: false}]

    const categoryBoards2 = TestBlockFactory.createCategoryBoards()
    categoryBoards2.id = 'category_2_id'
    categoryBoards2.name = 'Category 2'

    const categoryBoards3 = TestBlockFactory.createCategoryBoards()
    categoryBoards3.id = 'category_id_3'
    categoryBoards3.name = 'Category 3'

    const allCategoryBoards = [
        categoryBoards1,
        categoryBoards2,
        categoryBoards3,
    ]

    const state = {
        users: {
            me: {
                id: 'user_id_1',
                props: {},
            },
        },
        boards: {
            current: board.id,
            boards: {
                [board.id]: board,
            },
        },
        cards: {
            cards: {
                card_id_1: {title: 'Card'},
            },
            current: 'card_id_1',
        },
        views: {
            current: view.id,
            views: {
                [view.id]: view,
            },
        },
        teams: {
            current: {
                id: 'team-id',
            },
        },
    }

    test('sidebar call hideSidebar', () => {
        const mockStore = configureStore([])
        const store = mockStore(state)

        const component = wrapRBDNDDroppable(wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <SidebarCategory
                        hideSidebar={() => {}}
                        categoryBoards={categoryBoards1}
                        boards={boards}
                        allCategories={allCategoryBoards}
                        index={0}
                    />
                </Router>
            </ReduxProvider>,
        ))
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        // testing collapsed state of category
        const subItems = container.querySelectorAll('.category')
        expect(subItems).toBeDefined()
        userEvent.click(subItems[0] as Element)
        expect(container).toMatchSnapshot()
    })

    test('sidebar collapsed without active board', () => {
        const mockStore = configureStore([])
        const store = mockStore(state)

        const component = wrapRBDNDDroppable(wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <SidebarCategory
                        hideSidebar={() => {}}
                        categoryBoards={categoryBoards1}
                        boards={boards}
                        allCategories={allCategoryBoards}
                        index={0}
                    />
                </Router>
            </ReduxProvider>,
        ))
        const {container} = render(component)

        const subItems = container.querySelectorAll('.category-title')
        expect(subItems).toBeDefined()
        userEvent.click(subItems[0] as Element)
        expect(container).toMatchSnapshot()
    })

    test('sidebar collapsed with active board in it', () => {
        const mockStore = configureStore([])
        const store = mockStore(state)

        const component = wrapRBDNDDroppable(wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <SidebarCategory
                        hideSidebar={() => {}}
                        activeBoardID={board1.id}
                        categoryBoards={categoryBoards1}
                        boards={boards}
                        allCategories={allCategoryBoards}
                        index={0}
                    />
                </Router>
            </ReduxProvider>,
        ))
        const {container} = render(component)

        const subItems = container.querySelectorAll('.category-title')
        expect(subItems).toBeDefined()
        userEvent.click(subItems[0] as Element)
        expect(container).toMatchSnapshot()
    })

    test('sidebar template close self', () => {
        const mockStore = configureStore([])
        const store = mockStore(state)

        const mockTemplateClose = jest.fn()

        const component = wrapRBDNDDroppable(wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <SidebarCategory
                        activeBoardID={board1.id}
                        hideSidebar={() => {}}
                        categoryBoards={categoryBoards1}
                        boards={boards}
                        allCategories={allCategoryBoards}
                        index={0}
                        onBoardTemplateSelectorClose={mockTemplateClose}
                    />
                </Router>
            </ReduxProvider>,
        ))
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        // testing collapsed state of category
        const subItems = container.querySelectorAll('.subitem')
        expect(subItems).toBeDefined()
        userEvent.click(subItems[0] as Element)
        expect(mockTemplateClose).toBeCalled()
    })

    test('sidebar template close other', () => {
        const mockStore = configureStore([])
        const store = mockStore(state)

        const mockTemplateClose = jest.fn()

        const component = wrapRBDNDDroppable(wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <SidebarCategory
                        activeBoardID={board2.id}
                        hideSidebar={() => {}}
                        categoryBoards={categoryBoards1}
                        boards={boards}
                        allCategories={allCategoryBoards}
                        index={0}
                        onBoardTemplateSelectorClose={mockTemplateClose}
                    />
                </Router>
            </ReduxProvider>,
        ))
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        // testing collapsed state of category
        const subItems = container.querySelectorAll('.category-title')
        expect(subItems).toBeDefined()
        userEvent.click(subItems[0] as Element)
        expect(mockTemplateClose).not.toBeCalled()
    })

    describe('onDeleteBoard navigation', () => {
        // State that includes board memberships so the Delete option is visible
        const deleteNavState = {
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
                myConfig: {},
            },
            boards: {
                current: board1.id,
                boards: {
                    [board1.id]: board1,
                    [board2.id]: board2,
                },
                templates: {},
                myBoardMemberships: {
                    [board1.id]: {userId: 'user_id_1', schemeAdmin: true},
                    [board2.id]: {userId: 'user_id_1', schemeAdmin: true},
                },
            },
            cards: {
                cards: {},
                current: '',
            },
            views: {
                current: '',
                views: {},
            },
            teams: {
                current: {
                    id: 'team-id',
                },
            },
        }

        let portalEl: HTMLElement

        beforeEach(() => {
            jest.useFakeTimers()
            portalEl = document.createElement('div')
            portalEl.id = 'focalboard-root-portal'
            document.body.appendChild(portalEl)
        })

        afterEach(() => {
            jest.useRealTimers()
            document.body.removeChild(portalEl)
            mockedMutator.deleteBoard.mockReset()
        })

        const renderForDelete = (
            testHistory: ReturnType<typeof createMemoryHistory>,
            categoryBoarsdProps: typeof categoryBoards1,
            boardsList: typeof boards,
            allCats: typeof allCategoryBoards,
        ) => {
            const mockStore = configureStore([])
            const store = mockStore(deleteNavState)

            return render(
                wrapRBDNDDroppable(wrapIntl(
                    <ReduxProvider store={store}>
                        <Router history={testHistory}>
                            <Route path='/team/:teamId/:boardId'>
                                <SidebarCategory
                                    hideSidebar={jest.fn()}
                                    activeBoardID={board1.id}
                                    categoryBoards={categoryBoarsdProps}
                                    boards={boardsList}
                                    allCategories={allCats}
                                    index={0}
                                />
                            </Route>
                        </Router>
                    </ReduxProvider>,
                )),
            )
        }

        const triggerDeleteAndGetAfterRedo = async () => {
            // Open the first board item's options menu
            const menuWrappers = document.querySelectorAll('.SidebarBoardItem div.MenuWrapper')
            act(() => { userEvent.click(menuWrappers[0] as Element) })

            // Click "Delete board"
            const deleteOption = await screen.findByText('Delete board')
            act(() => { userEvent.click(deleteOption) })

            // Click "Delete" in the confirmation dialog
            const deleteButton = await screen.findByText('Delete')
            await act(async () => { userEvent.click(deleteButton) })

            await waitFor(() => expect(mockedMutator.deleteBoard).toBeCalledTimes(1))

            return mockedMutator.deleteBoard.mock.calls[0][2] as () => Promise<void>
        }

        test('navigates to adjacent board when multiple boards are in the category', async () => {
            const testHistory = createMemoryHistory()
            testHistory.push('/team/team-id/board_1_id')
            testHistory.push = jest.fn()

            // categoryBoards1 has board1 and board2 in boardMetadata
            renderForDelete(testHistory, categoryBoards1, [board1, board2], allCategoryBoards)

            const afterRedo = await triggerDeleteAndGetAfterRedo()

            await act(async () => {
                await afterRedo()
                jest.advanceTimersByTime(200)
            })

            // board1 is at index 0 of [board1, board2] → nextIndex = 1 → board2
            expect(testHistory.push).toHaveBeenCalledWith('/team/team-id/board_2_id')
        })

        test('falls back to a board in another category when no adjacent board exists', async () => {
            const testHistory = createMemoryHistory()
            testHistory.push('/team/team-id/board_1_id')
            testHistory.push = jest.fn()

            // Only board1 is in the current category
            const singleBoardCat = {...categoryBoards1, boardMetadata: [{boardID: board1.id, hidden: false}]}
            // Another category has board2
            const otherCatWithBoard2 = {...categoryBoards2, boardMetadata: [{boardID: board2.id, hidden: false}]}
            const allCatsForTest = [singleBoardCat, otherCatWithBoard2, categoryBoards3]

            renderForDelete(testHistory, singleBoardCat, [board1], allCatsForTest)

            const afterRedo = await triggerDeleteAndGetAfterRedo()

            await act(async () => {
                await afterRedo()
                jest.advanceTimersByTime(200)
            })

            expect(testHistory.push).toHaveBeenCalledWith('/team/team-id/board_2_id')
        })

        test('navigates to team page when no boards remain after deletion', async () => {
            const testHistory = createMemoryHistory()
            testHistory.push('/team/team-id/board_1_id')
            testHistory.push = jest.fn()

            // Only board1 in the only category
            const singleBoardCat = {...categoryBoards1, boardMetadata: [{boardID: board1.id, hidden: false}]}
            const allCatsForTest = [singleBoardCat]

            renderForDelete(testHistory, singleBoardCat, [board1], allCatsForTest)

            const afterRedo = await triggerDeleteAndGetAfterRedo()

            await act(async () => {
                await afterRedo()
                jest.advanceTimersByTime(200)
            })

            expect(testHistory.push).toHaveBeenCalledWith('/team/team-id')
        })
    })
})
