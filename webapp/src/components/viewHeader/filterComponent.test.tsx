// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {render, screen} from '@testing-library/react'
import {Provider as ReduxProvider} from 'react-redux'

import {mocked} from 'jest-mock'
import '@testing-library/jest-dom'

import userEvent from '@testing-library/user-event'

import {FilterClause} from '../../blocks/filterClause'
import {FilterGroup} from '../../blocks/filterGroup'
import {Board, PropertyTypeEnum} from '../../blocks/board'

import {TestBlockFactory} from '../../test/testBlockFactory'
import mutator from '../../mutator'
import propsRegistry from '../../properties'
import {Utils} from '../../utils'

import {wrapIntl, mockStateStore} from '../../testUtils'

import FilterComponenet, {nextFilterClause} from './filterComponent'

jest.mock('../../mutator')
const mockedMutator = mocked(mutator)

const board = TestBlockFactory.createBoard()
const activeView = TestBlockFactory.createBoardView(board)

const filter: FilterClause = {
    propertyId: board.cardProperties[0].id,
    condition: 'includes',
    values: ['Status'],
}
const unknownFilter: FilterClause = {
    propertyId: 'unknown',
    condition: 'includes',
    values: [],
}

const state = {
    users: {
        me: {
            id: 'user-id-1',
            username: 'username_1',
        },
    },
}
const store = mockStateStore([], state)
describe('components/viewHeader/filterComponent', () => {
    beforeEach(() => {
        jest.clearAllMocks()
        board.cardProperties[0].options = [{id: 'Status', value: 'Status', color: ''}]
        activeView.fields.filter.filters = [filter]
    })
    afterEach(() => {
        mockedMutator.changeViewFilter.mockReset()
    })
    test('return filterComponent', () => {
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getAllByRole('button', {name: 'menuwrapper'})[0]
        userEvent.click(buttonElement)
        expect(container).toMatchSnapshot()
    })
    test('return filterComponent and add Filter', () => {
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getAllByRole('button', {name: 'menuwrapper'})[0]
        userEvent.click(buttonElement)
        expect(container).toMatchSnapshot()
        const buttonAdd = screen.getByText('+ Add filter')
        userEvent.click(buttonAdd)
        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
    })

    test('return filterComponent and filter by status', () => {
        activeView.fields.filter.filters = [unknownFilter]
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getAllByRole('button', {name: 'menuwrapper'})[0]
        userEvent.click(buttonElement)
        expect(container).toMatchSnapshot()
        const buttonStatus = screen.getByRole('button', {name: 'Status'})
        userEvent.click(buttonStatus)
        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
    })

    test('return filterComponent and click is empty', () => {
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getAllByRole('button', {name: 'menuwrapper'})[1]
        userEvent.click(buttonElement)
        expect(container).toMatchSnapshot()
        const buttonNotInclude = screen.getByRole('button', {name: 'is empty'})
        userEvent.click(buttonNotInclude)
        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
    })

    test('add filter appends a real property, not an empty clause', () => {
        activeView.fields.filter.filters = []
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonAdd = screen.getByRole('button', {name: '+ Add filter'})
        userEvent.click(buttonAdd)

        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
        const newFilterGroup = mockedMutator.changeViewFilter.mock.calls[0][3] as FilterGroup
        const added = newFilterGroup.filters[newFilterGroup.filters.length - 1] as FilterClause
        expect(added.propertyId).not.toBe('')
        expect(added.propertyId).toBe(board.cardProperties[0].id)
    })

    test('disables + Add filter once every property and the title are in use', () => {
        activeView.fields.filter.filters = [
            ...board.cardProperties.map((p): FilterClause => ({propertyId: p.id, condition: 'includes', values: []})),
            {propertyId: 'title', condition: 'is', values: []},
        ]
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonAdd = screen.getByRole('button', {name: '+ Add filter'})
        expect(buttonAdd).toBeDisabled()
        userEvent.click(buttonAdd)
        expect(mockedMutator.changeViewFilter).not.toHaveBeenCalled()
    })

    test('clicking + Add filter past the available properties never adds an (unknown) row (MM-58100)', () => {
        activeView.fields.filter.filters = []
        mockedMutator.changeViewFilter.mockImplementation((_boardId, _viewId, _oldFilter, newFilter) => {
            activeView.fields.filter = newFilter as FilterGroup
            return Promise.resolve()
        })

        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )

        const buttonAdd = screen.getByRole('button', {name: '+ Add filter'})
        for (let i = 0; i < 12; i++) {
            userEvent.click(buttonAdd)
        }

        const finalFilters = activeView.fields.filter.filters as FilterClause[]
        const propertyIds = finalFilters.map((f) => f.propertyId)
        const filterable = board.cardProperties.filter((p) => propsRegistry.get(p.type).canFilter)

        expect(finalFilters).toHaveLength(filterable.length + 1)
        expect(propertyIds).toContain('title')
        expect(propertyIds).not.toContain('')
        expect(new Set(propertyIds).size).toBe(propertyIds.length)
        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(filterable.length + 1)
    })

    test('clears pending add state when the active view changes', () => {
        activeView.fields.filter.filters = []
        mockedMutator.changeViewFilter.mockResolvedValue(undefined)

        const {rerender} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )

        const buttonAdd = screen.getByRole('button', {name: '+ Add filter'})
        userEvent.click(buttonAdd)
        expect(buttonAdd).toBeDisabled()

        const otherView = TestBlockFactory.createBoardView(board)
        otherView.fields.filter.filters = []
        rerender(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={otherView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )

        expect(screen.getByRole('button', {name: '+ Add filter'})).toBeEnabled()
    })

    test('rapid clicks before the view updates only persist one new clause', () => {
        activeView.fields.filter.filters = []
        mockedMutator.changeViewFilter.mockResolvedValue(undefined)

        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )

        const buttonAdd = screen.getByRole('button', {name: '+ Add filter'})
        userEvent.click(buttonAdd)
        userEvent.click(buttonAdd)
        userEvent.click(buttonAdd)

        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
        const newFilterGroup = mockedMutator.changeViewFilter.mock.calls[0][3] as FilterGroup
        expect(newFilterGroup.filters).toHaveLength(1)
    })

    test('add filter on a board with a person property uses a valid person condition', () => {
        const personBoard = TestBlockFactory.createBoard()
        personBoard.cardProperties = [{id: 'assignee', name: 'Assignee', type: 'person', options: []}]
        activeView.fields.filter.filters = []
        const assertFailure = jest.spyOn(Utils, 'assertFailure')

        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={personBoard}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        userEvent.click(screen.getByRole('button', {name: '+ Add filter'}))

        expect(assertFailure).not.toHaveBeenCalled()
        const newFilterGroup = mockedMutator.changeViewFilter.mock.calls[0][3] as FilterGroup
        const added = newFilterGroup.filters[0] as FilterClause
        expect(added.propertyId).toBe('assignee')
        expect(added.condition).toBe('includes')
        assertFailure.mockRestore()
    })

    test('add filter ignores nested filter groups when choosing the next property', () => {
        activeView.fields.filter.filters = [
            {operation: 'or', filters: []} as FilterGroup,
            {propertyId: board.cardProperties[0].id, condition: 'includes', values: []},
        ]
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterComponenet
                        board={board}
                        activeView={activeView}
                        onClose={jest.fn()}
                    />
                </ReduxProvider>,
            ),
        )
        // The nested group must not be rendered as a filter row: only the one
        // real clause produces a FilterEntry (and therefore one Delete button).
        expect(screen.getAllByRole('button', {name: 'Delete'})).toHaveLength(1)

        userEvent.click(screen.getByRole('button', {name: '+ Add filter'}))

        const newFilterGroup = mockedMutator.changeViewFilter.mock.calls[0][3] as FilterGroup
        const added = newFilterGroup.filters[newFilterGroup.filters.length - 1] as FilterClause
        expect(added.propertyId).toBe(board.cardProperties[1].id)
    })
})

