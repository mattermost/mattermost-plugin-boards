// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {IntlProvider, createIntl} from 'react-intl'
import {Provider as ReduxProvider} from 'react-redux'
import {render, screen} from '@testing-library/react'
import {configureStore} from '@reduxjs/toolkit'

import {reducer as clientConfigReducer} from '../store/clientConfig'

import {getProductName, useLegacyMode, useProductName} from './useProductName'

const intl = createIntl({locale: 'en'})

const makeStore = (legacyMode?: boolean) => {
    return configureStore({
        reducer: {clientConfig: clientConfigReducer},
        preloadedState: {
            clientConfig: {
                value: {
                    telemetry: false,
                    telemetryid: '',
                    enablePublicSharedBoards: false,
                    teammateNameDisplay: 'username',
                    featureFlags: {},
                    maxFileSize: 0,
                    legacyMode: legacyMode as boolean,
                    movedBoardsReadOnly: false,
                },
            },
        },
    })
}

const TestComponent = () => {
    const productName = useProductName()
    const legacyMode = useLegacyMode()
    return (
        <div>
            <span data-testid='name'>{productName}</span>
            <span data-testid='legacy'>{String(legacyMode)}</span>
        </div>
    )
}

const renderWithStore = (legacyMode?: boolean) => {
    return render(
        <ReduxProvider store={makeStore(legacyMode)}>
            <IntlProvider locale='en'>
                <TestComponent/>
            </IntlProvider>
        </ReduxProvider>,
    )
}

describe('hooks/useProductName', () => {
    describe('getProductName', () => {
        test('returns Boards when legacy mode is off', () => {
            expect(getProductName(intl, false)).toBe('Boards')
        })

        test('returns Legacy Boards when legacy mode is on', () => {
            expect(getProductName(intl, true)).toBe('Legacy Boards')
        })

        test('uses translated messages', () => {
            const translated = createIntl({
                locale: 'en',
                messages: {'Product.name': 'Tableros', 'Product.name.legacy': 'Tableros heredados'},
            })
            expect(getProductName(translated, false)).toBe('Tableros')
            expect(getProductName(translated, true)).toBe('Tableros heredados')
        })
    })

    describe('useProductName', () => {
        test('returns Boards when legacy mode is off', () => {
            renderWithStore(false)
            expect(screen.getByTestId('name').textContent).toBe('Boards')
            expect(screen.getByTestId('legacy').textContent).toBe('false')
        })

        test('returns Legacy Boards when legacy mode is on', () => {
            renderWithStore(true)
            expect(screen.getByTestId('name').textContent).toBe('Legacy Boards')
            expect(screen.getByTestId('legacy').textContent).toBe('true')
        })

        test('treats a missing legacyMode value as off', () => {
            renderWithStore(undefined)
            expect(screen.getByTestId('name').textContent).toBe('Boards')
            expect(screen.getByTestId('legacy').textContent).toBe('false')
        })
    })
})
