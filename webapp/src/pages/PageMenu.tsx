// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useMemo, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'

import Dialog from '../components/dialog'
import Button from '../widgets/buttons/button'

import IconButton from '../widgets/buttons/iconButton'
import OptionsIcon from '../widgets/icons/options'
import DeleteIcon from '../widgets/icons/delete'
import DuplicateIcon from '../widgets/icons/duplicate'
import CloseIcon from '../widgets/icons/close'
import AddIcon from '../widgets/icons/add'
import CompassIcon from '../widgets/icons/compassIcon'
import CreateNewFolder from '../widgets/icons/newFolder'
import Menu from '../widgets/menu'
import MenuWrapper from '../widgets/menuWrapper'

import client from '../octoClient'
import {deletePage, fetchPagesForTeam, getPagesById, upsertPage} from '../store/pages'
import type {AppDispatch} from '../store'
import type {Page} from '../blocks/page'

const windowAny = (window as unknown) as {frontendBaseURL?: string}

// PageMenu — hover-revealed actions menu on a page tree row.
// Mirrors SidebarBoardItem's per-board menu: Move / Duplicate /
// New template / Export / Hide / Delete.
//
// MVP: Delete fully wired (with cascade vs promote confirmation).
// Export Markdown is wired (client-side conversion).
// Hide is local-only via localStorage. Other items are placeholders
// that surface "Coming soon" alerts until backend support lands.

const HIDDEN_KEY = 'focalboard:hiddenPageIds'

export function getHiddenPageIds(): Set<string> {
    try {
        return new Set(JSON.parse(localStorage.getItem(HIDDEN_KEY) || '[]'))
    } catch {
        return new Set()
    }
}

function setHiddenPageIds(ids: Set<string>) {
    try {
        localStorage.setItem(HIDDEN_KEY, JSON.stringify(Array.from(ids)))
    } catch { /* ignore */ }
}

type Props = {
    page: Page
    childCount: number
    onAfterDelete?: () => void
    onAfterHide?: () => void
}

