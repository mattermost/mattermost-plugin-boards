// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react'
import {useHistory, useRouteMatch} from 'react-router-dom'
import {useDispatch, useSelector} from 'react-redux'

import DocSidebar from './DocSidebar'
import PageView from './PageView'
import {fetchPagesForTeam, getChildPageIds, createPage} from '../store/pages'
import type {AppDispatch} from '../store'

// Pages feature — top-level Pages view (Model Y).
//
// Route: /team/:teamId/pages/:pageId?
// Renders DocSidebar (page tree) + PageView (active page editor).

type RouteParams = {
    teamId: string
    pageId?: string
}

export default function DocPage(): JSX.Element {
    const match = useRouteMatch<RouteParams>()
    const history = useHistory()
    const dispatch = useDispatch<AppDispatch>()
    const {teamId, pageId} = match.params

    const [error, setError] = useState<string>('')
    const rootPageIds = useSelector(getChildPageIds(''))
    const [creating, setCreating] = useState(false)

    useEffect(() => {
        if (!teamId) {
            return
        }
        dispatch(fetchPagesForTeam(teamId))
            .unwrap()
            .catch((e) => setError(String(e)))
    }, [teamId, dispatch])

    useEffect(() => {
        if (!pageId && rootPageIds.length > 0) {
            history.replace(`/team/${teamId}/pages/${rootPageIds[0]}`)
        }
    }, [pageId, rootPageIds, teamId, history])

    const onSelect = useCallback((id: string) => {
        history.push(`/team/${teamId}/pages/${id}`)
    }, [teamId, history])

    const onCreateFirst = useCallback(async () => {
        setCreating(true)
        try {
            const result = await dispatch(createPage({teamId, parentId: '', title: ''}))
            if (createPage.fulfilled.match(result)) {
                history.push(`/team/${teamId}/pages/${result.payload.id}`)
            }
        } finally {
            setCreating(false)
        }
    }, [dispatch, teamId, history])

    if (error) {
        return (
            <div style={{padding: 32}}>
                <h2>{'Failed to load pages'}</h2>
                <pre style={{color: '#dc2626'}}>{error}</pre>
            </div>
        )
    }

    return (
        <div className='DocPage focalboard-body' style={{display: 'grid', gridTemplateColumns: '240px 1fr', height: '100%'}}>
            <aside
                className='Sidebar octo-sidebar'
                style={{
                    background: 'rgba(var(--sidebar-bg-rgb), 1)',
                    color: 'rgba(var(--sidebar-text-rgb), 0.72)',
                    overflow: 'auto',
                }}
            >
                <DocSidebar
                    teamId={teamId}
                    activePageId={pageId}
                    onSelect={onSelect}
                />
            </aside>
            <main style={{padding: 24, overflow: 'auto', background: 'var(--center-channel-bg)'}}>
                {pageId ? (
                    <PageView pageId={pageId}/>
                ) : (
                    <div style={{textAlign: 'center', padding: 64, color: 'rgba(var(--center-channel-color-rgb), 0.56)'}}>
                        <h2>{'No pages yet'}</h2>
                        <button
                            onClick={onCreateFirst}
                            disabled={creating}
                            style={{
                                padding: '8px 16px',
                                background: 'var(--button-bg)',
                                color: 'var(--button-color)',
                                border: 0,
                                borderRadius: 4,
                                cursor: 'pointer',
                                fontSize: 14,
                                fontWeight: 600,
                            }}
                        >
                            {creating ? 'Creating…' : 'Create first page'}
                        </button>
                    </div>
                )}
            </main>
        </div>
    )
}
