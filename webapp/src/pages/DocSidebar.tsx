// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useMemo, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'

import type {Page} from '../blocks/page'
import {fetchPagesForTeam, getChildPageIds, getPagesById, createPage, setCurrentPage} from '../store/pages'
import type {AppDispatch} from '../store'

import SearchIcon from '../widgets/icons/search'
import AddIcon from '../widgets/icons/add'
import IconButton from '../widgets/buttons/iconButton'
import ChevronDown from '../widgets/icons/chevronDown'
import ChevronRight from '../widgets/icons/chevronRight'

import PageMenu, {getHiddenPageIds} from './PageMenu'

import './DocSidebar.scss'

// Pages feature — sidebar tree (Model Y, team-scoped).
//
// Header mirrors BoardsSwitcher visually: a "Find pages" search box +
// "+" add button. Search query filters the tree in-memory by title
// (case-insensitive substring), flattening all matches under the root.
//
// See docs/PAGES_PLAN.md.

type Props = {
    teamId: string
    activePageId?: string
    onSelect: (pageId: string) => void
}

export default function DocSidebar({teamId, activePageId, onSelect}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const pagesById = useSelector(getPagesById)
    const rootIds = useSelector(getChildPageIds(''))
    const [query, setQuery] = useState<string>('')
    const [categoryCollapsed, setCategoryCollapsed] = useState<boolean>(false)
    // Hidden page IDs (localStorage). bumped on hide/unhide to re-render.
    const [hiddenTick, setHiddenTick] = useState(0)
    const hiddenIds = useMemo(() => getHiddenPageIds(), [hiddenTick])
    const onAfterHide = useCallback(() => setHiddenTick((t) => t + 1), [])
    const visibleRootIds = useMemo(() => rootIds.filter((id) => !hiddenIds.has(id)), [rootIds, hiddenIds])

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

    // Search filter: when query is non-empty, return a flat list of matching pages.
    const flatMatches: Page[] = useMemo(() => {
        const q = query.trim().toLowerCase()
        if (!q) {
            return []
        }
        return Object.values(pagesById).filter((p) =>
            (p.title || '').toLowerCase().includes(q),
        )
    }, [query, pagesById])

    const isSearching = query.trim().length > 0

    return (
        <div className='DocSidebar'>
            <div className='BoardsSwitcherWrapper'>
                <div className='BoardsSwitcher' style={{flex: 1}}>
                    <SearchIcon/>
                    <input
                        type='text'
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        placeholder={'Find pages'}
                        style={{
                            flex: 1, background: 'transparent', border: 0,
                            outline: 'none', color: 'inherit', fontSize: 12,
                            marginLeft: 4,
                        }}
                    />
                    {query && (
                        <button
                            onClick={() => setQuery('')}
                            style={{
                                background: 'transparent', border: 0, cursor: 'pointer',
                                color: 'inherit', opacity: 0.6, fontSize: 14, padding: '0 4px',
                            }}
                            title='Clear'
                        >
                            {'×'}
                        </button>
                    )}
                </div>
                <IconButton
                    size='small'
                    inverted={true}
                    className='add-board-icon'
                    icon={<AddIcon/>}
                    title='Add page'
                    onClick={() => onAdd('')}
                />
            </div>

            {isSearching ? (
                <ul className='DocSidebar__searchResults' style={{listStyle: 'none', padding: 0, margin: '8px 0'}}>
                    {flatMatches.length === 0 && (
                        <li style={{padding: '8px 16px', color: '#9ca3af', fontSize: 12}}>
                            {`No pages match "${query}"`}
                        </li>
                    )}
                    {flatMatches.map((p) => {
                        const isActive = p.id === activePageId
                        return (
                            <li key={p.id}>
                                <button
                                    onClick={() => onSelect(p.id)}
                                    style={{
                                        display: 'flex', alignItems: 'center', width: '100%',
                                        padding: '4px 16px', background: isActive ? 'rgba(var(--button-bg-rgb), 0.12)' : 'transparent',
                                        color: isActive ? 'rgba(var(--button-bg-rgb), 1)' : 'inherit',
                                        border: 0, cursor: 'pointer', fontSize: 13,
                                        textAlign: 'left',
                                    }}
                                >
                                    <span style={{marginRight: 8}}>{p.icon || '📄'}</span>
                                    <span style={{flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                                        {p.title || '(Untitled)'}
                                    </span>
                                </button>
                            </li>
                        )
                    })}
                </ul>
            ) : (
                <div className='octo-sidebar-list'>
                    {/* Category header — mirrors Boards' SidebarCategory pattern */}
                    <div className={`octo-sidebar-item category ${categoryCollapsed ? 'collapsed' : 'expanded'}`}>
                        <div
                            className='octo-sidebar-title category-title'
                            title='Pages'
                            onClick={() => setCategoryCollapsed(!categoryCollapsed)}
                        >
                            {categoryCollapsed ? <ChevronRight/> : <ChevronDown/>}
                            {'Pages'}
                        </div>
                    </div>
                    {!categoryCollapsed && (
                        <ul className='DocSidebar__tree' style={{listStyle: 'none', padding: 0, margin: 0}}>
                            {visibleRootIds.map((id) => {
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
                                        onAfterHide={onAfterHide}
                                        hiddenIds={hiddenIds}
                                    />
                                )
                            })}
                            {visibleRootIds.length === 0 && (
                                <li style={{padding: '8px 16px', color: '#9ca3af', fontSize: 12}}>
                                    {rootIds.length === 0 ? 'No pages yet. Click + above to create one.' : 'All pages hidden.'}
                                </li>
                            )}
                        </ul>
                    )}
                </div>
            )}
        </div>
    )
}

type NodeProps = {
    page: Page
    depth: number
    activePageId?: string
    onSelect: (pageId: string) => void
    onAdd: (parentId: string) => void
    onAfterHide?: () => void
    hiddenIds: Set<string>
}

function PageTreeNode({page, depth, activePageId, onSelect, onAdd, onAfterHide, hiddenIds}: NodeProps): JSX.Element {
    const pagesById = useSelector(getPagesById)
    const childIds = useSelector(getChildPageIds(page.id))
    const visibleChildIds = childIds.filter((id) => !hiddenIds.has(id))
    const [expanded, setExpanded] = useState(true)

    const isActive = page.id === activePageId
    const indent = 16 + depth * 14
    const hasChildren = visibleChildIds.length > 0

    return (
        <li>
            <div
                className={`DocSidebar__item ${isActive ? 'active' : ''}`}
                style={{
                    display: 'flex', alignItems: 'center',
                    padding: `4px 8px 4px ${indent}px`,
                    cursor: 'pointer', fontSize: 13,
                    background: isActive ? 'rgba(var(--button-bg-rgb), 0.12)' : 'transparent',
                    color: isActive ? 'rgba(var(--button-bg-rgb), 1)' : 'inherit',
                    borderRadius: 4, margin: '1px 6px',
                }}
            >
                <span
                    onClick={(e) => {
                        e.stopPropagation()
                        if (hasChildren) {
                            setExpanded(!expanded)
                        }
                    }}
                    style={{
                        width: 12, marginRight: 4,
                        cursor: hasChildren ? 'pointer' : 'default',
                        opacity: hasChildren ? 0.8 : 0.3,
                        userSelect: 'none', fontSize: 10,
                    }}
                >
                    {hasChildren ? (expanded ? '▾' : '▸') : '•'}
                </span>
                <span
                    onClick={() => onSelect(page.id)}
                    style={{flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}
                >
                    {page.icon ? `${page.icon} ` : ''}{page.title || '(Untitled)'}
                </span>
                <button
                    onClick={(e) => {
                        e.stopPropagation()
                        onAdd(page.id)
                    }}
                    title='Add sub-page'
                    style={{
                        background: 'transparent', border: 0, cursor: 'pointer',
                        opacity: 0.5, padding: '0 4px', color: 'inherit', fontSize: 12,
                    }}
                >
                    {'+'}
                </button>
                <PageMenu page={page} childCount={childIds.length} onAfterHide={onAfterHide}/>
            </div>
            {expanded && hasChildren && (
                <ul style={{listStyle: 'none', padding: 0, margin: 0}}>
                    {visibleChildIds.map((cid) => {
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
                                onAfterHide={onAfterHide}
                                hiddenIds={hiddenIds}
                            />
                        )
                    })}
                </ul>
            )}
        </li>
    )
}
