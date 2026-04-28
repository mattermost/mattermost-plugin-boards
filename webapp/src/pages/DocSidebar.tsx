// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useMemo, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'
import {DragDropContext, Droppable, Draggable, DropResult} from 'react-beautiful-dnd'

import type {Page, PageCategory} from '../blocks/page'
import {fetchPagesForTeam, getChildPageIds, getPagesById, createPage, setCurrentPage} from '../store/pages'
import {
    fetchPageCategories,
    fetchPageCategoryAssignments,
    createPageCategory,
    deletePageCategory,
    upsertPageCategory,
    getPageCategories,
    getPageCategoryAssignments,
    setPageCategory,
    unsetPageCategory,
    setPageCategoryLocal,
} from '../store/pageCategories'
import wsClient, {PageCategoryMessage, PageCategoryAssignMessage} from '../wsclient'
import type {AppDispatch} from '../store'

import SearchIcon from '../widgets/icons/search'
import AddIcon from '../widgets/icons/add'
import IconButton from '../widgets/buttons/iconButton'
import ChevronDown from '../widgets/icons/chevronDown'
import ChevronRight from '../widgets/icons/chevronRight'
import OptionsIcon from '../widgets/icons/options'
import DeleteIcon from '../widgets/icons/delete'
import EditIcon from '../widgets/icons/edit'
import CompassIcon from '../widgets/icons/compassIcon'
import Menu from '../widgets/menu'
import MenuWrapper from '../widgets/menuWrapper'

import PageMenu, {getHiddenPageIds} from './PageMenu'
import CreatePageCategoryModal from './CreatePageCategoryModal'

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

// Special id for the default uncategorized droppable — must be a string
// (react-beautiful-dnd requires non-empty droppableId).
const UNCATEGORIZED_DROPPABLE = '__uncategorized__'

