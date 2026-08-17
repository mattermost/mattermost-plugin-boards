// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import configureStore from 'redux-mock-store'

import {createMemoryHistory} from 'history'
import {Provider as ReduxProvider} from 'react-redux'
import {Router} from 'react-router-dom'

import {render, waitFor} from '@testing-library/react'

import thunk from 'redux-thunk'

import {mocked} from 'jest-mock'

import {DropResult} from 'react-beautiful-dnd'

import {mockMatchMedia, wrapIntl} from '../../testUtils'

import {TestBlockFactory} from '../../test/testBlockFactory'
import octoClient from '../../octoClient'

import Sidebar from './sidebar'

const mockDnd: {onDragEnd?: (result: DropResult) => void} = {}

type MockRenderChildren = {
    children: (provided: unknown, snapshot: unknown) => React.ReactNode
}

jest.mock('react-beautiful-dnd', () => {
    const react = jest.requireActual('react')

    return {
        DragDropContext: (props: {onDragEnd: (result: DropResult) => void, children: React.ReactNode}) => {
            mockDnd.onDragEnd = props.onDragEnd
            return react.createElement(react.Fragment, null, props.children)
        },
        Droppable: (props: MockRenderChildren) => props.children(
            {innerRef: () => null, droppableProps: {}, placeholder: null},
            {isDraggingOver: false},
        ),
        Draggable: (props: MockRenderChildren) => props.children(
            {innerRef: () => null, draggableProps: {}, dragHandleProps: {}},
            {isDragging: false},
        ),
    }
})

jest.mock('../../octoClient')
const mockedOctoClient = mocked(octoClient, true)

beforeAll(() => {
    mockMatchMedia({matches: true})
})

describe('components/sidebar drag and drop ordering', () => {
    const mockStore = configureStore([thunk])

    const makeBoard = (id: string, title: string) => {
        const board = TestBlockFactory.createBoard()
        board.id = id
        board.title = title
        return board
    }

    const hiddenBoard = makeBoard('hidden_board', 'Hidden board')
    const board1 = makeBoard('board1', 'Board 1')
    const board2 = makeBoard('board2', 'Board 2')
    const board3 = makeBoard('board3', 'Board 3')

    const allBoards = [hiddenBoard, board1, board2, board3]

    const buildStore = () => {
        const category = TestBlockFactory.createCategoryBoards()
        category.id = 'category1'
        category.name = 'Category 1'
        category.boardMetadata = [
            {boardID: hiddenBoard.id, hidden: true},
            {boardID: board1.id, hidden: false},
            {boardID: board2.id, hidden: false},
            {boardID: board3.id, hidden: false},
        ]

        const boardsByID: Record<string, unknown> = {}
        allBoards.forEach((board) => {
            boardsByID[board.id] = board
        })

        return mockStore({
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                current: board1.id,
                boards: boardsByID,
                myBoardMemberships: boardsByID,
            },
            cards: {
                cards: {},
                current: '',
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
                categoryAttributes: [category],
                hiddenBoardIDs: [],
            },
        })
    }

    const renderSidebar = () => {
        const history = createMemoryHistory()

        return render(wrapIntl(
            <ReduxProvider store={buildStore()}>
                <Router history={history}>
                    <Sidebar onBoardTemplateSelectorOpen={jest.fn()}/>
                </Router>
            </ReduxProvider>,
        ))
    }

    beforeEach(() => {
        jest.clearAllMocks()
        mockDnd.onDragEnd = undefined
        mockedOctoClient.reorderSidebarCategoryBoards.mockResolvedValue([])
    })

    test('renders only visible boards as draggables', () => {
        const {container} = renderSidebar()

        const sidebarBoards = container.getElementsByClassName('SidebarBoardItem')
        expect(sidebarBoards.length).toBe(3)
    })

    test('reorders the dragged board when the category contains a hidden board', async () => {
        renderSidebar()
        expect(mockDnd.onDragEnd).toBeDefined()

        mockDnd.onDragEnd!({
            type: 'board',
            draggableId: board3.id,
            source: {droppableId: 'category1', index: 2},
            destination: {droppableId: 'category1', index: 0},
        } as DropResult)

        await waitFor(() => expect(mockedOctoClient.reorderSidebarCategoryBoards).toBeCalledWith(
            'team-id',
            'category1',
            [hiddenBoard.id, board3.id, board1.id, board2.id],
        ))
    })

    test('reorders to the end of the category without dropping any board', async () => {
        renderSidebar()

        mockDnd.onDragEnd!({
            type: 'board',
            draggableId: board1.id,
            source: {droppableId: 'category1', index: 0},
            destination: {droppableId: 'category1', index: 2},
        } as DropResult)

        await waitFor(() => expect(mockedOctoClient.reorderSidebarCategoryBoards).toBeCalledWith(
            'team-id',
            'category1',
            [hiddenBoard.id, board2.id, board3.id, board1.id],
        ))
    })
})
