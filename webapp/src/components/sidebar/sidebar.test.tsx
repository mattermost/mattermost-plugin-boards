// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import configureStore from 'redux-mock-store'

import {createMemoryHistory} from 'history'
import {Provider as ReduxProvider} from 'react-redux'
import {Router} from 'react-router-dom'

import {act, render, waitFor} from '@testing-library/react'

import {thunk} from 'redux-thunk'

import {mocked} from 'jest-mock'

import userEvent from '@testing-library/user-event'

import {mockMatchMedia, wrapIntl} from '../../testUtils'

import {TestBlockFactory} from '../../test/testBlockFactory'
import octoClient from '../../../../webapp/src/octoClient'
import {Constants} from '../../constants'

import Sidebar from './sidebar'

Object.defineProperty(Constants, 'versionString', {value: '1.0.0'})

jest.mock('../../../../webapp/src/octoClient')
const mockedOctoClient = mocked(octoClient)

beforeAll(() => {
    mockMatchMedia({matches: true})
})

describe('components/sidebarSidebar', () => {
    beforeEach(() => {
        jest.clearAllMocks()
    })

    const mockStore = configureStore([thunk])

    const board = TestBlockFactory.createBoard()
    board.id = 'board1'
    board.teamId = 'team-id'

    const categoryAttribute1 = TestBlockFactory.createCategoryBoards()
    categoryAttribute1.id = 'category1'
    categoryAttribute1.name = 'Category 1'
    categoryAttribute1.teamID = 'team-id'
    categoryAttribute1.boardMetadata = [{boardID: board.id, hidden: false}]

    const defaultCategory = TestBlockFactory.createCategoryBoards()
    defaultCategory.id = 'default_category'
    defaultCategory.name = 'Boards'
    defaultCategory.teamID = 'team-id'
    defaultCategory.boardMetadata = []

    test('sidebar hidden', () => {
        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board.id,
                boards: {
                    [board.id]: board,
                },
                myBoardMemberships: {
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
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    categoryAttribute1,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        const hideSidebar = container.querySelector('button > .HideSidebarIcon')
        expect(hideSidebar).toBeDefined()

        userEvent.click(hideSidebar as Element)
        expect(container).toMatchSnapshot()

        const showSidebar = container.querySelector('button > .ShowSidebarIcon')
        expect(showSidebar).toBeDefined()
    })

    test('sidebar expect hidden', () => {
        const customGlobal = global as any

        customGlobal.innerWidth = 500

        const localCategoryAttribute = TestBlockFactory.createCategoryBoards()
        localCategoryAttribute.id = 'category1'
        localCategoryAttribute.name = 'Category 1'
        categoryAttribute1.boardMetadata = [{boardID: board.id, hidden: false}]

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board.id,
                boards: {
                    [board.id]: board,
                },
                myBoardMemberships: {
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
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    categoryAttribute1,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        const hideSidebar = container.querySelector('button > .HideSidebarIcon')
        expect(hideSidebar).toBeNull()

        const showSidebar = container.querySelector('button > .ShowSidebarIcon')
        expect(showSidebar).toBeDefined()

        customGlobal.innerWidth = 1024
    })

    test('dont show hidden boards', () => {
        const localCategoryAttribute = TestBlockFactory.createCategoryBoards()
        localCategoryAttribute.id = 'category1'
        localCategoryAttribute.name = 'Category 1'
        localCategoryAttribute.boardMetadata = [{boardID: board.id, hidden: true}]

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board.id,
                boards: {
                    [board.id]: board,
                },
                myBoardMemberships: {
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
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                },
                myConfig: {
                    hiddenBoardIDs: {value: {
                        [board.id]: true,
                    }},
                },
            },
            sidebar: {
                categoryAttributes: [
                    localCategoryAttribute,
                ],
                hiddenBoardIDs: [board.id],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        const {container, getAllByText} = render(component)
        expect(container).toMatchSnapshot()

        const sidebarBoards = container.getElementsByClassName('SidebarBoardItem')

        // The only board in redux store is hidden, so there should
        // be no boards visible in sidebar
        expect(sidebarBoards.length).toBe(0)

        const noBoardsText = getAllByText('No boards inside')
        expect(noBoardsText.length).toBe(1)
    })

    test('some categories hidden', () => {
        const collapsedCategory = TestBlockFactory.createCategoryBoards()
        collapsedCategory.id = 'categoryCollapsed'
        collapsedCategory.name = 'Category 2'
        collapsedCategory.collapsed = true
        collapsedCategory.boardMetadata = []

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board.id,
                boards: {
                    [board.id]: board,
                },
                myBoardMemberships: {
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
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    categoryAttribute1,
                    collapsedCategory,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        const sidebarCollapsedCategory = container.querySelectorAll('.octo-sidebar-item.category.collapsed')
        expect(sidebarCollapsedCategory.length).toBe(1)
    })

    test('should assign default category if current board doesnt have a category', async () => {
        const board2 = TestBlockFactory.createBoard()
        board2.id = 'board2'
        board2.teamId = 'team-id'

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board2.id,
                boards: {
                    [board2.id]: board2,
                },
                myBoardMemberships: {
                    [board2.id]: board2,
                },
            },
            cards: {
                cards: {
                    card_id_1: {title: 'Card'},
                },
                current: 'card_id_1',
            },
            views: {
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    categoryAttribute1,
                    defaultCategory,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        mockedOctoClient.moveBoardToCategory.mockResolvedValueOnce({} as Response)

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        await waitFor(() => 
            expect(mockedOctoClient.moveBoardToCategory).toHaveBeenCalledWith('team-id', 'board2', 'default_category', '')
        )
        expect(mockedOctoClient.moveBoardToCategory).toHaveBeenCalledTimes(1)
    })

    test('shouldnt do any category assignment is board is in a category', () => {
        const board2 = TestBlockFactory.createBoard()
        board2.id = 'board2'
        board2.teamId = 'team-id'

        const categoryAttribute2 = TestBlockFactory.createCategoryBoards()
        categoryAttribute2.id = 'category2'
        categoryAttribute2.name = 'Category 2'
        categoryAttribute2.teamID = 'team-id'
        categoryAttribute2.boardMetadata = [{boardID: board2.id, hidden: false}]

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board2.id,
                boards: {
                    [board2.id]: board2,
                },
                myBoardMemberships: {
                    [board2.id]: board2,
                },
            },
            cards: {
                cards: {
                    card_id_1: {title: 'Card'},
                },
                current: 'card_id_1',
            },
            views: {
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    categoryAttribute1,
                    categoryAttribute2,
                    defaultCategory,
                ],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()

        expect(mockedOctoClient.moveBoardToCategory).toHaveBeenCalledTimes(0)
    })

    test('shouldnt assign default category when sidebar categories belong to a different team', async () => {
        // Simulates the window during a team switch where an in-flight fetch for
        // the previous team leaves its categories in the store while the new team
        // is already current. The board belongs to the current team but is not in
        // the stale categories; without the team guard it would be relocated to
        // the previous team's default "Boards" category (cross-team move).
        const board2 = TestBlockFactory.createBoard()
        board2.id = 'board2'
        board2.teamId = 'team-id'

        const staleCategory = TestBlockFactory.createCategoryBoards()
        staleCategory.id = 'other_category'
        staleCategory.name = 'Category 1'
        staleCategory.teamID = 'other-team-id'
        staleCategory.boardMetadata = []

        const staleDefaultCategory = TestBlockFactory.createCategoryBoards()
        staleDefaultCategory.id = 'other_default_category'
        staleDefaultCategory.name = 'Boards'
        staleDefaultCategory.teamID = 'other-team-id'
        staleDefaultCategory.boardMetadata = []

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board2.id,
                boards: {
                    [board2.id]: board2,
                },
                myBoardMemberships: {
                    [board2.id]: board2,
                },
            },
            cards: {
                cards: {
                    card_id_1: {title: 'Card'},
                },
                current: 'card_id_1',
            },
            views: {
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    staleCategory,
                    staleDefaultCategory,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        mockedOctoClient.getSidebarCategories.mockResolvedValue([])

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        render(component)

        // getSidebarCategories being called confirms the fetch effect ran and its
        // ".then(setInitialized(true))" will flip the flag the move effect waits
        // on, so the assertion below isn't vacuous. Drain the microtasks + one
        // macrotask so that chain and the follow-up effect have definitely run.
        await waitFor(() => expect(mockedOctoClient.getSidebarCategories).toHaveBeenCalled())
        await act(async () => {
            await Promise.resolve()
            await new Promise((resolve) => setTimeout(resolve, 0))
        })

        expect(mockedOctoClient.moveBoardToCategory).not.toHaveBeenCalled()
    })

    test('shouldnt assign default category when category list mixes current and other team', async () => {
        // A websocket update can unshift a current-team category in front of
        // stale categories from an in-flight fetch for the previous team.
        // Checking only the first category would miss that mixed state and move
        // the current team's board into the previous team's "Boards" category.
        const board2 = TestBlockFactory.createBoard()
        board2.id = 'board2'
        board2.teamId = 'team-id'

        const currentTeamCategory = TestBlockFactory.createCategoryBoards()
        currentTeamCategory.id = 'current_category'
        currentTeamCategory.name = 'Category 1'
        currentTeamCategory.teamID = 'team-id'
        currentTeamCategory.boardMetadata = []

        const staleDefaultCategory = TestBlockFactory.createCategoryBoards()
        staleDefaultCategory.id = 'other_default_category'
        staleDefaultCategory.name = 'Boards'
        staleDefaultCategory.teamID = 'other-team-id'
        staleDefaultCategory.boardMetadata = []

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board2.id,
                boards: {
                    [board2.id]: board2,
                },
                myBoardMemberships: {
                    [board2.id]: board2,
                },
            },
            cards: {
                cards: {
                    card_id_1: {title: 'Card'},
                },
                current: 'card_id_1',
            },
            views: {
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    currentTeamCategory,
                    staleDefaultCategory,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        mockedOctoClient.getSidebarCategories.mockResolvedValue([])

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        render(component)

        // getSidebarCategories being called confirms the fetch effect ran and its
        // ".then(setInitialized(true))" will flip the flag the move effect waits
        // on, so the assertion below isn't vacuous. Drain the microtasks + one
        // macrotask so that chain and the follow-up effect have definitely run.
        await waitFor(() => expect(mockedOctoClient.getSidebarCategories).toHaveBeenCalled())
        await act(async () => {
            await Promise.resolve()
            await new Promise((resolve) => setTimeout(resolve, 0))
        })

        expect(mockedOctoClient.moveBoardToCategory).not.toHaveBeenCalled()
    })

    test('shouldnt assign default category when current board belongs to a different team', async () => {
        // The current board belongs to a team other than the one being viewed
        // (stale board left over from the previous team during a switch).
        const board2 = TestBlockFactory.createBoard()
        board2.id = 'board2'
        board2.teamId = 'other-team-id'

        const store = mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board2.id,
                boards: {
                    [board2.id]: board2,
                },
                myBoardMemberships: {
                    [board2.id]: board2,
                },
            },
            cards: {
                cards: {
                    card_id_1: {title: 'Card'},
                },
                current: 'card_id_1',
            },
            views: {
                views: [],
            },
            users: {
                me: {
                    id: 'user_id_1',
                    props: {},
                },
            },
            sidebar: {
                categoryAttributes: [
                    categoryAttribute1,
                    defaultCategory,
                ],
                hiddenBoardIDs: [],
            },
        })

        const history = createMemoryHistory()
        const onBoardTemplateSelectorOpen = jest.fn()

        mockedOctoClient.getSidebarCategories.mockResolvedValue([])

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
                </Router>
            </ReduxProvider>,
        )
        render(component)

        // getSidebarCategories being called confirms the fetch effect ran and its
        // ".then(setInitialized(true))" will flip the flag the move effect waits
        // on, so the assertion below isn't vacuous. Drain the microtasks + one
        // macrotask so that chain and the follow-up effect have definitely run.
        await waitFor(() => expect(mockedOctoClient.getSidebarCategories).toHaveBeenCalled())
        await act(async () => {
            await Promise.resolve()
            await new Promise((resolve) => setTimeout(resolve, 0))
        })

        expect(mockedOctoClient.moveBoardToCategory).not.toHaveBeenCalled()
    })

    // TODO: Fix this later
    // test('global templates', () => {
    //     const store = mockStore({
    //         teams: {
    //             current: {id: 'team-id'},
    //         },
    //         boards: {
    //             boards: [],
    //             templates: [
    //                 {id: '1', title: 'Template 1', fields: {icon: '🚴🏻‍♂️'}},
    //                 {id: '2', title: 'Template 2', fields: {icon: '🚴🏻‍♂️'}},
    //                 {id: '3', title: 'Template 3', fields: {icon: '🚴🏻‍♂️'}},
    //                 {id: '4', title: 'Template 4', fields: {icon: '🚴🏻‍♂️'}},
    //             ],
    //         },
    //         views: {
    //             views: [],
    //         },
    //         users: {
    //             me: {},
    //         },
    //         globalTemplates: {
    //             value: [],
    //         },
    //         sidebar: {
    //             categoryAttributes: [
    //                 categoryAttribute1,
    //             ],
    //         },
    //     })

    //     const history = createMemoryHistory()

    //     const component = wrapIntl(
    //         <ReduxProvider store={store}>
    //             <Router history={history}>
    //                 <Sidebar onBoardTemplateSelectorOpen={onBoardTemplateSelectorOpen}/>
    //             </Router>
    //         </ReduxProvider>,
    //     )
    //     const {container} = render(component)
    //     expect(container).toMatchSnapshot()

    //     const addBoardButton = container.querySelector('.SidebarAddBoardMenu > .MenuWrapper')
    //     expect(addBoardButton).toBeDefined()
    //     userEvent.click(addBoardButton as Element)
    //     const templates = container.querySelectorAll('.SidebarAddBoardMenu > .MenuWrapper div:not(.hideOnWidescreen).menu-options .menu-name')
    //     expect(templates).toBeDefined()

    //     console.log(templates[0].innerHTML)
    //     console.log(templates[1].innerHTML)

    //     // 4 mocked templates, one "Select a template", one "Empty Board" and one "+ New Template"
    //     expect(templates.length).toBe(7)
    // })
})