export default function DocSidebar({teamId, activePageId, onSelect}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const pagesById = useSelector(getPagesById)
    const rootIds = useSelector(getChildPageIds(''))
    const categories = useSelector(getPageCategories)
    const assignmentByPageId = useSelector(getPageCategoryAssignments)
    const [query, setQuery] = useState<string>('')
    const [categoryCollapsed, setCategoryCollapsed] = useState<boolean>(false)
    // Hidden page IDs (localStorage). bumped on hide/unhide to re-render.
    const [hiddenTick, setHiddenTick] = useState(0)
    const hiddenIds = useMemo(() => getHiddenPageIds(), [hiddenTick])
    const onAfterHide = useCallback(() => setHiddenTick((t) => t + 1), [])
    const visibleRootIds = useMemo(() => rootIds.filter((id) => !hiddenIds.has(id)), [rootIds, hiddenIds])

    // Group visible top-level pages by their assigned category. Pages
    // without an assignment fall through to the uncategorized bucket.
    const pagesByDroppable = useMemo(() => {
        const groups: {[droppableId: string]: string[]} = {[UNCATEGORIZED_DROPPABLE]: []}
        for (const cat of categories) {
            groups[cat.id] = []
        }
        for (const id of visibleRootIds) {
            const catId = assignmentByPageId[id]
            if (catId && groups[catId]) {
                groups[catId].push(id)
            } else {
                groups[UNCATEGORIZED_DROPPABLE].push(id)
            }
        }
        return groups
    }, [visibleRootIds, categories, assignmentByPageId])

    const onDragEnd = useCallback(async (result: DropResult) => {
        const {destination, source, draggableId} = result
        if (!destination) {
            return
        }
        if (destination.droppableId === source.droppableId && destination.index === source.index) {
            return
        }
        const fromCat = source.droppableId
        const toCat = destination.droppableId
        const pageId = draggableId

        // Optimistic local update before WS confirms.
        if (toCat === UNCATEGORIZED_DROPPABLE) {
            dispatch(setPageCategoryLocal({pageId, categoryId: ''}))
        } else {
            dispatch(setPageCategoryLocal({pageId, categoryId: toCat}))
        }

        if (toCat === UNCATEGORIZED_DROPPABLE) {
            const r = await dispatch(unsetPageCategory({teamId, pageId}))
            if (unsetPageCategory.rejected.match(r)) {
                // rollback
                dispatch(setPageCategoryLocal({pageId, categoryId: fromCat === UNCATEGORIZED_DROPPABLE ? '' : fromCat}))
            }
        } else {
            const r = await dispatch(setPageCategory({teamId, categoryId: toCat, pageId, sortOrder: destination.index}))
            if (setPageCategory.rejected.match(r)) {
                dispatch(setPageCategoryLocal({pageId, categoryId: fromCat === UNCATEGORIZED_DROPPABLE ? '' : fromCat}))
            }
        }
    }, [dispatch, teamId])

    useEffect(() => {
        if (teamId) {
            dispatch(fetchPagesForTeam(teamId))
            dispatch(fetchPageCategories(teamId))
            dispatch(fetchPageCategoryAssignments(teamId))
        }
    }, [teamId, dispatch])

    // Live updates for own categories (per-user broadcast).
    useEffect(() => {
        const handler = (msg: PageCategoryMessage) => {
            if (msg.category) {
                dispatch(upsertPageCategory(msg.category as PageCategory))
            }
        }
        const assignHandler = (msg: PageCategoryAssignMessage) => {
            dispatch(setPageCategoryLocal({pageId: msg.pageId, categoryId: msg.categoryId || ''}))
        }
        wsClient.addPageCategoryHandler(handler)
        wsClient.addPageCategoryAssignHandler(assignHandler)
        return () => {
            wsClient.removePageCategoryHandler(handler)
            wsClient.removePageCategoryAssignHandler(assignHandler)
        }
    }, [dispatch])

    // Direct create — Boards' mutator pattern: click → POST, then the
    // server's WS broadcast adds the row to state. No modal.
    const onAddCategory = useCallback(() => {
        void dispatch(createPageCategory({teamId, name: '새 카테고리'}))
    }, [dispatch, teamId])

    // Rename still uses the modal since it needs an input field.
    const [renameTarget, setRenameTarget] = useState<PageCategory | null>(null)
    const onRenameCategory = useCallback((cat: PageCategory) => {
        setRenameTarget(cat)
    }, [])

    const onDeleteCategory = useCallback(async (cat: PageCategory) => {
        // eslint-disable-next-line no-alert
        if (!window.confirm(`"${cat.name}" 카테고리를 삭제할까요?`)) {
            return
        }
        await dispatch(deletePageCategory({teamId, categoryId: cat.id}))
    }, [dispatch, teamId])

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
                <MenuWrapper>
                    <IconButton
                        size='small'
                        inverted={true}
                        className='add-board-icon'
                        icon={<AddIcon/>}
                        title='추가'
                    />
                    <Menu position='auto'>
                        <Menu.Text
                            id='createNewPage'
                            name={'새 페이지 만들기'}
                            icon={<CompassIcon icon='plus'/>}
                            onClick={() => onAdd('')}
                        />
                        <Menu.Text
                            id='createNewCategory'
                            name={'새 카테고리 만들기'}
                            icon={<CompassIcon icon='folder-plus-outline'/>}
                            onClick={onAddCategory}
                        />
                    </Menu>
                </MenuWrapper>
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
            ) : null}
            {renameTarget && (
                <CreatePageCategoryModal
                    teamId={teamId}
                    categoryId={renameTarget.id}
                    initialValue={renameTarget.name}
                    onClose={() => setRenameTarget(null)}
                />
            )}
            {!isSearching && (
                <DragDropContext onDragEnd={onDragEnd}>
                    <div className='octo-sidebar-list'>
                        {/* User-defined categories with droppable bodies. */}
                        {categories.map((cat) => (
                            <SidebarPageCategory
                                key={cat.id}
                                category={cat}
                                pageIds={pagesByDroppable[cat.id] || []}
                                pagesById={pagesById}
                                activePageId={activePageId}
                                onSelect={onSelect}
                                onAdd={onAdd}
                                onAfterHide={onAfterHide}
                                hiddenIds={hiddenIds}
                                onRename={onRenameCategory}
                                onDelete={onDeleteCategory}
                            />
                        ))}

                        {/* Default uncategorized droppable section. */}
                        <div className={`octo-sidebar-item category ${categoryCollapsed ? 'collapsed' : 'expanded'}`}>
                            <div
                                className='octo-sidebar-title category-title'
                                title='Pages'
                                onClick={() => setCategoryCollapsed(!categoryCollapsed)}
                                style={{flex: 1, cursor: 'pointer'}}
                            >
                                {categoryCollapsed ? <ChevronRight/> : <ChevronDown/>}
                                {'Pages'}
                            </div>
                        </div>
                        {!categoryCollapsed && (
                            <Droppable droppableId={UNCATEGORIZED_DROPPABLE} type='page'>
                                {(provided) => (
                                    <div
                                        ref={provided.innerRef}
                                        {...provided.droppableProps}
                                        className='DocSidebar__tree'
                                        style={{minHeight: 4}}
                                    >
                                        {(pagesByDroppable[UNCATEGORIZED_DROPPABLE] || []).map((id, idx) => {
                                            const page = pagesById[id]
                                            if (!page) {
                                                return null
                                            }
                                            return (
                                                <Draggable key={id} draggableId={id} index={idx}>
                                                    {(p) => (
                                                        <div
                                                            ref={p.innerRef}
                                                            {...p.draggableProps}
                                                            {...p.dragHandleProps}
                                                        >
                                                            <PageTreeNode
                                                                page={page}
                                                                depth={0}
                                                                activePageId={activePageId}
                                                                onSelect={onSelect}
                                                                onAdd={onAdd}
                                                                onAfterHide={onAfterHide}
                                                                hiddenIds={hiddenIds}
                                                            />
                                                        </div>
                                                    )}
                                                </Draggable>
                                            )
                                        })}
                                        {provided.placeholder}
                                        {(pagesByDroppable[UNCATEGORIZED_DROPPABLE] || []).length === 0 && (
                                            <div style={{padding: '8px 16px', color: '#9ca3af', fontSize: 12}}>
                                                {rootIds.length === 0 ? 'No pages yet. Click + above to create one.' : '비어있음 — 페이지를 여기로 드래그하세요.'}
                                            </div>
                                        )}
                                    </div>
                                )}
                            </Droppable>
                        )}
                    </div>
                </DragDropContext>
            )}
        </div>
    )
}

