// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Pages feature — TS model (skeleton).
// See docs/PAGES_PLAN.md for the data-model rationale.

export type Page = {
    id: string
    teamId: string
    parentId: string  // '' for top-level pages, else parent page id
    title: string
    icon: string
    cover: string
    sortOrder: number
    createdBy: string
    modifiedBy: string
    createAt: number
    updateAt: number
    deleteAt: number
}

export type BoardPageRef = {
    boardId: string
    pageId: string
    sortOrder: number
    label: string
    addedBy: string
    addedAt: number
}

export type PageBoardRef = {
    pageId: string
    boardId: string
    label: string
    addedBy: string
    addedAt: number
}

export type PagePatch = Partial<Pick<Page, 'title' | 'icon' | 'cover' | 'parentId' | 'sortOrder'>>

// Tiptap document JSON shape (kept loose; Tiptap exports a richer type).
export type TiptapDoc = {
    type: 'doc'
    content?: unknown[]
}

export type PageContent = {
    pageId: string
    tiptapJson?: TiptapDoc
    yjsState?: Uint8Array
    yjsUpdatesCount: number
    lastSnapshotAt: number
    updateAt: number
    updateBy: string
}

// Phase 2 — per-page ACL override.
export type PageMember = {
    pageId: string
    userId: string
    schemeAdmin: boolean
    schemeEditor: boolean
    schemeCommenter: boolean
    schemeViewer: boolean
}

export type PageChannelLink = {
    pageId: string
    channelId: string
    pinnedBy: string
    pinnedAt: number
}

export const createEmptyPage = (teamId: string, parentId?: string): Page => ({
    id: '',
    teamId,
    parentId: parentId || '',
    title: '',
    icon: '',
    cover: '',
    sortOrder: 0,
    createdBy: '',
    modifiedBy: '',
    createAt: 0,
    updateAt: 0,
    deleteAt: 0,
})

export const createEmptyTiptapDoc = (): TiptapDoc => ({
    type: 'doc',
    content: [{type: 'paragraph'}],
})

export const MAX_PAGE_DEPTH = 10
export const MAX_CHILDREN_PER_PARENT = 1000
export const MAX_SLUG_LENGTH = 50
