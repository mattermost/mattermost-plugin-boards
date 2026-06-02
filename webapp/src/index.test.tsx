// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {render} from '@testing-library/react'
import '@testing-library/jest-dom'

import Plugin from './index'

jest.mock('mattermost-redux/actions/teams', () => ({
    selectTeam: jest.fn((teamId: string) => ({type: 'SELECT_TEAM', data: teamId})),
}))
jest.mock('./app', () => ({__esModule: true, default: () => null}))
jest.mock('./components/withWebSockets', () => ({
    __esModule: true,
    default: (props: {children: React.ReactNode}) => props.children,
}))
jest.mock('./store', () => ({
    __esModule: true,
    default: {
        getState: jest.fn(() => ({
            teams: {
                currentId: 'team-id',
            },
        })),
        subscribe: jest.fn(() => jest.fn()),
        dispatch: jest.fn(),
    },
}))
jest.mock('./theme', () => ({
    setMattermostTheme: jest.fn(),
}))

const fakeMattermostState = {
    entities: {
        general: {
            config: {
                SiteURL: '',
            },
        },
        preferences: {
            myPreferences: {
                'display_settings--name_format': {value: 'username'},
                theme: '{}',
            },
        },
        channels: {
            currentChannelId: 'channel-id',
            channels: {
                'channel-id': {id: 'channel-id'},
            },
        },
        teams: {
            currentTeamId: 'team-id',
        },
        users: {
            currentUserId: 'user-id',
        },
    },
}

const fakeMattermostStore = {
    getState: jest.fn(() => fakeMattermostState),
    subscribe: jest.fn(() => jest.fn()),
    dispatch: jest.fn(),
}

let CapturedMainApp: React.ComponentType<{webSocketClient: Record<string, never>}> | undefined

const fakeRegistry = {
    registerWebSocketEventHandler: jest.fn(),
    registerPostTypeComponent: jest.fn(),
    registerRightHandSidebarComponent: jest.fn(() => ({
        rhsId: 'rhs-id',
        toggleRHSPlugin: jest.fn(),
    })),
    registerChannelHeaderButtonAction: jest.fn(() => 'channel-header-button-id'),
    registerProduct: jest.fn((...args: unknown[]) => {
        CapturedMainApp = args[4] as React.ComponentType<{webSocketClient: Record<string, never>}>
    }),
    registerPostWillRenderEmbedComponent: jest.fn(),
    registerRootComponent: jest.fn(() => 'root-component-id'),
}

beforeEach(() => {
    document.body.className = ''
    document.body.innerHTML = '<div id="root"></div>'
    CapturedMainApp = undefined
    jest.clearAllMocks()
})

test('keeps app body class sticky while cleaning up Boards classes', async () => {
    const plugin = new Plugin()

    await plugin.initialize(
        fakeRegistry as unknown as Parameters<Plugin['initialize']>[0],
        fakeMattermostStore as unknown as Parameters<Plugin['initialize']>[1],
    )

    expect(document.body).not.toHaveClass('app__body')
    expect(document.body).not.toHaveClass('focalboard-body')
    expect(document.getElementById('root')).not.toHaveClass('focalboard-plugin-root')

    expect(CapturedMainApp).toBeDefined()
    const MainApp = CapturedMainApp as React.ComponentType<{webSocketClient: Record<string, never>}>
    const {unmount} = render(<MainApp webSocketClient={{}} />)

    expect(document.body).toHaveClass('app__body')
    expect(document.body).toHaveClass('focalboard-body')
    expect(document.getElementById('root')).toHaveClass('focalboard-plugin-root')

    unmount()

    expect(document.body).toHaveClass('app__body')
    expect(document.body).not.toHaveClass('focalboard-body')
    expect(document.getElementById('root')).not.toHaveClass('focalboard-plugin-root')
})
