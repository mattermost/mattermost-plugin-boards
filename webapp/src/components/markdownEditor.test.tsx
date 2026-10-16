// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {act, fireEvent, render, screen} from '@testing-library/react'
import '@testing-library/jest-dom'
import userEvent from '@testing-library/user-event'
import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {mocked} from 'jest-mock'

import {mockDOM, wrapDNDIntl, mockStateStore} from '../testUtils'

import {TestBlockFactory} from '../test/testBlockFactory'

import {Utils} from '../utils'

import {MarkdownEditor} from './markdownEditor'

jest.mock('../utils')
jest.useFakeTimers()
jest.mock('draft-js/lib/generateRandomKey', () => () => '123')

describe('components/markdownEditor', () => {
    beforeAll(mockDOM)
    beforeEach(jest.clearAllMocks)

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
        teams: {
            current: null,
            currentId: '',
            allTeams: [],
        },
    }
    const store = mockStateStore([], state)
    test('should match snapshot', async () => {
        let container
        await act(async () => {
            const result = render(wrapDNDIntl(
                <ReduxProvider store={store}>
                    <MarkdownEditor
                        id={'test-id'}
                        text={''}
                        placeholderText={'placeholder'}
                        className={'classname-test'}
                        readonly={false}
                        onChange={jest.fn()}
                        onFocus={jest.fn()}
                        onBlur={jest.fn()}
                    />
                </ReduxProvider>,
            ))
            container = result.container
        })
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot with initial text', async () => {
        let container
        await act(async () => {
            const result = render(wrapDNDIntl(
                <ReduxProvider store={store}>

                    <MarkdownEditor
                        id={'test-id'}
                        text={'some initial text already set'}
                        placeholderText={'placeholder'}
                        className={'classname-test'}
                        readonly={false}
                        onChange={jest.fn()}
                        onFocus={jest.fn()}
                        onBlur={jest.fn()}
                    />
                </ReduxProvider>,

            ))
            container = result.container
        })
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot with on click on preview element', async () => {
        let container
        await act(async () => {
            const result = render(wrapDNDIntl(
                <ReduxProvider store={store}>
                    <MarkdownEditor
                        id={'test-id'}
                        text={'some initial text already set'}
                        placeholderText={'placeholder'}
                        className={'classname-test'}
                        readonly={false}
                        onChange={jest.fn()}
                        onFocus={jest.fn()}
                        onBlur={jest.fn()}
                    />
                </ReduxProvider>,

            ))
            container = result.container
        })
        const previewElement = screen.getByTestId('preview-element')
        await act(async () => {
            userEvent.click(previewElement)
        })
        expect(container).toMatchSnapshot()
    })

    test('renders the preview on a published board where the Mattermost store is unavailable', async () => {
        const savedStore = (window as any).store
        delete (window as any).store
        mocked(Utils).htmlFromMarkdown.mockReturnValue('<p>published description</p>')

        try {
            let container: HTMLElement | undefined
            await act(async () => {
                const result = render(wrapDNDIntl(
                    <ReduxProvider store={store}>
                        <MarkdownEditor
                            id={'test-id'}
                            text={'Quarterly plan for the *public* board'}
                            placeholderText={'placeholder'}
                            className={'classname-test'}
                            readonly={true}
                            onChange={jest.fn()}
                            onFocus={jest.fn()}
                            onBlur={jest.fn()}
                        />
                    </ReduxProvider>,
                ))
                container = result.container
            })

            // Pre-fix this wrapped the preview in <Provider store={undefined}>,
            // which throws during render and blanks the published board.
            expect(mocked(Utils).htmlFromMarkdown).toHaveBeenCalledWith('Quarterly plan for the *public* board')
            expect(screen.getByText('published description')).toBeInTheDocument()
            expect(container!.querySelector('.octo-editor-preview')).toBeInTheDocument()
        } finally {
            (window as any).store = savedStore
        }
    })

    test('should match snapshot with on click on preview element and then click out of it', async () => {
        let container: HTMLElement | undefined
        await act(async () => {
            const result = render(wrapDNDIntl(
                <ReduxProvider store={store}>
                    <MarkdownEditor
                        id={'test-id'}
                        text={'some initial text already set'}
                        placeholderText={'placeholder'}
                        className={'classname-test'}
                        readonly={false}
                        onChange={jest.fn()}
                        onFocus={jest.fn()}
                        onBlur={jest.fn()}
                    />
                </ReduxProvider>,

            ))
            container = result.container
        })
        const previewElement = screen.getByTestId('preview-element')
        await act(async () => {
            userEvent.click(previewElement)
            fireEvent.keyDown(container!, {
                key: 'Escape',
                code: 'Escape',
                keyCode: 27,
                charCode: 27,
            })
        })
        expect(container).toMatchSnapshot()
    })
})
