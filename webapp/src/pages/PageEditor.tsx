// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useMemo, useRef, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'
import {useEditor, EditorContent} from '@tiptap/react'
import {getSchema} from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import * as Y from 'yjs'
import {prosemirrorJSONToYDoc} from 'y-prosemirror'
import {Awareness, encodeAwarenessUpdate, applyAwarenessUpdate} from 'y-protocols/awareness'

import {fetchPageContent, getPageContent, getPage} from '../store/pages'
import {getMe} from '../store/users'
import client from '../octoClient'
import wsClient, {PageYjsMessage, PageYjsAwarenessMessage, WSClient} from '../wsclient'
import {useWebsockets} from '../hooks/websockets'
import type {AppDispatch} from '../store'
import type {TiptapDoc} from '../blocks/page'

import './PageEditor.scss'

// Pages feature — Tiptap editor host (Phase B+: live sync + awareness/cursors).
//
// Y.Doc is the source of truth for content. Awareness (y-protocols) carries
// ephemeral per-peer state — cursor position, selection, user metadata.
// Both ride the same WS relay pattern: outgoing as SEND_YJS_*, incoming as
// UPDATE_PAGE_YJS_*. REMOTE_ORIGIN tags applied updates so the local
// listener doesn't echo them back.

const SAVE_DEBOUNCE_MS = 800
const REMOTE_ORIGIN = Symbol('yjs-remote')

type Props = {
    pageId: string
}

function uint8ToB64(u8: Uint8Array): string {
    let s = ''
    for (let i = 0; i < u8.length; i++) {
        s += String.fromCharCode(u8[i])
    }
    return btoa(s)
}

function b64ToUint8(b64: string): Uint8Array {
    const bin = atob(b64)
    const u8 = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) {
        u8[i] = bin.charCodeAt(i)
    }
    return u8
}

// Deterministic-ish HSL color from a user id so the same user gets a stable
// cursor color across sessions.
function colorForUserId(userId: string): string {
    let h = 0
    for (let i = 0; i < userId.length; i++) {
        h = (h * 31 + userId.charCodeAt(i)) & 0xffff
    }
    return `hsl(${h % 360}, 70%, 50%)`
}

const editorSchema = getSchema([StarterKit.configure({history: false})])

