// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createSlice, createAsyncThunk} from '@reduxjs/toolkit'

import {BoardMigration} from '../blocks/boardMigration'
import octoClient from '../octoClient'

import {getMovedBoardsReadOnly} from './clientConfig'

import {RootState} from './index'

type BoardMigrationState = {
    byBoardId: Record<string, BoardMigration>
}

// Responses can arrive out of order, so only the newest request per board may update the store.
const latestRequestByBoard = new Map<string, string>()

export const fetchBoardMigration = createAsyncThunk(
    'boardMigration/fetchBoardMigration',
    async (boardId: string, {requestId}) => {
        latestRequestByBoard.set(boardId, requestId)
        let migration: BoardMigration | undefined
        try {
            migration = await octoClient.getBoardMigration(boardId)
        } catch {
            migration = undefined
        }
        return {boardId, migration, stale: latestRequestByBoard.get(boardId) !== requestId}
    },
)

const boardMigrationSlice = createSlice({
    name: 'boardMigration',
    initialState: {byBoardId: {}} as BoardMigrationState,
    reducers: {},
    extraReducers: (builder) => {
        builder.addCase(fetchBoardMigration.fulfilled, (state, action) => {
            const {boardId, migration, stale} = action.payload
            if (stale) {
                return
            }
            if (migration) {
                state.byBoardId[boardId] = migration
            } else {
                delete state.byBoardId[boardId]
            }
        })
    },
})

export const {reducer} = boardMigrationSlice

// Selectors tolerate a missing slice because many component tests build partial stores.
export function getBoardMigration(boardId: string): (state: RootState) => BoardMigration | undefined {
    return (state: RootState) => state.boardMigration?.byBoardId?.[boardId]
}

export function isBoardMovedReadOnly(boardId: string): (state: RootState) => boolean {
    return (state: RootState) => {
        if (!getMovedBoardsReadOnly(state)) {
            return false
        }
        const migration = state.boardMigration?.byBoardId?.[boardId]
        return Boolean(migration?.moved && migration.read_only)
    }
}