export default function PageMenu({page, childCount, onAfterDelete, onAfterHide}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const pagesById = useSelector(getPagesById)
    const [showDelete, setShowDelete] = useState(false)
    const [cascade, setCascade] = useState<boolean>(false) // default: promote children
    const [busy, setBusy] = useState(false)
    const [showMove, setShowMove] = useState(false)
    const [duplicateBusy, setDuplicateBusy] = useState(false)

    const onDeleteConfirm = useCallback(async () => {
        setBusy(true)
        try {
            await dispatch(deletePage({pageId: page.id, cascade}))
            setShowDelete(false)
            onAfterDelete?.()
        } finally {
            setBusy(false)
        }
    }, [dispatch, page.id, cascade, onAfterDelete])

    const onHide = useCallback(() => {
        const ids = getHiddenPageIds()
        ids.add(page.id)
        setHiddenPageIds(ids)
        onAfterHide?.()
    }, [page.id, onAfterHide])

    const onExportMarkdown = useCallback(async () => {
        // Fetch tiptap JSON, do a minimal conversion to markdown, download.
        try {
            const resp = await client.getPageContent(page.id)
            if (!resp.ok) {
                alert('Failed to load page content.')
                return
            }
            const c = await resp.json()
            const md = tiptapToMarkdown(c.tiptapJson) || '(empty)'
            const fname = (page.title || 'untitled').replace(/[^\w\-가-힣 .]/g, '').slice(0, 50) + '.md'
            const blob = new Blob([`# ${page.title || '(Untitled)'}\n\n${md}`], {type: 'text/markdown;charset=utf-8'})
            const a = document.createElement('a')
            a.href = URL.createObjectURL(blob)
            a.download = fname
            a.click()
            setTimeout(() => URL.revokeObjectURL(a.href), 1000)
        } catch (e) {
            alert('Export failed: ' + String(e))
        }
    }, [page.id, page.title])

    const comingSoon = (label: string) => () => alert(`${label}: coming soon.`)

    // Move flow — show dialog with team page tree (excluding self+descendants).
    const onMove = useCallback(async (newParentID: string) => {
        setBusy(true)
        try {
            const resp = await client.movePage(page.id, newParentID, page.sortOrder)
            if (!resp.ok) {
                const txt = await resp.text()
                alert(`Move failed: ${txt}`)
                return
            }
            // Update local state — patch the page's parentId
            dispatch(upsertPage({...page, parentId: newParentID}))
            // Reload to ensure consistency
            dispatch(fetchPagesForTeam(page.teamId))
            setShowMove(false)
        } finally {
            setBusy(false)
        }
    }, [dispatch, page])

    // Build list of pages we cannot move under (self + descendants).
    const forbidden = useMemo(() => {
        const set = new Set<string>([page.id])
        const queue = [page.id]
        while (queue.length) {
            const cur = queue.shift()!
            for (const p of Object.values(pagesById)) {
                if (p.parentId === cur && !set.has(p.id)) {
                    set.add(p.id)
                    queue.push(p.id)
                }
            }
        }
        return set
    }, [page.id, pagesById])

    const moveCandidates = useMemo(() => {
        return Object.values(pagesById)
            .filter((p) => p.teamId === page.teamId && !forbidden.has(p.id) && p.deleteAt === 0)
            .sort((a, b) => (a.title || '').localeCompare(b.title || ''))
    }, [pagesById, page.teamId, forbidden])

    const onDuplicate = useCallback(async () => {
        setDuplicateBusy(true)
        try {
            const resp = await client.duplicatePage(page.id, true)
            if (!resp.ok) {
                const txt = await resp.text()
                alert(`Duplicate failed: ${txt}`)
                return
            }
            const json = await resp.json() as {id: string}
            dispatch(fetchPagesForTeam(page.teamId))
            const base = windowAny.frontendBaseURL || '/boards'
            window.location.href = `${base}/team/${page.teamId}/pages/${json.id}`
        } finally {
            setDuplicateBusy(false)
        }
    }, [dispatch, page])

    return (
        <>
            <MenuWrapper stopPropagationOnToggle={true}>
                <IconButton size='small' icon={<OptionsIcon/>}/>
                <Menu position='auto'>
                    <Menu.Text
                        id='movePage'
                        name={'Move to…'}
                        icon={<CreateNewFolder/>}
                        onClick={() => setShowMove(true)}
                    />
                    <Menu.Text
                        id='duplicatePage'
                        name={duplicateBusy ? 'Duplicating…' : 'Duplicate page'}
                        icon={<DuplicateIcon/>}
                        onClick={onDuplicate}
                    />
                    <Menu.Text
                        id='templateFromPage'
                        name={'New template from page'}
                        icon={<AddIcon/>}
                        onClick={comingSoon('New template from page')}
                    />
                    <Menu.Text
                        id='exportPageMd'
                        name={'Export as Markdown'}
                        icon={<CompassIcon icon='export-variant'/>}
                        onClick={onExportMarkdown}
                    />
                    <Menu.Text
                        id='hidePage'
                        name={'Hide page'}
                        icon={<CloseIcon/>}
                        onClick={onHide}
                    />
                    <Menu.Text
                        id='deletePage'
                        className='text-danger'
                        name={'Delete page'}
                        icon={<DeleteIcon/>}
                        onClick={() => setShowDelete(true)}
                    />
                </Menu>
            </MenuWrapper>

            {showMove && (
                <Dialog
                    size='small'
                    className='confirmation-dialog-box'
                    onClose={() => !busy && setShowMove(false)}
                >
                    <div className='box-area'>
                        <h3 className='text-heading5'>{`Move "${page.title || '(Untitled)'}"`}</h3>
                        <div style={{margin: '8px 0', fontSize: 13, color: 'rgba(var(--center-channel-color-rgb),0.72)'}}>
                            {'Pick a new parent (or move to the top level):'}
                        </div>
                        <div style={{maxHeight: 280, overflowY: 'auto', border: '1px solid rgba(var(--center-channel-color-rgb),0.12)', borderRadius: 4}}>
                            <button
                                onClick={() => onMove('')}
                                disabled={busy}
                                style={{
                                    display: 'flex', alignItems: 'center', width: '100%',
                                    padding: '10px 14px', background: 'transparent', border: 0,
                                    borderBottom: '1px solid rgba(var(--center-channel-color-rgb),0.08)',
                                    textAlign: 'left', cursor: 'pointer', fontSize: 14,
                                    color: 'inherit',
                                }}
                            >
                                <span style={{marginRight: 10, fontSize: 16}}>{'/'}</span>
                                <span style={{fontWeight: 600}}>{'(Top level)'}</span>
                            </button>
                            {moveCandidates.length === 0 && (
                                <div style={{padding: 16, fontSize: 13, color: '#9ca3af'}}>
                                    {'No other pages to move under.'}
                                </div>
                            )}
                            {moveCandidates.map((p) => (
                                <button
                                    key={p.id}
                                    onClick={() => onMove(p.id)}
                                    disabled={busy}
                                    style={{
                                        display: 'flex', alignItems: 'center', width: '100%',
                                        padding: '8px 14px', background: 'transparent', border: 0,
                                        borderTop: '1px solid rgba(var(--center-channel-color-rgb),0.04)',
                                        textAlign: 'left', cursor: 'pointer', fontSize: 13,
                                        color: 'inherit',
                                    }}
                                >
                                    <span style={{marginRight: 10}}>{p.icon || '📄'}</span>
                                    <span style={{flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                                        {p.title || '(Untitled)'}
                                    </span>
                                </button>
                            ))}
                        </div>
                        <div className='action-buttons' style={{display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 12}}>
                            <Button onClick={() => setShowMove(false)} emphasis='secondary' size='medium'>
                                {'Cancel'}
                            </Button>
                        </div>
                    </div>
                </Dialog>
            )}

            {showDelete && (
                <Dialog
                    size='small'
                    className='confirmation-dialog-box'
                    onClose={() => !busy && setShowDelete(false)}
                >
                    <div className='box-area'>
                        <h3 className='text-heading5'>{'Delete page'}</h3>
                        <div className='sub-text' style={{margin: '8px 0 16px', fontSize: 13}}>
                            {`Are you sure you want to delete "${page.title || '(Untitled)'}"?`}
                        </div>
                        {childCount > 0 && (
                            <div style={{margin: '0 0 16px', fontSize: 13}}>
                                <p style={{margin: '0 0 8px'}}>
                                    {`This page has ${childCount} sub-page${childCount > 1 ? 's' : ''}.`}
                                </p>
                                <label style={{display: 'flex', alignItems: 'center', cursor: 'pointer', padding: '4px 0'}}>
                                    <input
                                        type='radio'
                                        name='cascade'
                                        checked={!cascade}
                                        onChange={() => setCascade(false)}
                                        style={{marginRight: 8}}
                                    />
                                    {'Move sub-pages to the parent'}
                                </label>
                                <label style={{display: 'flex', alignItems: 'center', cursor: 'pointer', padding: '4px 0'}}>
                                    <input
                                        type='radio'
                                        name='cascade'
                                        checked={cascade}
                                        onChange={() => setCascade(true)}
                                        style={{marginRight: 8}}
                                    />
                                    {'Delete sub-pages too'}
                                </label>
                            </div>
                        )}
                        <div className='action-buttons' style={{display: 'flex', gap: 8, justifyContent: 'flex-end'}}>
                            <Button
                                onClick={() => setShowDelete(false)}
                                emphasis='secondary'
                                size='medium'
                            >
                                {'Cancel'}
                            </Button>
                            <Button
                                onClick={onDeleteConfirm}
                                emphasis='danger'
                                size='medium'
                            >
                                {busy ? 'Deleting…' : 'Delete'}
                            </Button>
                        </div>
                    </div>
                </Dialog>
            )}
        </>
    )
}

// Minimal Tiptap JSON → Markdown conversion. Covers heading, paragraph,
// bullet/ordered list, code block, blockquote, horizontal rule, hard break,
// and inline marks (bold, italic, strike, code, link). Anything unknown
// renders as plain text.
function tiptapToMarkdown(node: unknown): string {
    if (!node || typeof node !== 'object') {
        return ''
    }
    const n = node as {type?: string; text?: string; content?: unknown[]; attrs?: Record<string, unknown>; marks?: Array<{type: string; attrs?: Record<string, unknown>}>}

    const renderInline = (text: string, marks?: Array<{type: string; attrs?: Record<string, unknown>}>): string => {
        if (!marks || !marks.length) {
            return text
        }
        let out = text
        for (const m of marks) {
            if (m.type === 'bold') {
                out = `**${out}**`
            } else if (m.type === 'italic') {
                out = `*${out}*`
            } else if (m.type === 'strike') {
                out = `~~${out}~~`
            } else if (m.type === 'code') {
                out = `\`${out}\``
            } else if (m.type === 'link') {
                const href = m.attrs?.href as string | undefined
                out = `[${out}](${href || ''})`
            }
        }
        return out
    }

    const renderChildren = (children?: unknown[]): string => {
        if (!children) {
            return ''
        }
        return children.map((c) => tiptapToMarkdown(c)).join('')
    }

    if (n.type === 'text') {
        return renderInline(n.text || '', n.marks)
    }
    if (n.type === 'doc') {
        return renderChildren(n.content)
    }
    if (n.type === 'heading') {
        const level = (n.attrs?.level as number) || 1
        return '#'.repeat(level) + ' ' + renderChildren(n.content) + '\n\n'
    }
    if (n.type === 'paragraph') {
        return renderChildren(n.content) + '\n\n'
    }
    if (n.type === 'bulletList') {
        return (n.content || []).map((c) => '- ' + (tiptapToMarkdown(c).trimEnd())).join('\n') + '\n\n'
    }
    if (n.type === 'orderedList') {
        return (n.content || []).map((c, i) => `${i + 1}. ` + (tiptapToMarkdown(c).trimEnd())).join('\n') + '\n\n'
    }
    if (n.type === 'listItem') {
        return renderChildren(n.content).trim()
    }
    if (n.type === 'codeBlock') {
        const lang = (n.attrs?.language as string) || ''
        return '```' + lang + '\n' + renderChildren(n.content) + '\n```\n\n'
    }
    if (n.type === 'blockquote') {
        return renderChildren(n.content).split('\n').map((l) => '> ' + l).join('\n') + '\n\n'
    }
    if (n.type === 'horizontalRule') {
        return '\n---\n\n'
    }
    if (n.type === 'hardBreak') {
        return '\n'
    }
    return renderChildren(n.content)
}