export default function PageEditor({pageId}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const content = useSelector(getPageContent(pageId))
    const page = useSelector(getPage(pageId))
    const me = useSelector(getMe)
    const [loaded, setLoaded] = useState(false)
    const [saving, setSaving] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
    const saveTimer = useRef<number | null>(null)

    const clientId = useMemo(() => Math.random().toString(36).slice(2), [])

    const ydoc = useMemo(() => new Y.Doc(), [pageId])
    const awareness = useMemo(() => new Awareness(ydoc), [ydoc])

    useEffect(() => {
        setLoaded(false)
        dispatch(fetchPageContent(pageId)).finally(() => setLoaded(true))
    }, [pageId, dispatch])

    const bootstrapped = useRef(false)
    useEffect(() => {
        bootstrapped.current = false
    }, [pageId])

    useEffect(() => {
        if (!loaded || !content || bootstrapped.current) {
            return
        }
        bootstrapped.current = true
        if (content.yjsState && content.yjsState.length > 0) {
            try {
                Y.applyUpdate(ydoc, b64ToUint8(content.yjsState), REMOTE_ORIGIN)
                return
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] Y.applyUpdate failed; falling back to tiptapJson', e)
            }
        }
        if (content.tiptapJson) {
            try {
                const seeded = prosemirrorJSONToYDoc(editorSchema, content.tiptapJson)
                Y.applyUpdate(ydoc, Y.encodeStateAsUpdate(seeded), REMOTE_ORIGIN)
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] prosemirrorJSONToYDoc failed', e)
            }
        }
    }, [loaded, content, ydoc])

    // Live broadcast of local Y.Doc edits.
    useEffect(() => {
        if (!page) {
            return
        }
        const teamId = page.teamId
        const onLocalUpdate = (update: Uint8Array, origin: unknown) => {
            if (origin === REMOTE_ORIGIN) {
                return
            }
            try {
                wsClient.sendYjsUpdateCommand(teamId, pageId, uint8ToB64(update), clientId)
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] sendYjsUpdate failed', e)
            }
        }
        ydoc.on('update', onLocalUpdate)
        return () => {
            ydoc.off('update', onLocalUpdate)
        }
    }, [ydoc, pageId, page, clientId])

    // Live broadcast of local Awareness changes (cursor, user metadata).
    useEffect(() => {
        if (!page) {
            return
        }
        const teamId = page.teamId
        const onAwarenessUpdate = (
            {added, updated, removed}: {added: number[]; updated: number[]; removed: number[]},
            origin: unknown,
        ) => {
            if (origin === REMOTE_ORIGIN) {
                return
            }
            const changedClients = added.concat(updated, removed)
            if (changedClients.length === 0) {
                return
            }
            try {
                const update = encodeAwarenessUpdate(awareness, changedClients)
                wsClient.sendYjsAwarenessCommand(teamId, pageId, uint8ToB64(update), clientId)
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] sendYjsAwareness failed', e)
            }
        }
        awareness.on('update', onAwarenessUpdate)
        return () => {
            awareness.off('update', onAwarenessUpdate)
        }
    }, [awareness, pageId, page, clientId])

    // Live receive Y.Doc + Awareness updates from peers (also issues
    // SUBSCRIBE_TEAM via useWebsockets).
    useWebsockets(page?.teamId || '', (ws: WSClient) => {
        const yjsHandler = (msg: PageYjsMessage) => {
            if (msg.pageId !== pageId) {
                return
            }
            if (msg.originClientId && msg.originClientId === clientId) {
                return
            }
            try {
                Y.applyUpdate(ydoc, b64ToUint8(msg.updateB64), REMOTE_ORIGIN)
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] applyUpdate from peer failed', e)
            }
        }
        const awarenessHandler = (msg: PageYjsAwarenessMessage) => {
            if (msg.pageId !== pageId) {
                return
            }
            if (msg.originClientId && msg.originClientId === clientId) {
                return
            }
            try {
                applyAwarenessUpdate(awareness, b64ToUint8(msg.awarenessB64), REMOTE_ORIGIN)
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] applyAwarenessUpdate from peer failed', e)
            }
        }
        ws.addPageYjsHandler(yjsHandler)
        ws.addPageYjsAwarenessHandler(awarenessHandler)
        return () => {
            ws.removePageYjsHandler(yjsHandler)
            ws.removePageYjsAwarenessHandler(awarenessHandler)
        }
    }, [ydoc, awareness, pageId, clientId])

    // Set local awareness user info so peers know who we are.
    useEffect(() => {
        if (!me) {
            return
        }
        const name = me.username || me.nickname || me.firstname || me.id
        const color = colorForUserId(me.id)
        awareness.setLocalStateField('user', {name, color})
        return () => {
            // Clear local state on unmount so peers stop showing our cursor.
            awareness.setLocalState(null)
        }
    }, [awareness, me])

    const editor = useEditor({
        extensions: [
            StarterKit.configure({history: false}),
            Collaboration.configure({document: ydoc}),
            CollaborationCursor.configure({
                provider: {awareness},
                user: me ? {
                    name: me.username || me.nickname || me.firstname || me.id,
                    color: colorForUserId(me.id),
                } : {name: 'Anonymous', color: '#888'},
            }),
        ],
        onUpdate: ({editor: e}) => {
            if (saveTimer.current) {
                window.clearTimeout(saveTimer.current)
            }
            setSaving('saving')
            saveTimer.current = window.setTimeout(async () => {
                try {
                    const stateU8 = Y.encodeStateAsUpdate(ydoc)
                    const stateB64 = uint8ToB64(stateU8)
                    const tiptapJson = e.getJSON() as unknown as TiptapDoc
                    const resp = await client.saveYjsSnapshot(pageId, stateB64, tiptapJson)
                    if (!resp.ok) {
                        setSaving('error')
                        return
                    }
                    setSaving('saved')
                    window.setTimeout(() => setSaving('idle'), 1500)
                } catch {
                    setSaving('error')
                }
            }, SAVE_DEBOUNCE_MS) as unknown as number
        },
    }, [pageId, ydoc, awareness, me?.id])

    if (!loaded) {
        return <div className='PageEditor PageEditor--loading'>{'Loading…'}</div>
    }

    return (
        <div className='PageEditor'>
            <div
                className='PageEditor__status'
                style={{fontSize: 11, color: '#6b7280', marginBottom: 4, height: 14}}
            >
                {saving === 'saving' && 'Saving…'}
                {saving === 'saved' && 'Saved'}
                {saving === 'error' && 'Save failed'}
            </div>
            <EditorContent editor={editor}/>
        </div>
    )
}
