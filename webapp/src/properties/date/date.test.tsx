// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {render} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {IntlProvider} from 'react-intl'
import {mocked} from 'jest-mock'

import '@testing-library/jest-dom'

import {wrapIntl} from '../../testUtils'
import {IPropertyTemplate, createBoard} from '../../blocks/board'
import {createCard} from '../../blocks/card'
import mutator from '../../mutator'

import DateProperty from './property'
import DateProp from './date'

jest.mock('../../mutator')
const mockedMutator = mocked(mutator, true)

// create Dates for specific days for this year.
const June15 = new Date(Date.UTC(new Date().getFullYear(), 5, 15, 12))
const June15Local = new Date(new Date().getFullYear(), 5, 15, 12)
const June20 = new Date(Date.UTC(new Date().getFullYear(), 5, 20, 12))

describe('properties/dateRange', () => {
    const card = createCard()
    const board = createBoard()
    const propertyTemplate: IPropertyTemplate = {
        id: 'test',
        name: 'test',
        type: 'date',
        options: [],
    }

    beforeEach(() => {
        // Quick fix to disregard console error when unmounting a component
        console.error = jest.fn()
        document.execCommand = jest.fn()
        jest.resetAllMocks()
    })

    test('returns default correctly', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue=''
                showEmptyPlaceholder={false}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('returns local correctly - es local', () => {
        const component = (
            <IntlProvider locale='es'>
                <DateProp
                    property={new DateProperty()}
                    propertyValue={June15Local.getTime().toString()}
                    showEmptyPlaceholder={false}
                    readOnly={false}
                    board={{...board}}
                    card={{...card}}
                    propertyTemplate={propertyTemplate}
                />
            </IntlProvider>
        )

        const {container, getByText} = render(component)
        const input = getByText('15 de junio')
        expect(input).not.toBeNull()
        expect(container).toMatchSnapshot()
    })

    test('handles calendar click event', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue=''
                showEmptyPlaceholder={true}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        const date = new Date()
        const fifteenth = Date.UTC(date.getFullYear(), date.getMonth(), 15, 12)

        const {getByText, getByTitle} = render(component)
        const dayDisplay = getByText('Empty')
        userEvent.click(dayDisplay)

        const day = getByText('15')
        const modal = getByTitle('Close').children[0]
        userEvent.click(day)
        userEvent.click(modal)

        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, JSON.stringify({from: fifteenth}))
    })

    test('handles setting range', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue={''}
                showEmptyPlaceholder={true}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        // open modal
        const {getByText, getByTitle} = render(component)
        const dayDisplay = getByText('Empty')
        userEvent.click(dayDisplay)

        // select start date
        const date = new Date()
        const fifteenth = Date.UTC(date.getFullYear(), date.getMonth(), 15, 12)
        const start = getByText('15')
        userEvent.click(start)

        // create range
        const endDate = getByText('End date')
        userEvent.click(endDate)

        const twentieth = Date.UTC(date.getFullYear(), date.getMonth(), 20, 12)

        const end = getByText('20')
        const modal = getByTitle('Close').children[0]
        userEvent.click(end)
        userEvent.click(modal)

        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, JSON.stringify({from: fifteenth, to: twentieth}))
    })

    test('handle clear', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue={June15Local.getTime().toString()}
                showEmptyPlaceholder={false}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        const {container, getByText, getByTitle} = render(component)
        expect(container).toMatchSnapshot()

        // open modal
        const dayDisplay = getByText('June 15')
        userEvent.click(dayDisplay)

        const clear = getByText('Clear')
        const modal = getByTitle('Close').children[0]
        userEvent.click(clear)
        userEvent.click(modal)

        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, '')
    })

    test('set via text input', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue={'{"from": ' + June15.getTime().toString() + ',"to": ' + June20.getTime().toString() + '}'}
                showEmptyPlaceholder={false}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        const {container, getByRole, getByTitle, getByDisplayValue} = render(component)
        expect(container).toMatchSnapshot()

        // open modal
        const dayDisplay = getByRole('button', {name: 'June 15 → June 20'})

        userEvent.click(dayDisplay)

        const fromInput = getByDisplayValue('June 15')
        const toInput = getByDisplayValue('June 20')

        userEvent.type(fromInput, '{selectall}{delay}07/15/2021{enter}')
        userEvent.type(toInput, '{selectall}{delay}07/20/2021{enter}')

        const July15 = new Date(Date.UTC(2021, 6, 15, 12))
        const July20 = new Date(Date.UTC(2021, 6, 20, 12))
        const modal = getByTitle('Close').children[0]

        userEvent.click(modal)

        // {from: '2021-07-15', to: '2021-07-20'}
        const retVal = {from: July15.getTime(), to: July20.getTime()}
        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, JSON.stringify(retVal))
    })

    test('set via text input, es locale', () => {
        const component = (
            <IntlProvider locale='es'>
                <DateProp
                    property={new DateProperty()}
                    propertyValue={'{"from": ' + June15.getTime().toString() + ',"to": ' + June20.getTime().toString() + '}'}
                    showEmptyPlaceholder={false}
                    readOnly={false}
                    board={{...board}}
                    card={{...card}}
                    propertyTemplate={propertyTemplate}
                />
            </IntlProvider>
        )
        const {container, getByRole, getByTitle, getByDisplayValue} = render(component)
        expect(container).toMatchSnapshot()

        // open modal
        const dayDisplay = getByRole('button', {name: '15 de junio → 20 de junio'})

        userEvent.click(dayDisplay)

        const fromInput = getByDisplayValue('15 de junio')
        const toInput = getByDisplayValue('20 de junio')

        userEvent.type(fromInput, '{selectall}15/07/2021{enter}')
        userEvent.type(toInput, '{selectall}20/07/2021{enter}')

        const July15 = new Date(Date.UTC(2021, 6, 15, 12))
        const July20 = new Date(Date.UTC(2021, 6, 20, 12))
        const modal = getByTitle('Close').children[0]

        userEvent.click(modal)

        // {from: '2021-07-15', to: '2021-07-20'}
        const retVal = {from: July15.getTime(), to: July20.getTime()}
        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, JSON.stringify(retVal))
    })

    test('cancel set via text input', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue={'{"from": ' + June15.getTime().toString() + ',"to": ' + June20.getTime().toString() + '}'}
                showEmptyPlaceholder={false}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        const {container, getByRole, getByTitle, getByDisplayValue} = render(component)
        expect(container).toMatchSnapshot()

        // open modal
        const dayDisplay = getByRole('button', {name: 'June 15 → June 20'})
        userEvent.click(dayDisplay)

        const fromInput = getByDisplayValue('June 15')
        const toInput = getByDisplayValue('June 20')
        userEvent.type(fromInput, '{selectall}07/15/2021{delay}{esc}')
        userEvent.type(toInput, '{selectall}07/20/2021{delay}{esc}')

        const modal = getByTitle('Close').children[0]
        userEvent.click(modal)

        // const retVal = {from: '2021-06-15', to: '2021-06-20'}
        const retVal = {from: June15.getTime(), to: June20.getTime()}
        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, JSON.stringify(retVal))
    })

    test('handles `Today` button click event', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue={''}
                showEmptyPlaceholder={true}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        // To see if 'Today' button correctly selects today's date,
        // we can check it against `new Date()`.
        // About `Date()`
        // > "When called as a function, returns a string representation of the current date and time"
        const date = new Date()
        const today = Date.UTC(date.getFullYear(), date.getMonth(), date.getDate(), 12)

        const {getByText, getByTitle} = render(component)
        const dayDisplay = getByText('Empty')
        userEvent.click(dayDisplay)

        const day = getByText('Today')
        const modal = getByTitle('Close').children[0]
        userEvent.click(day)
        userEvent.click(modal)

        expect(mockedMutator.changePropertyValue).toHaveBeenCalledWith(board.id, card, propertyTemplate.id, JSON.stringify({from: today}))
    })

    test('returns component with new date after prop change', () => {
        const component = wrapIntl(
            <DateProp
                property={new DateProperty()}
                propertyValue=''
                showEmptyPlaceholder={false}
                readOnly={false}
                board={{...board}}
                card={{...card}}
                propertyTemplate={propertyTemplate}
            />,
        )

        const {container, rerender} = render(component)

        rerender(
            wrapIntl(
                <DateProp
                    property={new DateProperty()}
                    propertyValue={'{"from": ' + June15.getTime().toString() + '}'}
                    showEmptyPlaceholder={false}
                    readOnly={false}
                    board={{...board}}
                    card={{...card}}
                    propertyTemplate={propertyTemplate}
                />,
            ),
        )

        expect(container).toMatchSnapshot()
    })
})
