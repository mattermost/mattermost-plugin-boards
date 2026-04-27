// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useRef, useState} from 'react'
import {useDispatch, useSelector} from 'react-redux'
import {useEditor, EditorContent} from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'

import {fetchPageContent, savePageContent, getPageContent} from '../store/pages'
import type {AppDispatch} from '../store'
import type {TiptapDoc} from '../blocks/page'

// Pages feature — Tiptap editor host (Phase 1).
//
// Phase 1: single-editor mode with debounced save of Tiptap JSON.
// Phase 2: replace local Tiptap doc with Y.Doc + Collaboration extension
//          + WebSocket relay for Yjs binary updates.
// See docs/PAGES_PLAN.md.

const SAVE_DEBOUNCE_MS = 800

type Props = {
    pageId: string
}

export default function PageEditor({pageId}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const content = useSelector(getPageContent(pageId))
    const [loaded, setLoaded] = useState(false)
    const [saving, setSaving] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
    const saveTimer = useRef<number | null>(null)

    useEffect(() => {
        setLoaded(false)
        dispatch(fetchPageContent(pageId)).finally(() => setLoaded(true))
    }, [pageId, dispatch])

    const editor = useEditor({
        extensions: [StarterKit],
        content: (content?.tiptapJson as unknown) || {type: 'doc', content: [{type: 'paragraph'}]},
        onUpdate: ({editor: e}) => {
            const doc = e.getJSON() as unknown as TiptapDoc
            if (saveTimer.current) {
                window.clearTimeout(saveTimer.current)
            }
            setSaving('saving')
            saveTimer.current = window.setTimeout(async () => {
                const result = await dispatch(savePageContent({pageId, tiptapJson: doc}))
                setSaving(savePageContent.rejected.match(result) ? 'error' : 'saved')
                if (savePageContent.fulfilled.match(result)) {
                    window.setTimeout(() => setSaving('idle'), 1500)
                }
            }, SAVE_DEBOUNCE_MS) as unknown as number
        },
    }, [pageId])

    useEffect(() => {
        if (!editor || !loaded) {
            return
        }
        // Replace editor content with what we just fetched.
        const incoming = content?.tiptapJson || {type: 'doc', content: [{type: 'paragraph'}]}
        editor.commands.setContent(incoming as Record<string, unknown>, false)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [pageId, loaded, editor])

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