describe('components/viewHeader/filterComponent/nextFilterClause', () => {
    const buildBoard = (properties: Array<[string, PropertyTypeEnum]>): Board => {
        const b = TestBlockFactory.createBoard()
        b.cardProperties = properties.map(([id, type]) => ({id, name: id, type, options: []}))
        return b
    }

    test('returns the first filterable property that is not already used', () => {
        const b = buildBoard([['sel1', 'select'], ['sel2', 'select']])
        const used: FilterClause[] = [{propertyId: 'sel1', condition: 'includes', values: []}]
        const clause = nextFilterClause(b, used)
        expect(clause?.propertyId).toBe('sel2')
        expect(clause?.condition).toBe('includes')
    })

    test('skips property types that cannot be filtered', () => {
        const b = buildBoard([['num', 'number'], ['sel', 'select']])
        expect(nextFilterClause(b, [])?.propertyId).toBe('sel')
    })

    test('chooses a valid default condition for the property type', () => {
        expect(nextFilterClause(buildBoard([['d', 'date']]), [])?.condition).toBe('is')
        expect(nextFilterClause(buildBoard([['c', 'checkbox']]), [])?.condition).toBe('isSet')
        expect(nextFilterClause(buildBoard([['s', 'select']]), [])?.condition).toBe('includes')

        // A text-valued card property (e.g. url) defaults to 'is', not the
        // clause's own 'includes' default, exercising the property branch.
        const urlClause = nextFilterClause(buildBoard([['link', 'url']]), [])
        expect(urlClause?.propertyId).toBe('link')
        expect(urlClause?.condition).toBe('is')
    })

    test('offers the title when the board has no filterable properties', () => {
        expect(nextFilterClause(buildBoard([]), [])?.propertyId).toBe('title')
    })

    test('handles person properties without triggering an assertion failure', () => {
        const assertFailure = jest.spyOn(Utils, 'assertFailure')
        const clause = nextFilterClause(buildBoard([['assignee', 'person']]), [])
        expect(clause?.propertyId).toBe('assignee')
        expect(clause?.condition).toBe('includes')
        expect(assertFailure).not.toHaveBeenCalled()
        assertFailure.mockRestore()
    })

    test('falls back to the title once every property is used', () => {
        const b = buildBoard([['sel', 'select']])
        const used: FilterClause[] = [{propertyId: 'sel', condition: 'includes', values: []}]
        const clause = nextFilterClause(b, used)
        expect(clause?.propertyId).toBe('title')
        expect(clause?.condition).toBe('is')
    })

    test('returns undefined when all properties and the title are in use', () => {
        const b = buildBoard([['sel', 'select']])
        const used: FilterClause[] = [
            {propertyId: 'sel', condition: 'includes', values: []},
            {propertyId: 'title', condition: 'is', values: []},
        ]
        expect(nextFilterClause(b, used)).toBeUndefined()
    })
})
