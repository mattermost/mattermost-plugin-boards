// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {configureStore} from '@reduxjs/toolkit'
import {act, render, screen, waitFor} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import {mocked} from 'jest-mock'

import {BoardMigration} from '../../blocks/boardMigration'
import octoClient from '../../octoClient'
import {reducer as boardMigrationReducer} from '../../store/boardMigration'
import {reducer as clientConfigReducer} from '../../store/clientConfig'
import {resetBoardsV2SummaryForTests} from '../../migration/boardsV2'
import {wrapIntl} from '../../testUtils'

import MigrationBanner from './migrationBanner'

jest.mock('../../octoClient')
const mockedOctoClient = mocked(octoClient)

const movedAt = Date.UTC(2023, 10, 15, 12)
const formattedDate = new Intl.DateTimeFormat('en', {year: 'numeric', month: 'short', day: 'numeric'}).format(new Date(movedAt))
const dismissedKey = 'focalboard_migration_banner_dismissed_board-1'

const notMoved: BoardMigration = {moved: false, can_open: false, read_only: false}
const moved: BoardMigration = {
    moved: true,
    channel_id: 'channel-1',
    team_name: 'eng',
    channel_name: 'roadmap',
    channel_display_name: 'Roadmap',
    moved_at: movedAt,
    can_open: true,
    read_only: false,
}

const summary = {mode: 'self_serve', is_admin: false, can_migrate: true, available: true, not_moved_count: 3, count_capped: false}

const mockSummaryResponse = (status: number, body: unknown) => {
    global.fetch = jest.fn().mockResolvedValue({status, json: async () => body}) as unknown as typeof fetch
}

const clientConfig = (legacyMode: boolean) => ({
    value: {
        telemetry: false,
        telemetryid: '',
        enablePublicSharedBoards: false,
        teammateNameDisplay: 'username',
        featureFlags: {},
        maxFileSize: 0,
        legacyMode,
        movedBoardsReadOnly: true,
    },
})

const renderBanner = async (migration: BoardMigration | undefined, isTemplate = false, legacyMode = true) => {
    mockedOctoClient.getBoardMigration.mockResolvedValue(migration)
    const store = configureStore({
        reducer: {boardMigration: boardMigrationReducer, clientConfig: clientConfigReducer},
        preloadedState: {clientConfig: clientConfig(legacyMode)},
    })
    const result = render(wrapIntl(
        <ReduxProvider store={store}>
            <MigrationBanner
                boardId='board-1'
                isTemplate={isTemplate}
            />
        </ReduxProvider>,
    ))
    await act(async () => {
        await Promise.resolve()
    })
    return result
}

