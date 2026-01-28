// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {render, screen, fireEvent} from '@testing-library/react'
import {mocked} from 'jest-mock'

import octoClient from '../../../../octoClient'

import VideoBlock from '.'

jest.mock('../../../../octoClient')
jest.mock('../../../rootPortal', () => ({
    __esModule: true,
    default: ({children}: {children: React.ReactNode}) => <div data-testid='root-portal'>{children}</div>,
}))

describe('components/blocksEditor/blocks/video', () => {
    test('should match Display snapshot for file upload', async () => {
        const mockedOcto = mocked(octoClient, true)
        mockedOcto.getFileAsDataUrl.mockResolvedValue({url: 'test.jpg'})
        const Component = VideoBlock.Display
        const {container} = render(
            <Component
                onChange={jest.fn()}
                value={{file: 'test', filename: 'test-filename', sourceType: 'file'}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
            />,
        )
        await screen.findByTestId('video')
        expect(container).toMatchSnapshot()
    })

    test('should match Display snapshot for file upload with fileId', async () => {
        const mockedOcto = mocked(octoClient, true)
        mockedOcto.getFileAsDataUrl.mockResolvedValue({url: 'test.mp4'})
        const Component = VideoBlock.Display
        const {container} = render(
            <Component
                onChange={jest.fn()}
                value={{fileId: 'test-file-id', filename: 'test-video.mp4', sourceType: 'file'}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
            />,
        )
        await screen.findByTestId('video')
        expect(container).toMatchSnapshot()
        expect(mockedOcto.getFileAsDataUrl).toHaveBeenCalledWith('', 'test-file-id')
    })

    test('should match Display snapshot for YouTube', async () => {
        const Component = VideoBlock.Display
        const {container} = render(
            <Component
                onChange={jest.fn()}
                value={{sourceType: 'youtube', videoId: 'dQw4w9WgXcQ', videoUrl: 'https://youtube.com/watch?v=dQw4w9WgXcQ'}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
            />,
        )
        await screen.findByTestId('video-thumbnail')
        expect(container).toMatchSnapshot()
    })

    test('should match Display snapshot with empty value', async () => {
        const Component = VideoBlock.Display
        const {container} = render(
            <Component
                onChange={jest.fn()}
                value={{file: '', filename: ''}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
                currentBoardId=''
            />,
        )
        expect(container).toMatchSnapshot()
    })

    test('should match Input snapshot', async () => {
        const Component = VideoBlock.Input
        const {container} = render(
            <Component
                onChange={jest.fn()}
                value={{file: 'test', filename: 'test-filename'}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
            />,
        )
        expect(container).toMatchSnapshot()
    })

    test('should match Input snapshot with empty input', async () => {
        const Component = VideoBlock.Input
        const {container} = render(
            <Component
                onChange={jest.fn()}
                value={{file: '', filename: ''}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
            />,
        )
        expect(container).toMatchSnapshot()
    })

    test('should handle URL input and submission', async () => {
        const onSave = jest.fn()
        const Component = VideoBlock.Input
        render(
            <Component
                onChange={jest.fn()}
                value={{}}
                onCancel={jest.fn()}
                onSave={onSave}
            />,
        )

        const input = screen.getByPlaceholderText('Paste YouTube or Google Drive URL...')
        fireEvent.change(input, {target: {value: 'https://youtube.com/watch?v=dQw4w9WgXcQ'}})

        const addButton = screen.getByText('Add')
        fireEvent.click(addButton)

        expect(onSave).toBeCalledWith({
            sourceType: 'youtube',
            videoId: 'dQw4w9WgXcQ',
            videoUrl: 'https://youtube.com/watch?v=dQw4w9WgXcQ',
        })
    })

    test('should switch to file upload mode', async () => {
        const Component = VideoBlock.Input
        render(
            <Component
                onChange={jest.fn()}
                value={{}}
                onCancel={jest.fn()}
                onSave={jest.fn()}
            />,
        )

        const uploadButton = screen.getByText('Upload File')
        fireEvent.click(uploadButton)

        // After clicking, the file input should be shown
        const fileInput = screen.getByTestId('video-input')
        expect(fileInput).toBeTruthy()
    })
})
