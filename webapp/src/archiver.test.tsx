// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {render, act, screen} from '@testing-library/react'
import '@testing-library/jest-dom'
import {createIntl} from 'react-intl'

import {wrapIntl} from './testUtils'
import {FlashMessages} from './components/flashMessages'
import mutator from './mutator'
import {Archiver} from './archiver'

jest.mock('./mutator')
const mockedMutator = mutator as jest.Mocked<typeof mutator>

const intl = createIntl({locale: 'en'})

// Archiver.importFullArchive creates a hidden file <input> and calls .click() to
// open the browser's native file picker. We capture that input and stub the
// native click so the test can drive the real onchange import flow instead.
function triggerImport(): HTMLInputElement {
    const createElement = document.createElement.bind(document)
    let fileInput: HTMLInputElement | null = null

    const spy = jest.spyOn(document, 'createElement').mockImplementation((tagName: string, options?: ElementCreationOptions) => {
        const element = createElement(tagName, options)
        if (tagName === 'input') {
            fileInput = element as HTMLInputElement
            jest.spyOn(fileInput, 'click').mockImplementation(() => {})
        }
        return element
    })

    Archiver.importFullArchive(intl)
    spy.mockRestore()

    if (!fileInput) {
        throw new Error('file input was not created')
    }
    return fileInput
}

function selectFile(input: HTMLInputElement, file: File): Promise<void> {
    Object.defineProperty(input, 'files', {value: [file], configurable: true})
    return (input.onchange as (event: Event) => Promise<void>)(new Event('change'))
}

describe('archiver import feedback', () => {
    beforeEach(() => {
        jest.useFakeTimers()
        jest.clearAllMocks()
    })

    afterEach(() => {
        jest.clearAllTimers()
        jest.useRealTimers()
    })

    test('shows a loading indicator while importing, then a success message', async () => {
        let resolveImport: (response: Response) => void = () => {}
        mockedMutator.importFullArchive.mockReturnValue(new Promise<Response>((resolve) => {
            resolveImport = resolve
        }))

        render(wrapIntl(<FlashMessages milliseconds={2000}/>))

        const input = triggerImport()
        const file = new File(['data'], 'valid.boardarchive')

        let importComplete: Promise<void> = Promise.resolve()
        act(() => {
            importComplete = selectFile(input, file)
        })

        expect(mockedMutator.importFullArchive).toHaveBeenCalledWith(file)
        expect(screen.getByText('Importing archive…')).toBeInTheDocument()

        await act(async () => {
            resolveImport({status: 200} as Response)
            await importComplete
        })

        const success = screen.getByText('Archive imported')
        expect(success).toBeInTheDocument()
        expect(success).toHaveClass('normal')
        expect(screen.queryByText('Importing archive…')).not.toBeInTheDocument()
    })

    test('surfaces the server error message when the import is rejected', async () => {
        mockedMutator.importFullArchive.mockResolvedValue({
            status: 400,
            text: () => Promise.resolve(JSON.stringify({error: 'archive is missing required fields'})),
        } as Response)

        render(wrapIntl(<FlashMessages milliseconds={2000}/>))

        const input = triggerImport()

        await act(async () => {
            await selectFile(input, new File(['{}'], 'broken.boardarchive'))
        })

        const flash = screen.getByText('Import failed: archive is missing required fields')
        expect(flash).toBeInTheDocument()
        expect(flash).toHaveClass('high')
    })

    test('surfaces a non-JSON error body verbatim', async () => {
        mockedMutator.importFullArchive.mockResolvedValue({
            status: 500,
            text: () => Promise.resolve('unexpected server failure'),
        } as Response)

        render(wrapIntl(<FlashMessages milliseconds={2000}/>))

        const input = triggerImport()

        await act(async () => {
            await selectFile(input, new File(['{}'], 'broken.boardarchive'))
        })

        const flash = screen.getByText('Import failed: unexpected server failure')
        expect(flash).toBeInTheDocument()
        expect(flash).toHaveClass('high')
    })

    test('falls back to a generic failure message when the error body is empty', async () => {
        mockedMutator.importFullArchive.mockResolvedValue({
            status: 500,
            text: () => Promise.resolve(''),
        } as Response)

        render(wrapIntl(<FlashMessages milliseconds={2000}/>))

        const input = triggerImport()

        await act(async () => {
            await selectFile(input, new File(['{}'], 'broken.boardarchive'))
        })

        const flash = screen.getByText('Import failed')
        expect(flash).toBeInTheDocument()
        expect(flash).toHaveClass('high')
    })

    test('shows a generic failure message when the request throws', async () => {
        mockedMutator.importFullArchive.mockRejectedValue(new Error('network down'))

        render(wrapIntl(<FlashMessages milliseconds={2000}/>))

        const input = triggerImport()

        await act(async () => {
            await selectFile(input, new File(['{}'], 'broken.boardarchive'))
        })

        expect(screen.getByText('Import failed')).toHaveClass('high')
    })
})
