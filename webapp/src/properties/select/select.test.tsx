// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {render, screen} from '@testing-library/react'
import '@testing-library/jest-dom'
import {mocked} from 'jest-mock'

import userEvent from '@testing-library/user-event'

import {IPropertyTemplate, createBoard} from '../../blocks/board'
import {createCard} from '../../blocks/card'

import {wrapIntl} from '../../testUtils'
import mutator from '../../mutator'
import octoClient from '../../octoClient'

import SelectProperty from './property'
import Select from './select'

jest.mock('../../mutator')
jest.mock('../../octoClient')
const mockedMutator = mocked(mutator, true)
const mockedOctoClient = mocked(octoClient, true)

function selectPropertyTemplate(): IPropertyTemplate {
    return {
        id: 'select-template',
        name: 'select',
        type: 'select',
        options: [
            {
                id: 'option-1',
                value: 'one',
                color: 'propColorDefault',
            },
            {
                id: 'option-2',
                value: 'two',
                color: 'propColorGreen',
            },
            {
                id: 'option-3',
                value: 'three',
                color: 'propColorRed',
            },
        ],
    }
}

describe('properties/select', () => {
    const nonEditableSelectTestId = 'select-non-editable'

    const clearButton = () => screen.queryByRole('button', {name: /clear/i})
    const board = createBoard()
    const card = createCard()

    beforeEach(() => {
        // Mock octoClient.getStatusTransitionRules to return empty array (no rules)
        mockedOctoClient.getStatusTransitionRules.mockResolvedValue([])
    })

    it('shows the selected option', () => {
        const propertyTemplate = selectPropertyTemplate()
        const option = propertyTemplate.options[0]

        const {container} = render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
                propertyValue={option.id}
                readOnly={true}
                showEmptyPlaceholder={false}
            />,
        ))

        expect(screen.getByText(option.value)).toBeInTheDocument()
        expect(clearButton()).not.toBeInTheDocument()

        expect(container).toMatchSnapshot()
    })

    it('shows empty placeholder', () => {
        const propertyTemplate = selectPropertyTemplate()
        const emptyValue = 'Empty'

        const {container} = render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                showEmptyPlaceholder={true}
                propertyTemplate={propertyTemplate}
                propertyValue={''}
                readOnly={true}
            />,
        ))

        expect(screen.getByText(emptyValue)).toBeInTheDocument()
        expect(clearButton()).not.toBeInTheDocument()

        expect(container).toMatchSnapshot()
    })

    it('shows the menu with options when preview is clicked', () => {
        const propertyTemplate = selectPropertyTemplate()
        const selected = propertyTemplate.options[1]

        render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
                propertyValue={selected.id}
                showEmptyPlaceholder={false}
                readOnly={false}
            />,
        ))

        userEvent.click(screen.getByTestId(nonEditableSelectTestId))

        // check that all options are visible
        for (const option of propertyTemplate.options) {
            const elements = screen.getAllByText(option.value)

            // selected option is rendered twice: in the input and inside the menu
            const expected = option.id === selected.id ? 2 : 1
            expect(elements.length).toBe(expected)
        }

        expect(clearButton()).toBeInTheDocument()
    })

    it('can select the option from menu', () => {
        const propertyTemplate = selectPropertyTemplate()
        const optionToSelect = propertyTemplate.options[2]

        render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
                propertyValue={''}
                showEmptyPlaceholder={false}
                readOnly={false}
            />,
        ))

        userEvent.click(screen.getByTestId(nonEditableSelectTestId))
        userEvent.click(screen.getByText(optionToSelect.value))

        expect(clearButton()).not.toBeInTheDocument()
        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, optionToSelect.id)
    })

    it('can clear the selected option', () => {
        const propertyTemplate = selectPropertyTemplate()
        const selected = propertyTemplate.options[1]

        render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
                propertyValue={selected.id}
                showEmptyPlaceholder={false}
                readOnly={false}
            />,
        ))

        userEvent.click(screen.getByTestId(nonEditableSelectTestId))

        const clear = clearButton()
        expect(clear).toBeInTheDocument()
        userEvent.click(clear!)

        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, '')
    })

    it('can create new option', () => {
        const propertyTemplate = selectPropertyTemplate()
        const initialOption = propertyTemplate.options[0]
        const newOption = 'new-option'

        render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
                propertyValue={initialOption.id}
                showEmptyPlaceholder={false}
                readOnly={false}
            />,
        ))

        mockedMutator.insertPropertyOption.mockResolvedValue()

        userEvent.click(screen.getByTestId(nonEditableSelectTestId))
        userEvent.type(screen.getByRole('combobox', {name: /value selector/i}), `${newOption}{enter}`)

        expect(mockedMutator.insertPropertyOption).toHaveBeenCalledWith(board.id, board.cardProperties, propertyTemplate, expect.objectContaining({value: newOption}), 'add property option')
        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, 'option-3')
    })

    it('filters Status options based on transition rules', async () => {
        const propertyTemplate = selectPropertyTemplate()
        // Set the property name to 'Status' to trigger transition rule filtering
        propertyTemplate.name = 'Status'
        const currentOption = propertyTemplate.options[0] // option-1

        // Mock transition rules that disallow transition from option-1 to option-2
        const mockRules = [
            {
                id: 'rule-1',
                boardId: board.id,
                fromStatus: 'option-1',
                toStatus: 'option-2',
                allowed: false,
                createAt: Date.now(),
                updateAt: Date.now(),
            },
        ]
        mockedOctoClient.getStatusTransitionRules.mockResolvedValue(mockRules)

        render(wrapIntl(
            <Select
                property={new SelectProperty()}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
                propertyValue={currentOption.id}
                showEmptyPlaceholder={false}
                readOnly={false}
            />,
        ))

        // Click to open the dropdown
        userEvent.click(screen.getByTestId(nonEditableSelectTestId))

        // Wait for the async transition rules to load
        await screen.findByRole('combobox', {name: /value selector/i})

        // option-1 (current) should be visible (always allowed to keep current status)
        // It appears twice: once in the selector and once in the menu
        const oneElements = screen.getAllByText('one')
        expect(oneElements.length).toBeGreaterThan(0)

        // option-2 should NOT be visible (transition disallowed by rule)
        expect(screen.queryByText('two')).not.toBeInTheDocument()

        // option-3 should be visible (no rule means allowed by default)
        expect(screen.getByText('three')).toBeInTheDocument()
    })
})
