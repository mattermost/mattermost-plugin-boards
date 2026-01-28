// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {render, screen, waitFor} from '@testing-library/react'
import {mocked} from 'jest-mock'
import '@testing-library/jest-dom'
import configureStore from 'redux-mock-store'
import {Provider as ReduxProvider} from 'react-redux'

import {wrapIntl} from '../../testUtils'
import {TestBlockFactory} from '../../test/testBlockFactory'
import octoClient from '../../octoClient'
import {CardRelation, RelationType} from '../../blocks/cardRelation'

import CardRelations from './cardRelations'

jest.mock('../../octoClient')
const mockedOctoClient = mocked(octoClient, true)

describe('components/cardDetail/CardRelations', () => {
    const board = TestBlockFactory.createBoard()
    
    // Add status property with options
    const statusProperty = {
        id: 'status-prop-id',
        name: 'Status',
        type: 'select' as const,
        options: [
            {id: 'status-todo', value: 'To Do', color: 'propColorGray'},
            {id: 'status-in-progress', value: 'In Progress', color: 'propColorBlue'},
            {id: 'status-done', value: 'Done', color: 'propColorGreen'},
        ],
    }
    
    // Add assignee property
    const assigneeProperty = {
        id: 'assignee-prop-id',
        name: 'Assignee',
        type: 'person' as const,
        options: [],
    }
    
    board.cardProperties = [statusProperty, assigneeProperty]

    const card = TestBlockFactory.createCard(board)
    card.id = 'card-1'
    card.title = 'Test Card'

    const relatedCard = TestBlockFactory.createCard(board)
    relatedCard.id = 'card-2'
    relatedCard.title = 'Related Card'
    relatedCard.code = 'FB-123'
    relatedCard.fields.properties = {
        'status-prop-id': 'status-in-progress',
        'assignee-prop-id': 'user-id-1',
    }

    const testUser = {
        id: 'user-id-1',
        username: 'john.doe',
        email: 'john@example.com',
        nickname: 'John',
        firstname: 'John',
        lastname: 'Doe',
        props: {},
        create_at: Date.now(),
        update_at: Date.now(),
        is_bot: false,
        is_guest: false,
        roles: 'system_user',
    }

    const mockRelation: CardRelation = {
        id: 'relation-1',
        sourceCardId: card.id,
        targetCardId: relatedCard.id,
        relationType: RelationType.RelatesTo,
        createdBy: 'user-id-1',
        createAt: Date.now(),
        updateAt: Date.now(),
        boardId: board.id,
    }

    const mockStore = configureStore([])

    beforeEach(() => {
        jest.clearAllMocks()
    })

    function createStore() {
        return mockStore({
            users: {
                boardUsers: {
                    'user-id-1': testUser,
                },
            },
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                boards: {
                    [board.id]: board,
                },
                current: board.id,
                myBoardMemberships: {
                    [board.id]: {userId: 'user-id-1', schemeAdmin: true},
                },
            },
            cards: {
                cards: {
                    [card.id]: card,
                    [relatedCard.id]: relatedCard,
                },
                current: card.id,
            },
            clientConfig: {
                value: {},
            },
        })
    }

    function renderComponent(props?: Partial<React.ComponentProps<typeof CardRelations>>) {
        const store = createStore()
        
        return render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <CardRelations
                        card={card}
                        boardId={board.id}
                        readonly={false}
                        {...props}
                    />
                </ReduxProvider>
            )
        )
    }

    it('should render loading state initially', () => {
        mockedOctoClient.getCardRelations.mockReturnValue(new Promise(() => {})) // Never resolves
        
        renderComponent()
        
        expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should render empty state when no relations', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([])
        
        renderComponent()
        
        await waitFor(() => {
            expect(screen.getByText('No relations yet')).toBeInTheDocument()
        })
    })

    it('should render relation with status and assignee', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, relatedCard])
        
        renderComponent()
        
        await waitFor(() => {
            // Check card title is displayed
            expect(screen.getByText('Related Card')).toBeInTheDocument()
        })
        
        // Check card code is displayed
        expect(screen.getByText('FB-123')).toBeInTheDocument()
        
        // Check status label is displayed
        expect(screen.getByText('In Progress')).toBeInTheDocument()
        
        // Check assignee avatar is rendered (by alt text)
        expect(screen.getByAltText('john.doe')).toBeInTheDocument()
    })

    it('should render relation type', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, relatedCard])
        
        renderComponent()
        
        await waitFor(() => {
            expect(screen.getByText('Relates to')).toBeInTheDocument()
        })
    })

    it('should show add button when not readonly', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([])
        
        renderComponent({readonly: false})
        
        await waitFor(() => {
            expect(screen.getByText('Add relation')).toBeInTheDocument()
        })
    })

    it('should hide add button when readonly', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([])
        
        renderComponent({readonly: true})
        
        await waitFor(() => {
            expect(screen.queryByText('Add relation')).not.toBeInTheDocument()
        })
    })

    it('should hide delete button when readonly', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, relatedCard])
        
        renderComponent({readonly: true})
        
        await waitFor(() => {
            expect(screen.getByText('Related Card')).toBeInTheDocument()
        })
        
        expect(screen.queryByRole('button', {name: 'Remove relation'})).not.toBeInTheDocument()
    })

    it('should show delete button when not readonly', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, relatedCard])
        
        renderComponent({readonly: false})
        
        await waitFor(() => {
            expect(screen.getByText('Related Card')).toBeInTheDocument()
        })
        
        expect(screen.getByRole('button', {name: 'Remove relation'})).toBeInTheDocument()
    })

    it('should display relations count', async () => {
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, relatedCard])
        
        renderComponent()
        
        await waitFor(() => {
            expect(screen.getByText('1')).toBeInTheDocument()
        })
    })

    it('should handle relation without status property gracefully', async () => {
        const cardWithoutStatus = {...relatedCard, fields: {...relatedCard.fields, properties: {}}}
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, cardWithoutStatus])
        
        renderComponent()
        
        await waitFor(() => {
            expect(screen.getByText('Related Card')).toBeInTheDocument()
        })
        
        // Status should not be displayed
        expect(screen.queryByText('In Progress')).not.toBeInTheDocument()
    })

    it('should use first select property when status is renamed', async () => {
        // Create board with renamed status property
        const boardWithRenamedStatus = {...board}
        boardWithRenamedStatus.cardProperties = [
            {
                ...statusProperty,
                name: 'Task State', // Renamed from "Status"
            },
            assigneeProperty,
        ]
        
        const storeWithRenamedStatus = mockStore({
            users: {
                boardUsers: {'user-id-1': testUser},
            },
            teams: {
                current: {id: 'team-id'},
            },
            boards: {
                boards: {[boardWithRenamedStatus.id]: boardWithRenamedStatus},
                current: boardWithRenamedStatus.id,
                myBoardMemberships: {
                    [boardWithRenamedStatus.id]: {userId: 'user-id-1', schemeAdmin: true},
                },
            },
            cards: {
                cards: {[card.id]: card, [relatedCard.id]: relatedCard},
                current: card.id,
            },
            clientConfig: {value: {}},
        })
        
        mockedOctoClient.getCardRelations.mockResolvedValue([mockRelation])
        mockedOctoClient.getAllBlocks.mockResolvedValue([card, relatedCard])
        
        render(
            wrapIntl(
                <ReduxProvider store={storeWithRenamedStatus}>
                    <CardRelations
                        card={card}
                        boardId={boardWithRenamedStatus.id}
                        readonly={false}
                    />
                </ReduxProvider>
            )
        )
        
        await waitFor(() => {
            // Status should still be displayed even with renamed property
            expect(screen.getByText('In Progress')).toBeInTheDocument()
        })
    })
})
