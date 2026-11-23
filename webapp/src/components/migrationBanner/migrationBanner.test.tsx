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

const summary = {mode: 'self_serve', is_admin: false, can_migrate: true, available: 3, not_moved_count: 3, count_capped: false}

const mockSummaryResponse = (status: number, body: unknown) => {
    global.fetch = jest.fn().mockResolvedValue({status, json: async () => body}) as unknown as typeof fetch
}

const renderBanner = async (migration: BoardMigration | undefined, isTemplate = false) => {
    mockedOctoClient.getBoardMigration.mockResolvedValue(migration)
    const store = configureStore({reducer: {boardMigration: boardMigrationReducer}})
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
            expect(region).toHaveTextContent(`This board moved to the new Boards on ${formattedDate}. Changes made here won't be copied over.`)
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
            expect(region).toHaveTextContent(`This board moved to the new Boards on ${formattedDate}. This board is read-only.`)
            expect(region).not.toHaveTextContent("Changes made here won't be copied over.")
            expect(screen.getByRole('link', {name: 'Open new board'})).toBeInTheDocument()
        })

        it('shows without asking Boards v2 for its summary', async () => {
            await renderBanner(moved)

            expect(global.fetch).not.toHaveBeenCalled()
        })
    })

    describe('board that did not move', () => {
        it('offers to move the board when the user can migrate', async () => {
            const {container} = await renderBanner(notMoved)

            await waitFor(() => expect(screen.getByRole('region', {name: 'Board migration'})).toBeInTheDocument())
            expect(screen.getByRole('region')).toHaveTextContent('Boards now live in channels. Move this board to keep working on it with your team.')
            expect(screen.getByRole('link', {name: 'Move this board'})).toHaveAttribute('href', '/plug/com.mattermost.boardsv2/migrate?board=board-1')
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

        it('shows nothing when no boards are available to move', async () => {
            mockSummaryResponse(200, {...summary, available: 0})
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

    it('shows nothing when the migration lookup fails', async () => {
        await renderBanner(undefined)

        expect(screen.queryByRole('region')).not.toBeInTheDocument()
        expect(global.fetch).not.toHaveBeenCalled()
    })
})
