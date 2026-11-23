// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {Client4} from 'mattermost-redux/client'

const boardsV2PluginId = 'com.mattermost.boardsv2'
const dismissedKeyPrefix = 'focalboard_migration_banner_dismissed_'

export const dismissDurationMs = 7 * 24 * 60 * 60 * 1000

export type BoardsV2Summary = {
    mode: 'admin' | 'self_serve'
    is_admin: boolean
    can_migrate: boolean
    available: number | boolean
    not_moved_count: number
    count_capped: boolean
}

let summaryPromise: Promise<BoardsV2Summary | null> | null = null

function siteRoot(): string {
    return (Client4.getUrl() || '').replace(/\/+$/, '')
}

// Boards v2 may not be installed, so a failed or malformed answer means no summary.
// The answer is kept for the rest of the session.
export function fetchBoardsV2Summary(): Promise<BoardsV2Summary | null> {
    if (!summaryPromise) {
        summaryPromise = (async () => {
            try {
                const response = await fetch(`${siteRoot()}/plugins/${boardsV2PluginId}/api/v1/migration/summary`, Client4.getOptions({
                    method: 'GET',
                    headers: {Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest'},
                }))
                if (response.status !== 200) {
                    return null
                }
                const summary = await response.json()
                return summary && typeof summary === 'object' ? summary as BoardsV2Summary : null
            } catch {
                return null
            }
        })()
    }
    return summaryPromise
}

export function resetBoardsV2SummaryForTests(): void {
    summaryPromise = null
}

export function canMigrateFromSummary(summary: BoardsV2Summary | null): boolean {
    return Boolean(summary?.can_migrate && summary.available)
}

export function newBoardPath(teamName: string, channelName: string): string {
    return `/${encodeURIComponent(teamName)}/channels/${encodeURIComponent(channelName)}`
}

export function migrateDialogPath(boardId: string): string {
    return `/plug/${boardsV2PluginId}/migrate?board=${encodeURIComponent(boardId)}`
}

export function siteURL(path: string): string {
    return siteRoot() + path
}

export function isMigrationBannerDismissed(boardId: string, now = Date.now()): boolean {
    try {
        const dismissedAt = Number(localStorage.getItem(dismissedKeyPrefix + boardId))
        return dismissedAt > 0 && now - dismissedAt < dismissDurationMs
    } catch {
        return false
    }
}

export function dismissMigrationBanner(boardId: string, now = Date.now()): void {
    try {
        localStorage.setItem(dismissedKeyPrefix + boardId, String(now))
    } catch {
        // Dismissal only lasts for this view when storage is unavailable.
    }
}

// Opens a Mattermost route without a page reload when the host exposes its history.
export function navigateInMattermost(e: React.MouseEvent<HTMLAnchorElement>, path: string): void {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
        return
    }
    const history = (window as {WebappUtils?: {browserHistory?: {push: (path: string) => void}}}).WebappUtils?.browserHistory
    if (history?.push) {
        e.preventDefault()
        history.push(path)
    }
}
