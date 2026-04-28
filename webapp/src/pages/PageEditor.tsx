// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'
import {useEditor, EditorContent, ReactNodeViewRenderer} from '@tiptap/react'
import {getSchema} from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import Image from '@tiptap/extension-image'
import * as Y from 'yjs'
import {prosemirrorJSONToYDoc, yDocToProsemirrorJSON} from 'y-prosemirror'
import {Awareness, encodeAwarenessUpdate, applyAwarenessUpdate} from 'y-protocols/awareness'

import {fetchPageContent, getPageContent, getPage} from '../store/pages'
import {getMe} from '../store/users'
import client from '../octoClient'
import wsClient, {PageYjsMessage, PageYjsAwarenessMessage, WSClient} from '../wsclient'
import {useWebsockets} from '../hooks/websockets'
import type {AppDispatch} from '../store'

import PageImageNodeView from './PageImageNodeView'

import './PageEditor.scss'

// Tiptap Image with a custom React NodeView. The default <img src=URL>
// doesn't work because Mattermost's CSRF middleware rejects cookie-only
// GET requests; the NodeView fetches via XHR (with X-Requested-With) and
// renders from a Blob URL instead.
const PageImage = Image.extend({
    addNodeView() {
        return ReactNodeViewRenderer(PageImageNodeView)
    },
})

// Pages feature — Tiptap editor host (Phase C: snapshot lifecycle).
//
// Y.Doc is the source of truth. Three independent loops:
//  1. Live relay (Phase B): every Y.Doc / Awareness update goes out as
//     SEND_YJS_*; peers' updates arrive as UPDATE_PAGE_YJS_* and are
//     applied with REMOTE_ORIGIN to skip echo.
//  2. Snapshot persistence (Phase C): ANY Y.Doc change — local or remote —
//     queues a debounced save. Every active client persists, so if the
//     editing user navigates away the latest state still hits disk.
//  3. Lifecycle hooks (Phase C): visibilitychange (hidden) flushes
//     pending snapshot; pagehide flushes via fetch keepalive (best-effort);
//     WS reconnect refetches the server snapshot to merge anything missed
//     during the disconnect window.

