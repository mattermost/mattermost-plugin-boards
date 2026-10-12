// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {render} from '@testing-library/react'

import {mockStateStore, wrapIntl} from '../testUtils'

import RHSChannelBoardsHeader from './rhsChannelBoardsHeader'

describe('components/rhsChannelBoardsHeader', () => {
    it('renders the header', async () => {
        const state = {
            language: {
                value: 'en',
            },
            channels: {
                current: {
                    id: 'channel-id',
                    name: 'channel',
                    display_name: 'Channel Name',
                    type: 'O',
                },
            },
        }
        const store = mockStateStore([], state)
        const {container} = render(wrapIntl(
            <ReduxProvider store={store}>
                <RHSChannelBoardsHeader/>
            </ReduxProvider>
        ))
        expect(container).toMatchSnapshot()
    })

    it('renders the full group message member list in the subtitle', () => {
        const fullGmName = 'gmtester1, gmtester2, gmtester3, gmtester4, gmtester5, gmtester6, gmtester7'
        const state = {
            language: {
                value: 'en',
            },
            channels: {
                current: {
                    id: 'gm-channel-id',
                    name: 'gm-channel',
                    display_name: fullGmName,
                    type: 'G',
                },
            },
        }
        const store = mockStateStore([], state)
        const {container} = render(wrapIntl(
            <ReduxProvider store={store}>
                <RHSChannelBoardsHeader/>
            </ReduxProvider>
        ))

        const subtitle = container.querySelector('.sidebar--right__title__subtitle')
        expect(subtitle?.textContent).toBe(fullGmName)
    })
})
