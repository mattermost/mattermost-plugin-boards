// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {render, screen} from '@testing-library/react'
import '@testing-library/jest-dom'

import {mockStateStore, wrapIntl} from '../testUtils'

import RHSChannelBoardsHeader from './rhsChannelBoardsHeader'

const renderHeader = (channel: {id: string; name: string; display_name: string; type: string}) => {
    const state = {
        language: {
            value: 'en',
        },
        channels: {
            current: channel,
        },
    }
    const store = mockStateStore([], state)
    return render(wrapIntl(
        <ReduxProvider store={store}>
            <RHSChannelBoardsHeader/>
        </ReduxProvider>
    ))
}

describe('components/rhsChannelBoardsHeader', () => {
    it('renders the header', async () => {
        const {container} = renderHeader({
            id: 'channel-id',
            name: 'channel',
            display_name: 'Channel Name',
            type: 'O',
        })
        expect(container).toMatchSnapshot()
    })

    it('shows every member of a group message in the subtitle', async () => {
        const groupMembers = 'admin, boarduser_member_1, boarduser_member_2, boarduser_member_3, boarduser_member_4, boarduser_member_5, boarduser_member_6, boarduser_member_7'
        renderHeader({
            id: 'gm-channel-id',
            name: 'group-message',
            display_name: groupMembers,
            type: 'G',
        })

        const subtitle = screen.getByText(groupMembers)

        // The subtitle renders the full name it is given verbatim.
        expect(subtitle).toHaveTextContent(groupMembers)

        // The full list is also exposed via title so every username stays
        // readable even when the subtitle is visually truncated by the narrow
        // RHS panel. This title attribute is the component's part of the fix.
        expect(subtitle).toHaveAttribute('title', groupMembers)
    })
})
