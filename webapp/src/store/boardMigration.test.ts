// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {configureStore} from '@reduxjs/toolkit'
import {mocked} from 'jest-mock'

import {BoardMigration} from '../blocks/boardMigration'
import octoClient from '../octoClient'

import {fetchBoardMigration, getBoardMigration, isBoardMovedReadOnly, reducer} from './boardMigration'

import {RootState} from './index'

jest.mock('../octoClient')
const mockedOctoClient = mocked(octoClient)

const notMoved: BoardMigration = {moved: false, can_open: false, read_only: false}
const moved: BoardMigration = {moved: true, can_open: true, read_only: true, team_name: 'eng', channel_name: 'roadmap'}

describe('store/boardMigration', () => {
    afterEach(() => {
        jest.clearAllMocks()
    })

    test('an older response does not overwrite a newer one', async () => {
        const store = configureStore({reducer: {boardMigration: reducer}})
        let resolveOlder: (value: BoardMigration) => void = () => {}
        mockedOctoClient.getBoardMigration.mockReturnValueOnce(new Promise((resolve) => {
            resolveOlder = resolve
        }))
        mockedOctoClient.getBoardMigration.mockResolvedValueOnce(moved)

        const older = store.dispatch(fetchBoardMigration('board-1'))
        await store.dispatch(fetchBoardMigration('board-1'))
        resolveOlder(notMoved)
        await older

        expect(getBoardMigration('board-1')(store.getState() as unknown as RootState)).toEqual(moved)
    })

    test('requests for different boards do not discard each other', async () => {
        const store = configureStore({reducer: {boardMigration: reducer}})
        mockedOctoClient.getBoardMigration.mockResolvedValueOnce(moved)
        mockedOctoClient.getBoardMigration.mockResolvedValueOnce(notMoved)

        await Promise.all([store.dispatch(fetchBoardMigration('board-1')), store.dispatch(fetchBoardMigration('board-2'))])

        const state = store.getState() as unknown as RootState
        expect(getBoardMigration('board-1')(state)).toEqual(moved)
        expect(getBoardMigration('board-2')(state)).toEqual(notMoved)
    })

    describe('isBoardMovedReadOnly', () => {
        const stateWith = (legacyMode: boolean, movedBoardsReadOnly: boolean, migration: BoardMigration) => ({
            clientConfig: {value: {legacyMode, movedBoardsReadOnly}},
            boardMigration: {byBoardId: {'board-1': migration}},
        }) as unknown as RootState

        test('is read-only for a read-only moved board in Legacy Mode with the setting on', () => {
            expect(isBoardMovedReadOnly('board-1')(stateWith(true, true, moved))).toBe(true)
        })

        test('is not read-only outside Legacy Mode', () => {
            expect(isBoardMovedReadOnly('board-1')(stateWith(false, true, moved))).toBe(false)
        })

        test('is not read-only when the setting is off', () => {
            expect(isBoardMovedReadOnly('board-1')(stateWith(true, false, moved))).toBe(false)
        })

        test('is not read-only for a board that did not move', () => {
            expect(isBoardMovedReadOnly('board-1')(stateWith(true, true, notMoved))).toBe(false)
        })
    })
})
