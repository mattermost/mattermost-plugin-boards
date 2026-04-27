// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useMemo, useState} from 'react'
import {useSelector} from 'react-redux'
import type {Channel} from '@mattermost/types/channels'
import type {GlobalState} from '@mattermost/types/store'

import client from '../octoClient'
import type {PageChannelLink} from '../blocks/page'

// PageChannelsBar — small badge row inside PageView showing channels this
// page is pinned to, with unlink (×) and a "+ Pin to channel" action.
//
// Uses MM's Redux state directly for channel name lookups and the team's
// accessible channel list (we don't fetch channels from focalboard backend).

type Props = {
    pageId: string
    teamId: string
}

export default function PageChannelsBar({pageId, teamId}: Props): JSX.Element | null {
    const [links, setLinks] = useState<PageChannelLink[]>([])
    const [loaded, setLoaded] = useState(false)
    const [pickerOpen, setPickerOpen] = useState(false)
    const [busy, setBusy] = useState<string>('')

    const channelsById = useSelector((state: GlobalState) => state.entities.channels.channels)
    const myMembers = useSelector((state: GlobalState) => state.entities.channels.myMembers)
    const channelsInTeam = useSelector((state: GlobalState) => state.entities.channels.channelsInTeam)

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

    const onLink = useCallback(async (channelId: string) => {
        setBusy(channelId)
        try {
            await client.linkPageToChannel(pageId, channelId)
            await reload()
            setPickerOpen(false)
        } finally {
            setBusy('')
        }
    }, [pageId, reload])

    const onUnlink = useCallback(async (channelId: string) => {
        setBusy(channelId)
        try {
            await client.unlinkPageFromChannel(pageId, channelId)
            await reload()
        } finally {
            setBusy('')
        }
    }, [pageId, reload])

    const teamChannelIds: string[] = useMemo(() => {
        const set = (channelsInTeam as Record<string, Set<string> | string[]> | undefined)?.[teamId]
        if (!set) {
            return []
        }
        return Array.isArray(set) ? set : Array.from(set)
    }, [channelsInTeam, teamId])

    const linkedSet = useMemo(() => new Set(links.map((l) => l.channelId)), [links])

    const accessibleCandidates: Channel[] = useMemo(() => {
        return teamChannelIds
            .map((id) => channelsById[id])
            .filter((c): c is Channel => Boolean(c) && Boolean(myMembers[c.id]) && !linkedSet.has(c.id))
            .sort((a, b) => (a.display_name || a.name).localeCompare(b.display_name || b.name))
    }, [teamChannelIds, channelsById, myMembers, linkedSet])

    if (!loaded) {
        return null
    }

    return (
        <div className='PageChannelsBar' style={{margin: '8px 0', display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 6}}>
            {links.length === 0 && !pickerOpen && (
                <button
                    onClick={() => setPickerOpen(true)}
                    style={{
                        background: 'transparent', border: '1px dashed rgba(var(--center-channel-color-rgb), 0.24)',
                        borderRadius: 12, padding: '3px 10px', fontSize: 12,
                        color: 'rgba(var(--center-channel-color-rgb), 0.56)', cursor: 'pointer',
                    }}
                >
                    {'+ Pin to channel'}
                </button>
            )}
            {links.map((l) => {
                const ch = channelsById[l.channelId]
                const name = ch?.display_name || ch?.name || l.channelId.substring(0, 6) + '…'
                return (
                    <span
                        key={l.channelId}
                        style={{
                            display: 'inline-flex', alignItems: 'center', gap: 4,
                            background: 'rgba(var(--button-bg-rgb), 0.10)',
                            color: 'rgba(var(--button-bg-rgb), 1)',
                            borderRadius: 12, padding: '3px 4px 3px 10px', fontSize: 12,
                        }}
                    >
                        <span>{'#'} {name}</span>
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
            {links.length > 0 && !pickerOpen && (
                <button
                    onClick={() => setPickerOpen(true)}
                    style={{
                        background: 'transparent', border: '1px solid rgba(var(--center-channel-color-rgb), 0.16)',
                        borderRadius: 12, padding: '3px 8px', fontSize: 12, cursor: 'pointer',
                        color: 'rgba(var(--center-channel-color-rgb), 0.56)',
                    }}
                >
                    {'+'}
                </button>
            )}

            {pickerOpen && (
                <div style={{
                    position: 'relative', display: 'inline-block',
                }}>
                    <div style={{
                        position: 'absolute', top: 0, left: 0, zIndex: 100,
                        background: 'var(--center-channel-bg)',
                        border: '1px solid rgba(var(--center-channel-color-rgb), 0.16)',
                        borderRadius: 6, minWidth: 240, maxWidth: 320, maxHeight: 280, overflowY: 'auto',
                        boxShadow: '0 8px 32px rgba(0,0,0,0.16)',
                        padding: '4px 0',
                    }}>
                        <div style={{padding: '4px 12px 8px', borderBottom: '1px solid rgba(var(--center-channel-color-rgb),0.08)', display: 'flex', alignItems: 'center'}}>
                            <strong style={{flex: 1, fontSize: 12}}>{'Pin to channel'}</strong>
                            <button
                                onClick={() => setPickerOpen(false)}
                                style={{background: 'transparent', border: 0, cursor: 'pointer', fontSize: 14, color: '#9ca3af'}}
                            >{'×'}</button>
                        </div>
                        {accessibleCandidates.length === 0 && (
                            <div style={{padding: '12px', fontSize: 12, color: '#9ca3af'}}>
                                {'No channels to pin (already pinned everywhere or no membership).'}
                            </div>
                        )}
                        {accessibleCandidates.map((c) => (
                            <button
                                key={c.id}
                                disabled={busy === c.id}
                                onClick={() => onLink(c.id)}
                                style={{
                                    display: 'block', width: '100%', padding: '6px 12px',
                                    textAlign: 'left', background: 'transparent', border: 0,
                                    cursor: 'pointer', fontSize: 13, color: 'inherit',
                                }}
                            >
                                {c.type === 'D' ? '@' : '#'} {c.display_name || c.name}
                                {busy === c.id && <span style={{marginLeft: 6, color: '#9ca3af'}}>{'pinning…'}</span>}
                            </button>
                        ))}
                    </div>
                </div>
            )}
        </div>
    )
}
