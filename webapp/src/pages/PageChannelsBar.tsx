// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react'

import {getCurrentChannel} from '../store/channels'
import {useAppSelector} from '../store/hooks'

import client from '../octoClient'
import type {PageChannelLink} from '../blocks/page'

// PageChannelsBar — small badge row inside PageView showing channels this
// page is pinned to. Each badge has × to unlink. A "+ Pin to current
// channel" button pins to the user's currently-active channel (when one
// is known to focalboard's store).
//
// Limited by focalboard's redux state, which only tracks the current
// channel — we cannot enumerate all team channels here. Users wanting to
// pin to a different channel can do so from that channel's RHS via the
// PageSelector flow.

type Props = {
    pageId: string
    teamId: string
}

export default function PageChannelsBar({pageId}: Props): JSX.Element | null {
    const [links, setLinks] = useState<PageChannelLink[]>([])
    const [loaded, setLoaded] = useState(false)
    const [busy, setBusy] = useState<string>('')

    const currentChannel = useAppSelector(getCurrentChannel)

    const reload = useCallback(async () => {
        try {
            const resp = await client.getPageChannelLinks(pageId)
            if (resp.ok) {
                setLinks((await resp.json()) as PageChannelLink[])
            } else {
                setLinks([])
            }
        } catch {
            setLinks([])
        }
        setLoaded(true)
    }, [pageId])

    useEffect(() => { reload() }, [reload])

    const onPinCurrent = useCallback(async () => {
        if (!currentChannel?.id) {
            return
        }
        setBusy(currentChannel.id)
        try {
            await client.linkPageToChannel(pageId, currentChannel.id)
            await reload()
        } finally {
            setBusy('')
        }
    }, [currentChannel?.id, pageId, reload])

    const onUnlink = useCallback(async (channelId: string) => {
        setBusy(channelId)
        try {
            await client.unlinkPageFromChannel(pageId, channelId)
            await reload()
        } finally {
            setBusy('')
        }
    }, [pageId, reload])

    if (!loaded) {
        return null
    }

    const linkedSet = new Set(links.map((l) => l.channelId))
    const canPinCurrent = !!currentChannel?.id && !linkedSet.has(currentChannel.id)

    return (
        <div
            className='PageChannelsBar'
            style={{margin: '8px 0', display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 6}}
        >
            {links.map((l) => {
                const isCurrent = currentChannel?.id === l.channelId
                const label = isCurrent ?
                    (currentChannel?.display_name || currentChannel?.name || l.channelId.substring(0, 8)) :
                    l.channelId.substring(0, 8) + '…'
                return (
                    <span
                        key={l.channelId}
                        style={{
                            display: 'inline-flex', alignItems: 'center', gap: 4,
                            background: 'rgba(var(--button-bg-rgb), 0.10)',
                            color: 'rgba(var(--button-bg-rgb), 1)',
                            borderRadius: 12, padding: '3px 4px 3px 10px', fontSize: 12,
                        }}
                        title={l.channelId}
                    >
                        <span>{'#'} {label}</span>
                        <button
                            onClick={() => onUnlink(l.channelId)}
                            disabled={busy === l.channelId}
                            title='Unlink'
                            style={{
                                background: 'transparent', border: 0, cursor: 'pointer',
                                color: 'inherit', fontSize: 14, padding: '0 4px', opacity: 0.7,
                            }}
                        >
                            {busy === l.channelId ? '…' : '×'}
                        </button>
                    </span>
                )
            })}
            {canPinCurrent && (
                <button
                    onClick={onPinCurrent}
                    disabled={busy !== ''}
                    style={{
                        background: 'transparent',
                        border: '1px dashed rgba(var(--center-channel-color-rgb), 0.24)',
                        borderRadius: 12, padding: '3px 10px', fontSize: 12,
                        color: 'rgba(var(--center-channel-color-rgb), 0.56)', cursor: 'pointer',
                    }}
                    title={`Pin to #${currentChannel?.display_name || currentChannel?.name}`}
                >
                    {busy === currentChannel?.id ? 'Pinning…' : `+ Pin to #${currentChannel?.display_name || currentChannel?.name}`}
                </button>
            )}
            {!currentChannel && links.length === 0 && (
                <span style={{fontSize: 11, color: '#9ca3af'}}>
                    {'Open a channel and use its Pages panel to link this page.'}
                </span>
            )}
        </div>
    )
}
