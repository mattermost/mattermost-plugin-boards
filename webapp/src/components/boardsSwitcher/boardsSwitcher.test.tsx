// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {render} from '@testing-library/react'
import {createMemoryHistory} from 'history'
import {Router} from 'react-router-dom'

import {mockStateStore, wrapIntl} from '../../testUtils'

import BoardsSwitcher from './boardsSwitcher'

describe('components/boardsSwitcher', () => {
    const history = createMemoryHistory()

    const renderSwitcher = (legacyMode?: boolean) => {
        const store = mockStateStore([], {
            users: {
                me: {id: 'user_id_1', props: {}},
                myConfig: {},
            },
            cards: {current: '', cards: {}},
            clientConfig: {value: {legacyMode}},
        })
        return render(wrapIntl(
            <ReduxProvider store={store}>
                <Router history={history}>
                    <BoardsSwitcher
                        onBoardTemplateSelectorOpen={jest.fn()}
                        userIsGuest={true}
                    />
                </Router>
            </ReduxProvider>,
        ))
    }

    test('shows Find Boards by default', () => {
        const {container} = renderSwitcher(false)
        expect(container.querySelector('.BoardsSwitcher')?.textContent).toBe('Find Boards')
    })

    test('shows Find Boards when the config has no legacyMode', () => {
        const {container} = renderSwitcher(undefined)
        expect(container.querySelector('.BoardsSwitcher')?.textContent).toBe('Find Boards')
    })

    test('shows Find Legacy Boards in legacy mode', () => {
        const {container} = renderSwitcher(true)
        expect(container.querySelector('.BoardsSwitcher')?.textContent).toBe('Find Legacy Boards')
    })
})
