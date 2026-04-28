// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react'
import {NodeViewWrapper, NodeViewProps} from '@tiptap/react'

// Pages — custom Tiptap NodeView for images.
//
// Why not a plain <img src={serverURL}>? Mattermost's CSRF middleware
// rejects cookie-only GET requests without an `X-Requested-With` header,
// which the browser doesn't add for <img> tag fetches → 400. We fetch the
// file ourselves via fetch() (which lets us set the header), then render
// from a Blob URL.
//
// The Y.Doc still stores the server URL — each peer creates its own
// session-local Blob URL on render, so the doc stays portable.

export default function PageImageNodeView(props: NodeViewProps): JSX.Element {
    const src = (props.node.attrs.src as string) || ''
    const alt = (props.node.attrs.alt as string) || ''
    const title = (props.node.attrs.title as string) || ''
    const [blobUrl, setBlobUrl] = useState<string | null>(null)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        if (!src) {
            return
        }
        let cancelled = false
        let url: string | null = null
        const load = async () => {
            try {
                const resp = await fetch(src, {
                    headers: {'X-Requested-With': 'XMLHttpRequest'},
                    credentials: 'same-origin',
                })
                if (!resp.ok) {
                    if (!cancelled) {
                        setError(`HTTP ${resp.status}`)
                    }
                    return
                }
                const blob = await resp.blob()
                if (cancelled) {
                    return
                }
                url = URL.createObjectURL(blob)
                setBlobUrl(url)
            } catch (e) {
                if (!cancelled) {
                    setError(String(e))
                }
            }
        }
        void load()
        return () => {
            cancelled = true
            if (url) {
                URL.revokeObjectURL(url)
            }
        }
    }, [src])

    return (
        <NodeViewWrapper
            as='span'
            className='PageEditor__image-wrap'
            data-drag-handle=''
        >
            {blobUrl ? (
                <img
                    src={blobUrl}
                    alt={alt}
                    title={title}
                    className='PageEditor__image'
                />
            ) : (
                <span className='PageEditor__image-placeholder'>
                    {error ? `이미지 로드 실패 (${error})` : '이미지 로딩 중…'}
                </span>
            )}
        </NodeViewWrapper>
    )
}
