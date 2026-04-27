// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'

import type {Page} from '../blocks/page'
import {fetchPagesForTeam, getChildPageIds, getPagesById, createPage, setCurrentPage} from '../store/pages'
import type {AppDispatch} from '../store'

// Pages feature — sidebar tree (Model Y, team-scoped).
// See docs/PAGES_PLAN.md.

type Props = {
    teamId: string
    activePageId?: string
    onSelect: (pageId: string) => void
}

export default function DocSidebar({teamId, activePageId, onSelect}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const pagesById = useSelector(getPagesById)
    // Top-level pages have parentId === '' (empty string)
    const rootIds = useSelector(getChildPageIds(''))

    useEffect(() => {
        if (teamId) {
            dispatch(fetchPagesForTeam(teamId))
        }
    }, [teamId, dispatch])

    const onAdd = useCallback(async (parentId: string) => {
        const result = await dispatch(createPage({teamId, parentId, title: ''}))
        if (createPage.fulfilled.match(result)) {
            dispatch(setCurrentPage(result.payload.id))
            onSelect(result.payload.id)
        }
    }, [teamId, dispatch, onSelect])

    return (
        <div className='DocSidebar'>
            <div className='DocSidebar__header'>
                <button onClick={() => onAdd('')}>{'+ Add page'}</button>
            </div>
            <ul className='DocSidebar__tree' style={{listStyle: 'none', padding: 0, margin: 0}}>
                {rootIds.map((id) => {
                    const page = pagesById[id]
                    if (!page) {
                        return null
                    }
                    return (
                        <PageTreeNode
                            key={id}
                            page={page}
                            depth={0}
                            activePageId={activePageId}
                            onSelect={onSelect}
                            onAdd={onAdd}
                        />
                    )
                })}
                {rootIds.length === 0 && (
                    <li style={{padding: '8px 12px', color: '#9ca3af', fontSize: 13}}>
                        {'No pages yet'}
                    </li>
                )}
            </ul>
        </div>
    )
}

type NodeProps = {
    page: Page
    depth: number
    activePageId?: string
    onSelect: (pageId: string) => void
    onAdd: (parentId: string) => void
}

function PageTreeNode({page, depth, activePageId, onSelect, onAdd}: NodeProps): JSX.Element {
    const pagesById = useSelector(getPagesById)
    const childIds = useSelector(getChildPageIds(page.id))
    const [expanded, setExpanded] = useState(true)

    const isActive = page.id === activePageId
    const indent = 8 + depth * 16
    const hasChildren = childIds.length > 0

    return (
        <li>
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    padding: `4px 8px 4px ${indent}px`,
                    background: isActive ? '#e9efff' : 'transparent',
                    cursor: 'pointer',
                    fontSize: 14,
                    borderRadius: 4,
                }}
            >
                <span
                    onClick={() => hasChildren && setExpanded(!expanded)}
                    style={{
                        width: 14,
                        display: 'inline-block',
                        cursor: hasChildren ? 'pointer' : 'default',
                        color: hasChildren ? '#6b7280' : 'transparent',
                        userSelect: 'none',
                    }}
                >
                    {hasChildren ? (expanded ? '▾' : '▸') : '•'}
                </span>
                <span
                    onClick={() => onSelect(page.id)}
                    style={{flex: 1, marginLeft: 4}}
                >
                    {page.icon ? `${page.icon} ` : ''}{page.title || '(Untitled)'}
                </span>
                <button
                    onClick={(e) => {
                        e.stopPropagation()
                        onAdd(page.id)
                    }}
                    style={{
                        background: 'transparent',
                        border: 0,
                        color: '#6b7280',
                        cursor: 'pointer',
                        padding: '0 4px',
                    }}
                    title='Add sub-page'
                >
                    {'+'}
                </button>
            </div>
            {expanded && hasChildren && (
                <ul style={{listStyle: 'none', padding: 0, margin: 0}}>
                    {childIds.map((cid) => {
                        const c = pagesById[cid]
                        if (!c) {
                            return null
                        }
                        return (
                            <PageTreeNode
                                key={cid}
                                page={c}
                                depth={depth + 1}
                                activePageId={activePageId}
                                onSelect={onSelect}
                                onAdd={onAdd}
                            />
                        )
                    })}
                </ul>
            )}
        </li>
    )
}