const SAVE_DEBOUNCE_MS = 2000
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

    // ─── Snapshot persistence (Phase C) ────────────────────────────────
    const saveTimer = useRef<number | null>(null)
    const dirtyRef = useRef(false)

    const flushSnapshot = useCallback(async (opts?: {keepalive?: boolean}) => {
        if (!dirtyRef.current || !bootstrapped.current) {
            return
        }
        dirtyRef.current = false
        if (saveTimer.current) {
            window.clearTimeout(saveTimer.current)
            saveTimer.current = null
        }
        try {
            const stateU8 = Y.encodeStateAsUpdate(ydoc)
            if (stateU8.length === 0) {
                return
            }
            const stateB64 = uint8ToB64(stateU8)
            const tiptapJson = yDocToProsemirrorJSON(ydoc) as unknown
            setSaving('saving')
            const resp = await client.saveYjsSnapshot(pageId, stateB64, tiptapJson, opts)
            if (!resp.ok) {
                setSaving('error')
                return
            }
            setSaving('saved')
            window.setTimeout(() => setSaving('idle'), 1500)
        } catch {
            setSaving('error')
        }
    }, [pageId, ydoc])

    const queueSnapshot = useCallback(() => {
        dirtyRef.current = true
        if (saveTimer.current) {
            window.clearTimeout(saveTimer.current)
        }
        setSaving('saving')
        saveTimer.current = window.setTimeout(() => flushSnapshot(), SAVE_DEBOUNCE_MS) as unknown as number
    }, [flushSnapshot])

    // Snapshot trigger: fires for ALL Y.Doc updates (local AND remote).
    // Every active client persists so whoever's still around when the
    // editor closes guarantees the latest state has hit disk.
    useEffect(() => {
        const onAny = () => queueSnapshot()
        ydoc.on('update', onAny)
        return () => {
            ydoc.off('update', onAny)
        }
    }, [ydoc, queueSnapshot])

    // visibilitychange / pagehide → flush. visibilitychange has time to
    // await; pagehide uses fetch keepalive for best-effort delivery.
    useEffect(() => {
        const onVisibility = () => {
            if (document.visibilityState === 'hidden') {
                void flushSnapshot()
            }
        }
        const onPageHide = () => {
            void flushSnapshot({keepalive: true})
        }
        document.addEventListener('visibilitychange', onVisibility)
        window.addEventListener('pagehide', onPageHide)
        return () => {
            document.removeEventListener('visibilitychange', onVisibility)
            window.removeEventListener('pagehide', onPageHide)
        }
    }, [flushSnapshot])

    // ─── Live broadcast — Y.Doc edits ──────────────────────────────────
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

    // ─── Live broadcast — Awareness changes ────────────────────────────
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

    // ─── Live receive + SUBSCRIBE_TEAM ─────────────────────────────────
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

    // ─── Reconnect refetch (Phase C) ───────────────────────────────────
    // After a WS reconnect, GET the server snapshot and merge it. Y CRDT
    // handles convergence — applying any state vector is safe.
    useEffect(() => {
        const onReconnect = async () => {
            try {
                const resp = await client.getPageContent(pageId)
                if (!resp.ok) {
                    return
                }
                const json = await resp.json() as {yjsState?: string}
                if (json.yjsState) {
                    Y.applyUpdate(ydoc, b64ToUint8(json.yjsState), REMOTE_ORIGIN)
                }
            } catch (e) {
                // eslint-disable-next-line no-console
                console.warn('[PageEditor] reconnect refetch failed', e)
            }
        }
        wsClient.addOnReconnect(onReconnect)
        return () => {
            wsClient.removeOnReconnect(onReconnect)
        }
    }, [pageId, ydoc])

    // Set local awareness user info so peers know who we are.
    useEffect(() => {
        if (!me) {
            return
        }
        const name = me.username || me.nickname || me.firstname || me.id
        const color = colorForUserId(me.id)
        awareness.setLocalStateField('user', {name, color})
        return () => {
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
            PageImage.configure({
                inline: false,
                HTMLAttributes: {class: 'PageEditor__image'},
            }),
        ],
    }, [pageId, ydoc, awareness, me?.id])

    // Image upload + insert. Paste/drop handlers attach to the editor DOM
    // node directly via useEffect — putting them in editorProps caused the
    // editor to recreate whenever any closure dep changed, which lost
    // selection state and made debugging hard.
    const teamIdForUpload = page?.teamId
    const uploadAndInsert = useCallback(async (file: File) => {
        if (!teamIdForUpload) {
            // eslint-disable-next-line no-console
            console.warn('[PageEditor] uploadAndInsert: missing teamId')
            return
        }
        if (!file.type.startsWith('image/')) {
            // eslint-disable-next-line no-console
            console.warn('[PageEditor] uploadAndInsert: non-image file', file.type)
            return
        }
        // eslint-disable-next-line no-console
        console.log('[PageEditor] uploading image', file.name, file.type, file.size, 'bytes')
        const fileId = await client.uploadPageFile(teamIdForUpload, pageId, file)
        // eslint-disable-next-line no-console
        console.log('[PageEditor] upload returned fileId=', fileId)
        if (!fileId) {
            // eslint-disable-next-line no-alert
            alert('Image upload failed. Check console.')
            return
        }
        const src = client.pageFileURL(teamIdForUpload, pageId, fileId)
        // eslint-disable-next-line no-console
        console.log('[PageEditor] inserting <img src=', src, '>')
        if (!editor) {
            // eslint-disable-next-line no-console
            console.warn('[PageEditor] editor not ready')
            return
        }
        editor.chain().focus().setImage({src}).run()
    }, [editor, teamIdForUpload, pageId])

    // DOM-level paste handler: catches clipboard images BEFORE ProseMirror
    // converts them to base64 and stuffs them into the doc.
    useEffect(() => {
        if (!editor) {
            return
        }
        const dom = editor.view.dom
        const onPaste = (e: ClipboardEvent) => {
            const items = e.clipboardData?.items
            if (!items) {
                return
            }
            for (const item of Array.from(items)) {
                if (item.kind === 'file' && item.type.startsWith('image/')) {
                    const file = item.getAsFile()
                    if (file) {
                        e.preventDefault()
                        e.stopPropagation()
                        // eslint-disable-next-line no-console
                        console.log('[PageEditor] paste image detected')
                        void uploadAndInsert(file)
                        return
                    }
                }
            }
        }
        const onDrop = (e: DragEvent) => {
            const files = e.dataTransfer?.files
            if (!files || files.length === 0) {
                return
            }
            let handled = false
            for (const file of Array.from(files)) {
                if (file.type.startsWith('image/')) {
                    handled = true
                    void uploadAndInsert(file)
                }
            }
            if (handled) {
                e.preventDefault()
                e.stopPropagation()
                // eslint-disable-next-line no-console
                console.log('[PageEditor] drop image detected, count=', files.length)
            }
        }
        dom.addEventListener('paste', onPaste, true)
        dom.addEventListener('drop', onDrop, true)
        return () => {
            dom.removeEventListener('paste', onPaste, true)
            dom.removeEventListener('drop', onDrop, true)
        }
    }, [editor, uploadAndInsert])


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
