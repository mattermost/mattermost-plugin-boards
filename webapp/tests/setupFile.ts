// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import { jest } from '@jest/globals'

jest.mock('../src/webapp_globals', () =>
    Object.assign({}, jest.requireActual('../src/webapp_globals'), {
        messageHtmlToComponent: jest.fn(() =>
            React.createElement('div', { className: 'mocked-message-html' }, 'Test Comment')
        ),
    })
)
