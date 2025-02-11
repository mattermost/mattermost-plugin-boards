// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {render, screen} from '@testing-library/react'
import configureStore from 'redux-mock-store'
import '@testing-library/jest-dom'
import userEvents from '@testing-library/user-event'

import 'isomorphic-fetch'
import {mocked} from 'jest-mock'

import {TestBlockFactory} from '../../test/testBlockFactory'
import {FetchMock} from '../../test/fetchMock'
import {BoardView} from '../../blocks/boardView'

import {IUser} from '../../user'

import {Utils, IDType} from '../../utils'

import {wrapDNDIntl} from '../../testUtils'

import Mutator from '../../mutator'

import Table from './table'

global.fetch = FetchMock.fn

beforeEach(() => {
    FetchMock.fn.mockReset()
})

jest.mock('../../mutator')
jest.mock('../../utils')
jest.mock('../../telemetry/telemetryClient')
const mockedMutator = mocked(Mutator, true)

describe('components/table/Table', () => {
    const board = TestBlockFactory.createBoard()
    const view = TestBlockFactory.createBoardView(board)
    view.fields.viewType = 'table'
    view.fields.groupById = undefined
    view.fields.visiblePropertyIds = ['property1', 'property2']

    const view2 = TestBlockFactory.createBoardView(board)
    view2.fields.sortOptions = []

    const card = TestBlockFactory.createCard(board)
    const cardTemplate = TestBlockFactory.createCard(board)
    cardTemplate.fields.isTemplate = true

    const state = {
        users: {
            boardUsers: {
                'user-id-1': {username: 'username_1'} as IUser,
                'user-id-2': {username: 'username_2'} as IUser,
                'user-id-3': {username: 'username_3'} as IUser,
                'user-id-4': {username: 'username_4'} as IUser,
            },
        },
        comments: {
            comments: {},
            commentsByCard: {},
        },
        contents: {
            contents: {},
            contentsByCard: {},
        },
        cards: {
            cards: {
                [card.id]: card,
            },
        },
        teams: {
            current: {id: 'team-id'},
        },
        boards: {
            current: board.id,
            boards: {
                [board.id]: board,
            },
            myBoardMemberships: {
                [board.id]: {userId: 'user_id_1', schemeAdmin: true},
            },
        },
    }

    test('should match snapshot', async () => {
        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore(state)

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card]}
                    views={[view, view2]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot without permissions', async () => {
        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore({...state, teams: {current: undefined}})

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card]}
                    views={[view, view2]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot, read-only', async () => {
        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore(state)

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card]}
                    views={[view, view2]}
                    selectedCardIds={[]}
                    readonly={true}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot with GroupBy', async () => {
        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore(state)

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={{...view, fields: {...view.fields, groupById: 'property1'}} as BoardView}
                    visibleGroups={[{option: {id: '', value: 'test', color: ''}, cards: []}]}
                    groupByProperty={{
                        id: '',
                        name: 'Property 1',
                        type: 'text',
                        options: [{id: 'property1', value: 'Property 1', color: ''}],
                    }}
                    cards={[card]}
                    views={[view, view2]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('limited card in table view', () => {
        const callback = jest.fn()
        const addCard = jest.fn()
        const boardTest = TestBlockFactory.createBoard()
        const card1 = TestBlockFactory.createCard(boardTest)
        const card2 = TestBlockFactory.createCard(boardTest)
        const mockStore = configureStore([])

        const stateTest = {
            comments: {
                comments: {},
            },
            contents: {
                contents: {},
            },
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: boardTest.id,
                boards: {
                    [boardTest.id]: boardTest,
                },
                myBoardMemberships: {
                    [boardTest.id]: {userId: 'user_id_1', schemeAdmin: true},
                },
            },
        }

        const storeTest = mockStore(stateTest)
        card.limited = true

        const component = wrapDNDIntl(
            <ReduxProvider store={storeTest}>
                <Table
                    board={boardTest}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view, view2]}
                    selectedCardIds={[]}
                    readonly={true}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={2}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )
        const {container, getByTitle} = render(component)
        expect(getByTitle('hidden-card-count')).toHaveTextContent('2')
        expect(container).toMatchSnapshot()
    })
})