describe('components/migrationBanner', () => {
    const originalFetch = global.fetch

    beforeEach(() => {
        localStorage.clear()
        resetBoardsV2SummaryForTests()
        mockSummaryResponse(200, summary)
    })

    afterEach(() => {
        global.fetch = originalFetch
        delete (window as {WebappUtils?: unknown}).WebappUtils
        jest.clearAllMocks()
    })

    describe('moved board', () => {
        it('links to the new board when the user can open it', async () => {
            await renderBanner(moved)

            const region = screen.getByRole('region', {name: 'Board migration'})
            expect(region).toHaveTextContent(`This board was migrated to the new version of Boards on ${formattedDate}. Changes made here won't be copied over.`)
            expect(screen.getByRole('status')).toBeInTheDocument()

            const link = screen.getByRole('link', {name: 'Open new board'})
            expect(link).toHaveAttribute('href', '/eng/channels/roadmap')
            expect(screen.queryByText('Ask a board member to add you to the new board.')).not.toBeInTheDocument()
            expect(screen.queryByRole('button')).not.toBeInTheDocument()
        })

        it('opens the new board through the Mattermost history when available', async () => {
            const push = jest.fn()
            ;(window as {WebappUtils?: unknown}).WebappUtils = {browserHistory: {push}}
            await renderBanner(moved)

            userEvent.click(screen.getByRole('link', {name: 'Open new board'}))

            expect(push).toHaveBeenCalledWith('/eng/channels/roadmap')
        })

        it('asks for an invitation when the user cannot open the new board', async () => {
            await renderBanner({...moved, can_open: false, channel_name: undefined, channel_display_name: undefined})

            expect(screen.getByRole('region', {name: 'Board migration'})).toHaveTextContent('Ask a board member to add you to the new board.')
            expect(screen.queryByRole('link')).not.toBeInTheDocument()
            expect(screen.queryByRole('button')).not.toBeInTheDocument()
        })

        it('says the board is read-only instead of promising copied changes', async () => {
            await renderBanner({...moved, read_only: true})

            const region = screen.getByRole('region', {name: 'Board migration'})
            expect(region).toHaveTextContent(`This board was migrated to the new version of Boards on ${formattedDate}. This board is now read-only here.`)
            expect(region).not.toHaveTextContent("Changes made here won't be copied over.")
            expect(screen.getByRole('link', {name: 'Open new board'})).toBeInTheDocument()
        })

        it('shows without asking Boards v2 for its summary', async () => {
            await renderBanner(moved)

            expect(global.fetch).not.toHaveBeenCalled()
        })
    })

    describe('refreshing', () => {
        it('reloads the migration state when the window regains focus', async () => {
            await renderBanner(notMoved)
            expect(mockedOctoClient.getBoardMigration).toHaveBeenCalledTimes(1)

            mockedOctoClient.getBoardMigration.mockResolvedValue(moved)
            await act(async () => {
                window.dispatchEvent(new Event('focus'))
            })

            expect(mockedOctoClient.getBoardMigration).toHaveBeenCalledTimes(2)
            await waitFor(() => expect(screen.getByRole('link', {name: 'Open new board'})).toBeInTheDocument())
        })
    })

    describe('board that did not move', () => {
        it('offers to migrate the board when the user can migrate', async () => {
            const {container} = await renderBanner(notMoved)

            await waitFor(() => expect(screen.getByRole('region', {name: 'Board migration'})).toBeInTheDocument())
            expect(screen.getByRole('region')).toHaveTextContent("There's a new version of Boards. Migrate this board to keep working on it with your team.")
            expect(screen.getByRole('link', {name: 'Migrate this board'})).toHaveAttribute('href', '/plug/com.mattermost.boardsv2/migrate?board=board-1')
            expect(screen.getByRole('button', {name: 'Not now'})).toBeInTheDocument()
            expect(container).toMatchSnapshot()
            expect(global.fetch).toHaveBeenCalledTimes(1)
        })

        it('shows nothing when the user cannot migrate', async () => {
            mockSummaryResponse(200, {...summary, can_migrate: false})
            await renderBanner(notMoved)

            await waitFor(() => expect(global.fetch).toHaveBeenCalled())
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
        })

        it('shows nothing when no boards are available to migrate', async () => {
            mockSummaryResponse(200, {...summary, available: false})
            await renderBanner(notMoved)

            await waitFor(() => expect(global.fetch).toHaveBeenCalled())
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
        })

        it('shows nothing when the summary flags are not booleans', async () => {
            mockSummaryResponse(200, {...summary, can_migrate: 'false', available: '0'})
            await renderBanner(notMoved)

            await waitFor(() => expect(global.fetch).toHaveBeenCalled())
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
        })

        it('shows nothing when Boards v2 is not installed', async () => {
            mockSummaryResponse(404, {})
            await renderBanner(notMoved)

            await waitFor(() => expect(global.fetch).toHaveBeenCalled())
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
        })

        it('shows nothing when the Boards v2 request fails', async () => {
            global.fetch = jest.fn().mockRejectedValue(new Error('network')) as unknown as typeof fetch
            await renderBanner(notMoved)

            await waitFor(() => expect(global.fetch).toHaveBeenCalled())
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
        })

        it('shows nothing for templates', async () => {
            await renderBanner(notMoved, true)

            expect(global.fetch).not.toHaveBeenCalled()
            expect(mockedOctoClient.getBoardMigration).not.toHaveBeenCalled()
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
        })

        it('hides the banner and remembers the dismissal', async () => {
            const dateNow = jest.spyOn(Date, 'now').mockReturnValue(1700000000000)
            await renderBanner(notMoved)
            await waitFor(() => expect(screen.getByRole('button', {name: 'Not now'})).toBeInTheDocument())

            userEvent.click(screen.getByRole('button', {name: 'Not now'}))

            expect(screen.queryByRole('region')).not.toBeInTheDocument()
            expect(localStorage.getItem(dismissedKey)).toBe('1700000000000')
            dateNow.mockRestore()
        })

        it('stays hidden for 7 days after a dismissal', async () => {
            const dateNow = jest.spyOn(Date, 'now').mockReturnValue(1700000000000 + (6 * 24 * 60 * 60 * 1000))
            localStorage.setItem(dismissedKey, '1700000000000')
            await renderBanner(notMoved)

            await waitFor(() => expect(global.fetch).toHaveBeenCalled())
            expect(screen.queryByRole('region')).not.toBeInTheDocument()
            dateNow.mockRestore()
        })

        it('shows again after the 7 day dismissal ends', async () => {
            const dateNow = jest.spyOn(Date, 'now').mockReturnValue(1700000000000 + (8 * 24 * 60 * 60 * 1000))
            localStorage.setItem(dismissedKey, '1700000000000')
            await renderBanner(notMoved)

            await waitFor(() => expect(screen.getByRole('region', {name: 'Board migration'})).toBeInTheDocument())
            dateNow.mockRestore()
        })
    })

    describe('outside Legacy Mode', () => {
        it('shows nothing and makes no requests for a board that did not move', async () => {
            await renderBanner(notMoved, false, false)

            expect(screen.queryByRole('region')).not.toBeInTheDocument()
            expect(mockedOctoClient.getBoardMigration).not.toHaveBeenCalled()
            expect(global.fetch).not.toHaveBeenCalled()
        })

        it('shows nothing and makes no requests for a moved board', async () => {
            await renderBanner(moved, false, false)

            expect(screen.queryByRole('region')).not.toBeInTheDocument()
            expect(mockedOctoClient.getBoardMigration).not.toHaveBeenCalled()
            expect(global.fetch).not.toHaveBeenCalled()
        })

        it('does not refresh the migration state when the window regains focus', async () => {
            await renderBanner(notMoved, false, false)

            await act(async () => {
                window.dispatchEvent(new Event('focus'))
            })

            expect(mockedOctoClient.getBoardMigration).not.toHaveBeenCalled()
        })
    })

    it('shows nothing when the migration lookup fails', async () => {
        await renderBanner(undefined)

        expect(screen.queryByRole('region')).not.toBeInTheDocument()
        expect(global.fetch).not.toHaveBeenCalled()
    })
})
