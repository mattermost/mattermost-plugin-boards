// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {render, screen} from '@testing-library/react'
import {Provider as ReduxProvider} from 'react-redux'

import '@testing-library/jest-dom'
import userEvent from '@testing-library/user-event'

import {mocked} from 'jest-mock'

import {FilterClause} from '../../blocks/filterClause'
import {IPropertyTemplate} from '../../blocks/board'

import {TestBlockFactory} from '../../test/testBlockFactory'

import {wrapIntl, mockStateStore} from '../../testUtils'

import mutator from '../../mutator'
import propsRegistry from '../../properties'

import FilterValue from './filterValue'

jest.mock('../../mutator')
const mockedMutator = mocked(mutator)

const board = TestBlockFactory.createBoard()
const activeView = TestBlockFactory.createBoardView(board)
const state = {
    users: {
        me: {
            id: 'user-id-1',
            username: 'username_1',
        },
    },
}
const store = mockStateStore([], state)
const filter: FilterClause = {
    propertyId: '1',
    condition: 'includes',
    values: ['Status'],
}

describe('components/viewHeader/filterValue', () => {
    beforeEach(() => {
        jest.clearAllMocks()
        board.cardProperties[0].options = [{id: 'Status', value: 'Status', color: ''}]
        activeView.fields.filter.filters = [filter]
    })
    test('return filterValue', () => {
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={activeView}
                        filter={filter}
                        template={board.cardProperties[0]}
                        propertyType={propsRegistry.get(board.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getByRole('button', {name: 'menuwrapper'})
        userEvent.click(buttonElement)
        expect(container).toMatchSnapshot()
    })
    test('return filterValue and click Status', () => {
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={activeView}
                        filter={filter}
                        template={board.cardProperties[0]}
                        propertyType={propsRegistry.get(board.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getByRole('button', {name: 'menuwrapper'})
        userEvent.click(buttonElement)
        const switchStatus = screen.getAllByText('Status')[1]
        userEvent.click(switchStatus)
        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
        expect(container).toMatchSnapshot()
    })
    test('return filterValue and click Status with Status not in filter', () => {
        filter.values = ['test']
        activeView.fields.filter.filters = [filter]
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={activeView}
                        filter={filter}
                        template={board.cardProperties[0]}
                        propertyType={propsRegistry.get(board.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getByRole('button', {name: 'menuwrapper'})
        userEvent.click(buttonElement)
        const switchStatus = screen.getAllByText('Status')[0]
        userEvent.click(switchStatus)
        expect(mockedMutator.changeViewFilter).toHaveBeenCalledTimes(1)
        expect(container).toMatchSnapshot()
    })
    test('return filterValue and verify that menu is not closed after clicking on the item', () => {
        filter.values = []
        activeView.fields.filter.filters = [filter]
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={activeView}
                        filter={filter}
                        template={board.cardProperties[0]}
                        propertyType={propsRegistry.get(board.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getByRole('button', {name: '(empty)'})
        userEvent.click(buttonElement)

        const switchStatus = screen.getByRole('button', {name: 'Status'})
        userEvent.click(switchStatus)
        expect(switchStatus).toBeInTheDocument()
    })

    test('value button exposes the full comma-separated option list as its title tooltip', () => {
        const multiBoard = TestBlockFactory.createBoard()
        const multiView = TestBlockFactory.createBoardView(multiBoard)
        multiBoard.cardProperties[0].options = [
            {id: 'opt1', value: 'Not Started', color: ''},
            {id: 'opt2', value: 'In Progress', color: ''},
            {id: 'opt3', value: 'In Review', color: ''},
            {id: 'opt4', value: 'Completed', color: ''},
        ]
        const multiFilter: FilterClause = {
            propertyId: multiBoard.cardProperties[0].id,
            condition: 'includes',
            values: ['opt1', 'opt2', 'opt3', 'opt4'],
        }
        multiView.fields.filter.filters = [multiFilter]
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={multiView}
                        filter={multiFilter}
                        template={multiBoard.cardProperties[0]}
                        propertyType={propsRegistry.get(multiBoard.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const expected = 'Not Started, In Progress, In Review, Completed'
        const buttonElement = screen.getByRole('button', {name: expected})
        expect(buttonElement).toHaveAttribute('title', expected)
        expect(buttonElement).toHaveTextContent(expected)
    })

    test('value button title falls back to (empty) when no options are selected', () => {
        const emptyBoard = TestBlockFactory.createBoard()
        const emptyView = TestBlockFactory.createBoardView(emptyBoard)
        emptyBoard.cardProperties[0].options = [{id: 'opt1', value: 'Not Started', color: ''}]
        const emptyFilter: FilterClause = {
            propertyId: emptyBoard.cardProperties[0].id,
            condition: 'includes',
            values: [],
        }
        emptyView.fields.filter.filters = [emptyFilter]
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={emptyView}
                        filter={emptyFilter}
                        template={emptyBoard.cardProperties[0]}
                        propertyType={propsRegistry.get(emptyBoard.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getByRole('button', {name: '(empty)'})
        expect(buttonElement).toHaveAttribute('title', '(empty)')
    })

    test('value button title falls back to (Unknown) for a selected option missing from the template', () => {
        const unknownBoard = TestBlockFactory.createBoard()
        const unknownView = TestBlockFactory.createBoardView(unknownBoard)
        unknownBoard.cardProperties[0].options = [{id: 'opt1', value: 'Not Started', color: ''}]
        const unknownFilter: FilterClause = {
            propertyId: unknownBoard.cardProperties[0].id,
            condition: 'includes',
            values: ['deleted-option-id'],
        }
        unknownView.fields.filter.filters = [unknownFilter]
        render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={unknownView}
                        filter={unknownFilter}
                        template={unknownBoard.cardProperties[0]}
                        propertyType={propsRegistry.get(unknownBoard.cardProperties[0].type)}
                    />
                </ReduxProvider>,
            ),
        )
        const buttonElement = screen.getByRole('button', {name: '(Unknown)'})
        expect(buttonElement).toHaveAttribute('title', '(Unknown)')
    })

    test('return date filter value', () => {
        const propertyTemplate: IPropertyTemplate = {
            id: 'datePropertyID',
            name: 'My Date Property',
            type: 'date',
            options: [],
        }
        board.cardProperties.push(propertyTemplate)

        const dateFilter: FilterClause = {
            propertyId: 'datePropertyID',
            condition: 'is',
            values: [],
        }

        // filter.values = []
        activeView.fields.filter.filters = [dateFilter]
        const {container} = render(
            wrapIntl(
                <ReduxProvider store={store}>
                    <FilterValue
                        view={activeView}
                        filter={filter}
                        template={propertyTemplate}
                        propertyType={propsRegistry.get(propertyTemplate.type)}
                    />
                </ReduxProvider>,
            ),
        )
        expect(container).toMatchSnapshot()

        const buttonElement = screen.getByRole('button', {name: 'Empty'})
        userEvent.click(buttonElement)

        // make sure modal is displayed
        const clearButton = screen.getByRole('button', {name: 'Clear'})
        expect(clearButton).toBeInTheDocument()
    })
})