describe('components/table/Table extended', () => {
    const state = {
        users: {
            boardUsers: {
                'user-id-1': {username: 'username_1'} as IUser,
                'user-id-2': {username: 'username_2'} as IUser,
                'user-id-3': {username: 'username_3'} as IUser,
                'user-id-4': {username: 'username_4'} as IUser,
            },
        },
        comments: {
            comments: {},
            commentsByCard: {},
        },
        contents: {
            contents: {},
            contentsByCard: {},
        },
        cards: {
            cards: {},
        },
        teams: {
            current: {id: 'team-id'},
        },
        boards: {
            current: 'board_id',
            boards: {
                board_id: {id: 'board_id'},
            },
            myBoardMemberships: {
                board_id: {userId: 'user_id_1', schemeAdmin: true},
            },
        },
        clientConfig: {
            value: {
                teammateNameDisplay: 'username',
            },
        },
    }

    test('should match snapshot with CreatedAt', async () => {
        const board = TestBlockFactory.createBoard()

        const dateCreatedId = Utils.createGuid(IDType.User)
        board.cardProperties.push({
            id: dateCreatedId,
            name: 'Date Created',
            type: 'createdTime',
            options: [],
        })

        const card1 = TestBlockFactory.createCard(board)
        card1.createAt = Date.parse('15 Jun 2021 16:22:00')

        const card2 = TestBlockFactory.createCard(board)
        card2.createAt = Date.parse('15 Jun 2021 16:22:00')

        const view = TestBlockFactory.createBoardView(board)
        view.fields.viewType = 'table'
        view.fields.groupById = undefined
        view.fields.visiblePropertyIds = ['property1', 'property2', dateCreatedId]

        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore({
            ...state,
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board.id,
                boards: {
                    [board.id]: board,
                },
                myBoardMemberships: {
                    [board.id]: {userId: 'user_id_1', schemeAdmin: true},
                },
            },
        })

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot with UpdatedAt', async () => {
        const board = TestBlockFactory.createBoard()
        const dateUpdatedId = Utils.createGuid(IDType.User)
        board.cardProperties.push({
            id: dateUpdatedId,
            name: 'Date Updated',
            type: 'updatedTime',
            options: [],
        })

        const card1 = TestBlockFactory.createCard(board)
        card1.updateAt = Date.parse('20 Jun 2021 12:22:00')

        const card2 = TestBlockFactory.createCard(board)
        card2.updateAt = Date.parse('20 Jun 2021 12:22:00')

        const card2Comment = TestBlockFactory.createCard(board)
        card2Comment.parentId = card2.id
        card2Comment.type = 'comment'
        card2Comment.updateAt = Date.parse('21 Jun 2021 15:23:00')

        const card2Text = TestBlockFactory.createCard(board)
        card2Text.parentId = card2.id
        card2Text.type = 'text'
        card2Text.updateAt = Date.parse('22 Jun 2021 11:23:00')

        card2.fields.contentOrder = [card2Text.id]

        const view = TestBlockFactory.createBoardView(board)
        view.fields.viewType = 'table'
        view.fields.groupById = undefined
        view.fields.visiblePropertyIds = ['property1', 'property2', dateUpdatedId]

        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore({
            ...state,
            comments: {
                comments: {
                    [card2Comment.id]: card2Comment,
                },
                commentsByCard: {
                    [card2.id]: [card2Comment],
                },
            },
            contents: {
                contents: {
                    [card2Text.id]: card2Text,
                },
                contentsByCard: {
                    [card2.id]: [card2Text],
                },
            },
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
        })

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )
        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot with CreatedBy', async () => {
        const board = TestBlockFactory.createBoard()

        const createdById = Utils.createGuid(IDType.User)
        board.cardProperties.push({
            id: createdById,
            name: 'Created By',
            type: 'createdBy',
            options: [],
        })

        const card1 = TestBlockFactory.createCard(board)
        card1.createdBy = 'user-id-1'

        const card2 = TestBlockFactory.createCard(board)
        card2.createdBy = 'user-id-2'

        const view = TestBlockFactory.createBoardView(board)
        view.fields.viewType = 'table'
        view.fields.groupById = undefined
        view.fields.visiblePropertyIds = ['property1', 'property2', createdById]

        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore({
            ...state,
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
        })

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot with UpdatedBy', async () => {
        const board = TestBlockFactory.createBoard()

        const modifiedById = Utils.createGuid(IDType.User)
        board.cardProperties.push({
            id: modifiedById,
            name: 'Last Modified By',
            type: 'updatedBy',
            options: [],
        })

        const card1 = TestBlockFactory.createCard(board)
        card1.modifiedBy = 'user-id-1'
        card1.updateAt = Date.parse('15 Jun 2021 16:22:00')

        const card1Text = TestBlockFactory.createCard(board)
        card1Text.parentId = card1.id
        card1Text.type = 'text'
        card1Text.modifiedBy = 'user-id-4'
        card1Text.updateAt = Date.parse('16 Jun 2021 16:22:00')

        card1.fields.contentOrder = [card1Text.id]

        const card2 = TestBlockFactory.createCard(board)
        card2.modifiedBy = 'user-id-2'
        card2.updateAt = Date.parse('15 Jun 2021 16:22:00')

        const card2Comment = TestBlockFactory.createCard(board)
        card2Comment.parentId = card2.id
        card2Comment.type = 'comment'
        card2Comment.modifiedBy = 'user-id-3'
        card2.updateAt = Date.parse('16 Jun 2021 16:22:00')

        const view = TestBlockFactory.createBoardView(board)
        view.fields.viewType = 'table'
        view.fields.groupById = undefined
        view.fields.visiblePropertyIds = ['property1', 'property2', modifiedById]

        const callback = jest.fn()
        const addCard = jest.fn()

        const mockStore = configureStore([])
        const store = mockStore({
            ...state,
            comments: {
                comments: {
                    [card2Comment.id]: card2Comment,
                },
                commentsByCard: {
                    [card2.id]: [card2Comment],
                },
            },
            contents: {
                contents: {
                    [card1Text.id]: card1Text,
                },
                contentsByCard: {
                    [card1.id]: [card1Text],
                },
            },
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
        })

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={callback}
                    addCard={addCard}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should delete snapshot', async () => {
        const board = TestBlockFactory.createBoard()

        const modifiedById = Utils.createGuid(IDType.User)
        board.cardProperties.push({
            id: modifiedById,
            name: 'Last Modified By',
            type: 'updatedBy',
            options: [],
        })
        const card1 = TestBlockFactory.createCard(board)
        card1.title = 'card1'
        const card2 = TestBlockFactory.createCard(board)
        card2.title = 'card2'
        const view = TestBlockFactory.createBoardView(board)
        view.fields.viewType = 'table'
        view.fields.groupById = undefined
        view.fields.visiblePropertyIds = ['property1', 'property2', modifiedById]
        const mockStore = configureStore([])
        const store = mockStore({
            ...state,
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
        })

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={jest.fn()}
                    addCard={jest.fn()}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )

        const {getByTitle, getByRole, getAllByTitle} = render(component)
        const card1Name = getByTitle(card1.title)
        userEvents.hover(card1Name)
        const menuBtn = getAllByTitle('MenuBtn')
        userEvents.click(menuBtn[0])
        const deleteBtn = getByRole('button', {name: 'Delete'})
        userEvents.click(deleteBtn)
        const dailogDeleteBtn = screen.getByRole('button', {name: 'Delete'})
        userEvents.click(dailogDeleteBtn)
        expect(mockedMutator.deleteBlock).toBeCalledTimes(1)
    })

    test('should have Duplicate Button', async () => {
        const board = TestBlockFactory.createBoard()

        const modifiedById = Utils.createGuid(IDType.User)
        board.cardProperties.push({
            id: modifiedById,
            name: 'Last Modified By',
            type: 'updatedBy',
            options: [],
        })
        const card1 = TestBlockFactory.createCard(board)
        card1.title = 'card1'
        const card2 = TestBlockFactory.createCard(board)
        card2.title = 'card2'
        const view = TestBlockFactory.createBoardView(board)
        view.fields.viewType = 'table'
        view.fields.groupById = undefined
        view.fields.visiblePropertyIds = ['property1', 'property2', modifiedById]
        const mockStore = configureStore([])
        const store = mockStore({
            ...state,
            cards: {
                cards: {
                    [card1.id]: card1,
                    [card2.id]: card2,
                },
            },
        })

        const component = wrapDNDIntl(
            <ReduxProvider store={store}>
                <Table
                    board={board}
                    activeView={view}
                    visibleGroups={[]}
                    cards={[card1, card2]}
                    views={[view]}
                    selectedCardIds={[]}
                    readonly={false}
                    cardIdToFocusOnRender=''
                    showCard={jest.fn()}
                    addCard={jest.fn()}
                    onCardClicked={jest.fn()}
                    hiddenCardsCount={0}
                    showHiddenCardCountNotification={jest.fn()}
                />
            </ReduxProvider>,
        )

        const {getByTitle, getByRole, getAllByTitle, container} = render(component)
        const card1Name = getByTitle(card1.title)
        userEvents.hover(card1Name)
        const menuBtn = getAllByTitle('MenuBtn')
        userEvents.click(menuBtn[0])
        const duplicateBtn = getByRole('button', {name: 'Duplicate'})
        expect(duplicateBtn).not.toBe(null)
        userEvents.click(duplicateBtn)
        expect(mockedMutator.duplicateCard).toBeCalledTimes(1)
        expect(container).toMatchSnapshot()
    })
})
