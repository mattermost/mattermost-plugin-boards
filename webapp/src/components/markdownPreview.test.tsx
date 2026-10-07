// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {render, screen} from '@testing-library/react'
import '@testing-library/jest-dom'
import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'

import {mockStateStore, wrapIntl} from '../testUtils'

import {MarkdownPreview} from './markdownPreview'

describe('components/markdownPreview', () => {
    const state = {
        teams: {
            current: {id: 'team-id-1'},
        },
        users: {
            boardUsers: {},
        },
    }
    const store = mockStateStore([], state)

    const renderPreview = (text: string) => render(wrapIntl(
        <ReduxProvider store={store}>
            <MarkdownPreview text={text}/>
        </ReduxProvider>,
    ))

    describe('inside the Mattermost webapp (window.store present)', () => {
        // tests/setupFile.ts provides window.store and mocks the webapp
        // messageHtmlToComponent renderer with a .mocked-message-html element.
        test('renders markdown through the Mattermost webapp renderer', () => {
            const {container} = renderPreview('Quarterly plan for the *public* board')
            expect(container.querySelector('.mocked-message-html')).toBeInTheDocument()
        })
    })

    describe('standalone / published board (window.store absent)', () => {
        let savedStore: unknown

        beforeEach(() => {
            savedStore = (window as any).store
            delete (window as any).store
        })

        afterEach(() => {
            (window as any).store = savedStore
        })

        test('does not throw when the Mattermost store is unavailable', () => {
            expect(() => renderPreview('Quarterly plan for the *public* board')).not.toThrow()
        })

        test('renders markdown through the Boards-native renderer', () => {
            renderPreview('Quarterly plan for the *public* board')

            // The native renderer turns *public* into an <em> element instead of
            // routing through the Mattermost webapp renderer.
            const emphasized = screen.getByText('public')
            expect(emphasized.tagName.toLowerCase()).toBe('em')

            // The mocked Mattermost renderer must not have been used.
            expect(document.querySelector('.mocked-message-html')).not.toBeInTheDocument()
        })

        test('renders at-mentions and channel links as plain text', () => {
            const {container} = renderPreview('Ping @jdoe in ~town-square')

            // An unauthenticated public viewer has no user/channel data, so these
            // must stay plain text rather than resolving to mention components.
            expect(container.textContent).toContain('@jdoe')
            expect(container.textContent).toContain('~town-square')
            expect(container.querySelector('.mention--highlight')).not.toBeInTheDocument()
            expect(container.querySelector('.mocked-message-html')).not.toBeInTheDocument()
        })

        test('renders links with safe target and rel attributes', () => {
            const {container} = renderPreview('[Docs](https://example.com)')

            const link = container.querySelector('a')
            expect(link).toHaveAttribute('target', '_blank')
            expect(link).toHaveAttribute('rel', 'noreferrer')
            expect(link).toHaveAttribute('href', 'https://example.com')
        })

        test('does not throw when a link href contains a malformed percent escape', () => {
            expect(() => renderPreview('[site](https://example.org/100%)')).not.toThrow()
            expect(() => renderPreview('https://example.org/100%')).not.toThrow()

            const {container} = renderPreview('[site](https://example.org/100%)')
            expect(container.querySelector('a')).toHaveAttribute('href', 'https://example.org/100%25')
        })

        test('escapes raw HTML so markdown cannot inject elements', () => {
            const {container} = renderPreview('<img src=x onerror=alert(1)>')

            // The native renderer escapes '<', so no real <img> element is created.
            expect(container.querySelector('img')).not.toBeInTheDocument()
            expect(container.textContent).toContain('<img src=x onerror=alert(1)>')
        })

        test('does not render javascript: destinations as active links', () => {
            const {container} = renderPreview('[Open](javascript:alert%281%29)')

            expect(container.querySelector('a')).not.toBeInTheDocument()
            expect(container.textContent).toContain('Open')
        })

        test('does not render javascript: destinations encoded as HTML entities as active links', () => {
            const {container} = renderPreview('[Open](jav&#x61;script:alert%281%29)')

            expect(container.querySelector('a')).not.toBeInTheDocument()
            expect(container.textContent).toContain('Open')
        })

        test('does not render nested-entity javascript: destinations as active links', () => {
            const {container} = renderPreview('[Open](&amp;#x6a;avascript:alert(1))')

            expect(container.querySelector('a')).not.toBeInTheDocument()
            expect(container.textContent).toContain('Open')
        })

        test('does not throw when a href numeric entity is an unpaired surrogate', () => {
            expect(() => renderPreview('[site](https://example.org/&#xD800;)')).not.toThrow()
        })

        test('renders empty text without throwing', () => {
            const {container} = renderPreview('')
            expect(container.textContent).toBe('')
            expect(container.querySelector('.mocked-message-html')).not.toBeInTheDocument()
        })
    })
})
