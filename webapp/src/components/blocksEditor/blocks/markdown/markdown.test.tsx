// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {render, act} from '@testing-library/react'

import {mockDOM, wrapDNDIntl, mockStateStore} from '../../../../testUtils'
import {TestBlockFactory} from '../../../../test/testBlockFactory'

import MarkdownBlock from '.'

jest.mock('draft-js/lib/generateRandomKey', () => () => '123')

describe('components/blocksEditor/blocks/markdown', () => {
    beforeEach(mockDOM)

    const board1 = TestBlockFactory.createBoard()
    board1.id = 'board-id-1'

    const state = {
        users: {
            boardUsers: {
                1: {username: 'abc'},
                2: {username: 'd'},
                3: {username: 'e'},
                4: {username: 'f'},
                5: {username: 'g'},
            },
        },
        boards: {
            current: 'board-id-1',
            boards: {
                [board1.id]: board1,
            },
        },
        clientConfig: {
            value: {},
        },
    }
    const store = mockStateStore([], state)

    test('should match Display snapshot', async () => {
        const Component = MarkdownBlock.Display
        const {container} = render(wrapDNDIntl(
            <ReduxProvider store={store}>
                <Component
                    onChange={jest.fn()}
                    value='test-value'
                    onCancel={jest.fn()}
                    onSave={jest.fn()}
                />
            </ReduxProvider>,
        ))
        expect(container).toMatchSnapshot()
    })

    test('should match Input snapshot', async () => {
        let container
        await act(async () => {
            const Component = MarkdownBlock.Input
            const result = render(wrapDNDIntl(
                <ReduxProvider store={store}>
                    <Component
                        onChange={jest.fn()}
                        value='test-value'
                        onCancel={jest.fn()}
                        onSave={jest.fn()}
                    />
                </ReduxProvider>,
            ))
            container = result.container
        })
        expect(container).toMatchSnapshot()
    })

    test('should render markdown with bold text', async () => {
        const Component = MarkdownBlock.Display
        const {container} = render(wrapDNDIntl(
            <ReduxProvider store={store}>
                <Component
                    onChange={jest.fn()}
                    value='This is **bold** text'
                    onCancel={jest.fn()}
                    onSave={jest.fn()}
                />
            </ReduxProvider>,
        ))
        expect(container).toMatchSnapshot()
    })

    test('should render markdown with code block', async () => {
        const Component = MarkdownBlock.Display
        const {container} = render(wrapDNDIntl(
            <ReduxProvider store={store}>
                <Component
                    onChange={jest.fn()}
                    value='```javascript\nconst x = 1;\n```'
                    onCancel={jest.fn()}
                    onSave={jest.fn()}
                />
            </ReduxProvider>,
        ))
        expect(container).toMatchSnapshot()
    })

    test('should render markdown with list', async () => {
        const Component = MarkdownBlock.Display
        const {container} = render(wrapDNDIntl(
            <ReduxProvider store={store}>
                <Component
                    onChange={jest.fn()}
                    value='- Item 1\n- Item 2\n- Item 3'
                    onCancel={jest.fn()}
                    onSave={jest.fn()}
                />
            </ReduxProvider>,
        ))
        expect(container).toMatchSnapshot()
    })

    test('should show placeholder when empty', async () => {
        const Component = MarkdownBlock.Display
        const {container} = render(wrapDNDIntl(
            <ReduxProvider store={store}>
                <Component
                    onChange={jest.fn()}
                    value=''
                    onCancel={jest.fn()}
                    onSave={jest.fn()}
                />
            </ReduxProvider>,
        ))
        expect(container).toMatchSnapshot()
    })
})

