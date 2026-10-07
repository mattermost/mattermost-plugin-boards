// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {createMemoryHistory, MemoryHistory} from 'history'
import {render, act, screen} from '@testing-library/react'
import {Provider as ReduxProvider} from 'react-redux'
import {configureStore} from '@reduxjs/toolkit'

import {wrapIntl} from './testUtils'

import {reducer as usersReducer, setMe} from './store/users'
import {reducer as clientConfigReducer, setClientConfig} from './store/clientConfig'
import {reducer as globalErrorReducer} from './store/globalError'
import {IUser} from './user'
import {ClientConfig} from './config/clientConfig'

import {SuiteWindow} from './types/index'

// The board/welcome/access-denied pages pull in heavy, websocket-driven trees
// that are irrelevant to routing behaviour, so stub them out. ErrorPage is kept
// real because the tests assert on the error message it renders.
jest.mock('./pages/boardPage/boardPage', () => ({
    __esModule: true,
    default: () => <div>MOCK_BOARD_PAGE</div>,
}))
jest.mock('./pages/welcome/welcomePage', () => ({
    __esModule: true,
    default: () => <div>MOCK_WELCOME_PAGE</div>,
}))
jest.mock('./pages/accessDeniedPage', () => ({
    __esModule: true,
    default: () => <div>MOCK_ACCESS_DENIED_PAGE</div>,
}))

import FocalboardRouter from './router'

const windowAny = (window as SuiteWindow)

const makeStore = () => {
    const store = configureStore({
        reducer: {
            users: usersReducer,
            clientConfig: clientConfigReducer,
            globalError: globalErrorReducer,
        },
    })

    // A logged-in user, so FBRoute renders the requested route.
    store.dispatch(setMe({id: 'user-1', username: 'user-1'} as IUser))

    // Disable the first-time tour so FBRoute does not redirect to /welcome.
    store.dispatch(setClientConfig({
        telemetry: false,
        telemetryid: '',
        enablePublicSharedBoards: false,
        teammateNameDisplay: 'username',
        featureFlags: {disableTour: 'true'},
        maxFileSize: 0,
    } as unknown as ClientConfig))

    return store
}

const renderAt = (windowPath: string, store: ReturnType<typeof makeStore>): MemoryHistory => {
    // The router re-syncs to window.location on mount, so drive it from there.
    window.history.replaceState({}, '', windowPath)
    // Start the in-memory history on the bare boards-relative path (basename
    // stripped, query/hash removed) so the '/' HomeToCurrentTeam route isn't
    // exercised AND any surviving query/hash can only come from the mount-time
    // re-sync to window.location, not from the seeded entry.
    const initialEntry = (windowPath.replace('/boards', '') || '/').split('?')[0].split('#')[0] || '/'
    const history = createMemoryHistory({initialEntries: [initialEntry]})
    act(() => {
        render(
            <ReduxProvider store={store}>
                {wrapIntl(<FocalboardRouter history={history}/>)}
            </ReduxProvider>,
        )
    })
    return history
}

describe('router', () => {
    let originalFrontendBaseURL: string | undefined
    let originalIsPlugin: boolean | undefined

    beforeEach(() => {
        originalFrontendBaseURL = windowAny.frontendBaseURL
        originalIsPlugin = windowAny.isFocalboardPlugin
        windowAny.frontendBaseURL = '/boards'
        // ErrorPage only renders its message (instead of auto-redirecting) in plugin mode.
        windowAny.isFocalboardPlugin = true
    })

    afterEach(() => {
        windowAny.frontendBaseURL = originalFrontendBaseURL
        windowAny.isFocalboardPlugin = originalIsPlugin
        window.history.replaceState({}, '', '/')
    })

    describe('FocalboardRouter initial location sync', () => {
        it('preserves the error id query string so a specific error is shown (MM-69658)', () => {
            const store = makeStore()
            const history = renderAt('/boards/error?id=not-logged-in&r=%2Fteam%2Ft1%2F', store)

            // The query string must survive the mount-time re-sync...
            expect(history.location.pathname).toBe('/error')
            expect(history.location.search).toContain('id=not-logged-in')
            expect(history.location.search).toContain('r=')

            // ...so the user sees the specific "log in" error with a working login
            // button, not a bare, generic error page.
            // getByText/getByRole throw when absent, so they double as assertions.
            screen.getByText(/session may have expired/)
            screen.getByRole('button', {name: 'Log in'})
            expect(screen.queryByText('An error occurred.')).toBeNull()
        })

        it('preserves the query string and hash together on the initial sync (MM-69658)', () => {
            const store = makeStore()
            const history = renderAt('/boards/team/t1/b1?foo=bar#section', store)

            expect(history.location.pathname).toBe('/team/t1/b1')
            expect(history.location.search).toBe('?foo=bar')
            expect(history.location.hash).toBe('#section')
            screen.getByText('MOCK_BOARD_PAGE')
        })

        it('leaves a plain path without query or hash untouched', () => {
            const store = makeStore()
            const history = renderAt('/boards/team/t1/b1', store)

            // Guards against the fix appending a stray '?' or '#' to clean URLs.
            expect(history.location.pathname).toBe('/team/t1/b1')
            expect(history.location.search).toBe('')
            expect(history.location.hash).toBe('')
            screen.getByText('MOCK_BOARD_PAGE')
        })
    })
})
