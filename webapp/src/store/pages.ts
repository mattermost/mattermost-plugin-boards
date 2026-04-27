// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createSlice, PayloadAction, createAsyncThunk, createSelector} from '@reduxjs/toolkit'

import {default as client} from '../octoClient'
import type {Page, PageContent, TiptapDoc} from '../blocks/page'

import type {RootState} from './index'

// Pages feature — Redux slice.
// See docs/PAGES_PLAN.md.

export type PagesState = {
    current: string
    loading: boolean
    pages: {[id: string]: Page}
    contents: {[pageId: string]: PageContent}
    childrenByParent: {[parentId: string]: string[]}
    // PageSelector modal state — when non-empty, the channel id we are
    // in "link a page to this channel" mode for. Mirrors boards' linkToChannel.
    linkToChannel: string
}

const initialState: PagesState = {
    current: '',
    loading: false,
    pages: {},
    contents: {},
    childrenByParent: {},
    linkToChannel: '',
}

// ─── Thunks ──────────────────────────────────────────────────────────

export const fetchPagesForTeam = createAsyncThunk<Page[], string>(
    'pages/fetchForTeam',
    async (teamId) => {
        const resp = await client.getPagesForTeam(teamId)
        if (!resp.ok) {
            throw new Error(`fetchPagesForTeam ${teamId}: HTTP ${resp.status}`)
        }
        const json = await resp.json()
        return json as Page[]
    },
)

export const fetchPage = createAsyncThunk<Page, string>(
    'pages/fetchOne',
    async (pageId) => {
        const resp = await client.getPage(pageId)
        if (!resp.ok) {
            throw new Error(`fetchPage ${pageId}: HTTP ${resp.status}`)
        }
        return (await resp.json()) as Page
    },
)

export const fetchPageContent = createAsyncThunk<PageContent, string>(
    'pages/fetchContent',
    async (pageId) => {
        const resp = await client.getPageContent(pageId)
        if (!resp.ok) {
            throw new Error(`fetchPageContent ${pageId}: HTTP ${resp.status}`)
        }
        return (await resp.json()) as PageContent
    },
)

export const createPage = createAsyncThunk<
    Page,
    {teamId: string; parentId?: string; title?: string}
>(
    'pages/create',
    async ({teamId, parentId, title}) => {
        const resp = await client.createPage(teamId, parentId || '', title || '')
        if (!resp.ok) {
            const txt = await resp.text()
            throw new Error(`createPage failed: HTTP ${resp.status} ${txt}`)
        }
        return (await resp.json()) as Page
    },
)

export const savePageContent = createAsyncThunk<
    void,
    {pageId: string; tiptapJson: TiptapDoc}
>(
    'pages/saveContent',
    async ({pageId, tiptapJson}) => {
        const resp = await client.savePageContent(pageId, tiptapJson)
        if (!resp.ok) {
            const txt = await resp.text()
            throw new Error(`savePageContent failed: HTTP ${resp.status} ${txt}`)
        }
    },
)

// ─── Slice ───────────────────────────────────────────────────────────

const pagesSlice = createSlice({
    name: 'pages',
    initialState,
    reducers: {
        setCurrentPage: (state, action: PayloadAction<string>) => {
            state.current = action.payload
        },
        upsertPage: (state, action: PayloadAction<Page>) => {
            state.pages[action.payload.id] = action.payload
            // index for tree lookup
            const parent = action.payload.parentId
            const list = state.childrenByParent[parent] || []
            if (!list.includes(action.payload.id)) {
                state.childrenByParent[parent] = [...list, action.payload.id]
            }
        },
        upsertPages: (state, action: PayloadAction<Page[]>) => {
            for (const p of action.payload) {
                state.pages[p.id] = p
            }
            // rebuild childrenByParent index over the loaded set
            const fresh: {[k: string]: string[]} = {}
            for (const p of action.payload) {
                if (!fresh[p.parentId]) {
                    fresh[p.parentId] = []
                }
                fresh[p.parentId].push(p.id)
            }
            for (const k of Object.keys(fresh)) {
                state.childrenByParent[k] = fresh[k]
            }
        },
        removePage: (state, action: PayloadAction<string>) => {
            const id = action.payload
            const p = state.pages[id]
            delete state.pages[id]
            delete state.contents[id]
            if (p) {
                const list = state.childrenByParent[p.parentId]
                if (list) {
                    state.childrenByParent[p.parentId] = list.filter((x) => x !== id)
                }
            }
        },
        upsertPageContent: (state, action: PayloadAction<PageContent>) => {
            state.contents[action.payload.pageId] = action.payload
        },
        setLinkPageToChannel: (state, action: PayloadAction<string>) => {
            state.linkToChannel = action.payload
        },
    },
    extraReducers: (builder) => {
        builder.addCase(fetchPagesForTeam.pending, (state) => {
            state.loading = true
        })
        builder.addCase(fetchPagesForTeam.fulfilled, (state, action) => {
            state.loading = false
            for (const p of action.payload) {
                state.pages[p.id] = p
            }
            const fresh: {[k: string]: string[]} = {}
            for (const p of action.payload) {
                if (!fresh[p.parentId]) {
                    fresh[p.parentId] = []
                }
                fresh[p.parentId].push(p.id)
            }
            state.childrenByParent = fresh
        })
        builder.addCase(fetchPagesForTeam.rejected, (state) => {
            state.loading = false
        })
        builder.addCase(fetchPage.fulfilled, (state, action) => {
            state.pages[action.payload.id] = action.payload
        })
        builder.addCase(fetchPageContent.fulfilled, (state, action) => {
            state.contents[action.payload.pageId] = action.payload
        })
        builder.addCase(createPage.fulfilled, (state, action) => {
            const p = action.payload
            state.pages[p.id] = p
            const list = state.childrenByParent[p.parentId] || []
            if (!list.includes(p.id)) {
                state.childrenByParent[p.parentId] = [...list, p.id]
            }
        })
    },
})

export const {
    setCurrentPage,
    upsertPage,
    upsertPages,
    removePage,
    upsertPageContent,
    setLinkPageToChannel,
} = pagesSlice.actions

export const {reducer} = pagesSlice

// ─── Selectors ───────────────────────────────────────────────────────

export const getPagesState = (state: RootState): PagesState => (state as unknown as {pages: PagesState}).pages

export const getCurrentPageId = (state: RootState): string => getPagesState(state).current

export const getPagesById = (state: RootState): {[id: string]: Page} => getPagesState(state).pages

export const getPage = (id: string) => createSelector(
    getPagesById,
    (byId): Page | undefined => byId[id],
)

export const getChildPageIds = (parentId: string) => createSelector(
    getPagesState,
    (p): string[] => p.childrenByParent[parentId] || [],
)

export const getChildPages = (parentId: string) => createSelector(
    getPagesById,
    getChildPageIds(parentId),
    (byId, ids): Page[] => ids.map((id) => byId[id]).filter(Boolean),
)

export const getAncestorTrail = (pageId: string) => createSelector(
    getPagesById,
    (byId): Page[] => {
        const trail: Page[] = []
        let cur = byId[pageId]
        const seen = new Set<string>()
        while (cur && !seen.has(cur.id)) {
            seen.add(cur.id)
            trail.unshift(cur)
            cur = byId[cur.parentId]
        }
        return trail
    },
)

export const getPageContent = (pageId: string) => createSelector(
    getPagesState,
    (p): PageContent | undefined => p.contents[pageId],
)

export const getLinkPageToChannel = (state: RootState): string => getPagesState(state).linkToChannel

// TODO Phase 1: register `reducer` in store/index.ts under key 'pages'.
