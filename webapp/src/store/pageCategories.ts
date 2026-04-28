// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createSlice, PayloadAction, createAsyncThunk, createSelector} from '@reduxjs/toolkit'

import {default as client} from '../octoClient'
import type {PageCategory, PageCategoryAssignment} from '../blocks/page'

import type {RootState} from './index'

// Per-user page categories — sidebar grouping.
// Slice 1: CRUD on categories themselves.
// Slice 2: page→category assignments (assignmentByPageId).

export type PageCategoriesState = {
    byId: {[id: string]: PageCategory}
    // pageId → categoryId. Absence means uncategorized.
    assignmentByPageId: {[pageId: string]: string}
    loading: boolean
}

const initialState: PageCategoriesState = {
    byId: {},
    assignmentByPageId: {},
    loading: false,
}

export const fetchPageCategories = createAsyncThunk<PageCategory[], string>(
    'pageCategories/fetch',
    async (teamId) => {
        const resp = await client.getPageCategories(teamId)
        if (!resp.ok) {
            throw new Error(`fetchPageCategories ${teamId}: HTTP ${resp.status}`)
        }
        const json = await resp.json()
        return (json || []) as PageCategory[]
    },
)

export const createPageCategory = createAsyncThunk<PageCategory, {teamId: string; name: string}>(
    'pageCategories/create',
    async ({teamId, name}) => {
        const resp = await client.createPageCategory(teamId, name)
        if (!resp.ok) {
            const txt = await resp.text()
            throw new Error(`createPageCategory: ${resp.status} ${txt}`)
        }
        return (await resp.json()) as PageCategory
    },
)

export const renamePageCategory = createAsyncThunk<PageCategory, {teamId: string; categoryId: string; name: string}>(
    'pageCategories/rename',
    async ({teamId, categoryId, name}) => {
        const resp = await client.updatePageCategory(teamId, categoryId, {name})
        if (!resp.ok) {
            throw new Error(`renamePageCategory: HTTP ${resp.status}`)
        }
        return (await resp.json()) as PageCategory
    },
)

export const deletePageCategory = createAsyncThunk<string, {teamId: string; categoryId: string}>(
    'pageCategories/delete',
    async ({teamId, categoryId}) => {
        const resp = await client.deletePageCategory(teamId, categoryId)
        if (!resp.ok) {
            throw new Error(`deletePageCategory: HTTP ${resp.status}`)
        }
        return categoryId
    },
)

export const fetchPageCategoryAssignments = createAsyncThunk<PageCategoryAssignment[], string>(
    'pageCategories/fetchAssignments',
    async (teamId) => {
        const resp = await client.getPageCategoryAssignments(teamId)
        if (!resp.ok) {
            throw new Error(`fetchPageCategoryAssignments: HTTP ${resp.status}`)
        }
        const json = await resp.json()
        return (json || []) as PageCategoryAssignment[]
    },
)

export const setPageCategory = createAsyncThunk<void, {teamId: string; categoryId: string; pageId: string; sortOrder?: number}>(
    'pageCategories/setAssignment',
    async ({teamId, categoryId, pageId, sortOrder}) => {
        const resp = await client.setPageCategory(teamId, categoryId, pageId, sortOrder || 0)
        if (!resp.ok) {
            throw new Error(`setPageCategory: HTTP ${resp.status}`)
        }
    },
)

export const unsetPageCategory = createAsyncThunk<void, {teamId: string; pageId: string}>(
    'pageCategories/unsetAssignment',
    async ({teamId, pageId}) => {
        const resp = await client.unsetPageCategory(teamId, pageId)
        if (!resp.ok) {
            throw new Error(`unsetPageCategory: HTTP ${resp.status}`)
        }
    },
)

const slice = createSlice({
    name: 'pageCategories',
    initialState,
    reducers: {
        upsertPageCategory: (state, action: PayloadAction<PageCategory>) => {
            const c = action.payload
            if (c.deleteAt && c.deleteAt > 0) {
                delete state.byId[c.id]
                // Also drop any assignments to this deleted category.
                for (const pageId of Object.keys(state.assignmentByPageId)) {
                    if (state.assignmentByPageId[pageId] === c.id) {
                        delete state.assignmentByPageId[pageId]
                    }
                }
                return
            }
            state.byId[c.id] = c
        },
        removePageCategory: (state, action: PayloadAction<string>) => {
            delete state.byId[action.payload]
        },
        // Optimistic local update before server WS confirms.
        setPageCategoryLocal: (state, action: PayloadAction<{pageId: string; categoryId: string}>) => {
            const {pageId, categoryId} = action.payload
            if (categoryId) {
                state.assignmentByPageId[pageId] = categoryId
            } else {
                delete state.assignmentByPageId[pageId]
            }
        },
    },
    extraReducers: (builder) => {
        builder.addCase(fetchPageCategories.pending, (state) => {
            state.loading = true
        })
        builder.addCase(fetchPageCategories.fulfilled, (state, action) => {
            state.loading = false
            const fresh: {[id: string]: PageCategory} = {}
            for (const c of action.payload) {
                fresh[c.id] = c
            }
            state.byId = fresh
        })
        builder.addCase(fetchPageCategories.rejected, (state) => {
            state.loading = false
        })
        builder.addCase(fetchPageCategoryAssignments.fulfilled, (state, action) => {
            const fresh: {[pageId: string]: string} = {}
            for (const a of action.payload) {
                fresh[a.pageID] = a.categoryID
            }
            state.assignmentByPageId = fresh
        })
        // No state mutation on create/rename/delete fulfilled — the
        // server's WS broadcast is the single source of truth (mirrors
        // Boards' mutator pattern). Updating here AND on WS would cause
        // a brief double-entry and complicates dedupe.
    },
})

export const {upsertPageCategory, removePageCategory, setPageCategoryLocal} = slice.actions
export const {reducer} = slice

const getState = (state: RootState): PageCategoriesState => (state as unknown as {pageCategories: PageCategoriesState}).pageCategories

export const getPageCategories = createSelector(
    getState,
    (s): PageCategory[] => Object.values(s.byId).sort((a, b) =>
        (a.sortOrder - b.sortOrder) || (a.createAt - b.createAt),
    ),
)

export const getPageCategoryAssignments = createSelector(
    getState,
    (s) => s.assignmentByPageId,
)