// SidebarPageCategory — collapsible row for a user-created category with
// a Droppable body. Pages assigned to this category render here as
// Draggables. Mirrors Boards' SidebarCategory.
type SidebarPageCategoryProps = {
    category: PageCategory
    pageIds: string[]
    pagesById: {[id: string]: Page}
    activePageId?: string
    onSelect: (pageId: string) => void
    onAdd: (parentId: string) => void
    onAfterHide?: () => void
    hiddenIds: Set<string>
    onRename: (cat: PageCategory) => void
    onDelete: (cat: PageCategory) => void
}

function SidebarPageCategory(props: SidebarPageCategoryProps): JSX.Element {
    const {category, pageIds, pagesById, activePageId, onSelect, onAdd, onAfterHide, hiddenIds, onRename, onDelete} = props
    const [collapsed, setCollapsed] = useState<boolean>(category.collapsed)
    return (
        <>
            <div className={`octo-sidebar-item category ${collapsed ? 'collapsed' : 'expanded'}`}>
                <div
                    className='octo-sidebar-title category-title'
                    onClick={() => setCollapsed(!collapsed)}
                    style={{flex: 1, display: 'flex', alignItems: 'center', cursor: 'pointer'}}
                >
                    {collapsed ? <ChevronRight/> : <ChevronDown/>}
                    {category.name}
                </div>
                <MenuWrapper stopPropagationOnToggle={true}>
                    <IconButton size='small' icon={<OptionsIcon/>}/>
                    <Menu position='auto'>
                        <Menu.Text
                            id='renamePageCategory'
                            name={'이름 변경'}
                            icon={<EditIcon/>}
                            onClick={() => onRename(category)}
                        />
                        <Menu.Text
                            id='deletePageCategory'
                            className='text-danger'
                            name={'카테고리 삭제'}
                            icon={<DeleteIcon/>}
                            onClick={() => onDelete(category)}
                        />
                    </Menu>
                </MenuWrapper>
            </div>
            {!collapsed && (
                <Droppable droppableId={category.id} type='page'>
                    {(provided) => (
                        <div
                            ref={provided.innerRef}
                            {...provided.droppableProps}
                            style={{minHeight: 4}}
                        >
                            {pageIds.map((id, idx) => {
                                const page = pagesById[id]
                                if (!page) {
                                    return null
                                }
                                return (
                                    <Draggable key={id} draggableId={id} index={idx}>
                                        {(p) => (
                                            <div
                                                ref={p.innerRef}
                                                {...p.draggableProps}
                                                {...p.dragHandleProps}
                                            >
                                                <PageTreeNode
                                                    page={page}
                                                    depth={0}
                                                    activePageId={activePageId}
                                                    onSelect={onSelect}
                                                    onAdd={onAdd}
                                                    onAfterHide={onAfterHide}
                                                    hiddenIds={hiddenIds}
                                                />
                                            </div>
                                        )}
                                    </Draggable>
                                )
                            })}
                            {provided.placeholder}
                            {pageIds.length === 0 && (
                                <div style={{padding: '6px 28px', color: 'rgba(var(--center-channel-color-rgb), 0.5)', fontSize: 11, fontStyle: 'italic'}}>
                                    {'페이지를 여기로 드래그하세요'}
                                </div>
                            )}
                        </div>
                    )}
                </Droppable>
            )}
        </>
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
        <div>
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
                <div>
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
                </div>
            )}
        </div>
    )
}
