// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import { jest } from '@jest/globals'

// jsdom does not implement scrollIntoView — stub it out globally.
window.HTMLElement.prototype.scrollIntoView = jest.fn()

// Provide a minimal stub for the Mattermost webapp's global redux store,
// which MarkdownEditor accesses via (window as any).store.getState() for
// channel name mapping, and wraps content in a <Provider store={...}>.
;(window as any).store = {
    getState: () => ({
        entities: {
            channels: {
                channels: {},
                channelsInTeam: {},
                myMembers: {},
                stats: {},
                messageCounts: {},
                membersInChannel: {},
            },
        },
    }),
    subscribe: () => () => {},
    dispatch: () => {},
}

jest.mock('../src/webapp_globals', () =>
    Object.assign({}, jest.requireActual('../src/webapp_globals'), {
        messageHtmlToComponent: jest.fn(() =>
            React.createElement('div', { className: 'mocked-message-html' }, 'Test Comment')
        ),
    })
)
