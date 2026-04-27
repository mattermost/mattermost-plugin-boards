// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState, useCallback} from 'react'
import {FormattedMessage, IntlProvider, useIntl} from 'react-intl'

import {getCurrentChannel} from '../store/channels'
import {getCurrentTeamId} from '../store/teams'
import {getLanguage} from '../store/language'
import {setLinkPageToChannel} from '../store/pages'
import {useAppDispatch, useAppSelector} from '../store/hooks'
import Button from '../widgets/buttons/button'

import {getMessages} from '../i18n'

import client from '../octoClient'
import type {Page} from '../blocks/page'

const windowAny = (window as unknown) as {frontendBaseURL?: string}

// Right Hand Sidebar — pages linked to the current channel.
//
// Empty state mirrors RHSChannelBoards (h2 + paragraph + primary button).
// Primary button dispatches setLinkPageToChannel(channelId) which triggers
// the PageSelector modal (registered as a root component, mirrors
// BoardSelector). Listed pages can be unlinked via the × button.

const RHSChannelPages = (): JSX.Element | null => {
    const channel = useAppSelector(getCurrentChannel)
    const teamId = useAppSelector(getCurrentTeamId)
    const dispatch = useAppDispatch()
    const intl = useIntl()

    const [linked, setLinked] = useState<Page[]>([])
    const [loaded, setLoaded] = useState(false)
    const [busyPageId, setBusyPageId] = useState<string>('')

    const reload = useCallback(async () => {
        if (!channel?.id) {
            setLinked([])
            setLoaded(true)
            return
        }
        try {
            const resp = await client.getPagesForChannel(channel.id)
            if (resp.ok) {
                setLinked((await resp.json()) as Page[])
            } else {
                setLinked([])
            }
        } catch {
            setLinked([])
        }
        setLoaded(true)
    }, [channel?.id])

    useEffect(() => { reload() }, [reload])

    const onSelect = (id: string) => {
        if (!teamId) {
            return
        }
        // RHS is mounted outside our React Router tree, so useHistory() is
        // unavailable. Drive a full navigation via window.location instead;
        // the focalboard plugin's product entry will pick up the URL.
        const base = windowAny.frontendBaseURL || '/boards'
        window.location.href = `${base}/team/${teamId}/pages/${id}`
    }

    const onUnlink = useCallback(async (pageId: string) => {
        if (!channel?.id) {
            return
        }
        setBusyPageId(pageId)
        try {
            await client.unlinkPageFromChannel(pageId, channel.id)
            await reload()
        } finally {
            setBusyPageId('')
        }
    }, [channel?.id, reload])

    if (!channel || !loaded) {
        return null
    }

    let channelName: React.ReactNode = channel.display_name
    let headerChannelName: React.ReactNode = channel.display_name
    if (channel.type === 'D') {
        channelName = intl.formatMessage({id: 'rhs-pages.dm', defaultMessage: 'DM'})
        headerChannelName = intl.formatMessage({id: 'rhs-pages.header.dm', defaultMessage: 'this Direct Message'})
    } else if (channel.type === 'G') {
        channelName = intl.formatMessage({id: 'rhs-pages.gm', defaultMessage: 'GM'})
        headerChannelName = intl.formatMessage({id: 'rhs-pages.header.gm', defaultMessage: 'this Group Message'})
    }

    // Empty state — mirrors RHSChannelBoards.empty 1:1
    if (linked.length === 0) {
        return (
            <div className='focalboard-body'>
                <div className='RHSChannelBoards empty'>
                    <h2>
                        <FormattedMessage
                            id='rhs-pages.no-pages-linked-to-channel'
                            defaultMessage='No pages are linked to {channelName} yet'
                            values={{channelName: headerChannelName}}
                        />
                    </h2>
                    <div className='empty-paragraph'>
                        <FormattedMessage
                            id='rhs-pages.no-pages-linked-to-channel-description'
                            defaultMessage='Pages is a collaborative document workspace for manuals, onboarding, and team knowledge — pin a page here to make it easy to find from this channel.'
                        />
                    </div>
                    <Button
                        onClick={() => dispatch(setLinkPageToChannel(channel.id))}
                        emphasis='primary'
                        size='medium'
                    >
                        <FormattedMessage
                            id='rhs-pages.link-pages-to-channel'
                            defaultMessage='Link pages to {channelName}'
                            values={{channelName}}
                        />
                    </Button>
                </div>
            </div>
        )
    }

    // Populated state — list + Add button
    return (
        <div className='focalboard-body'>
            <div className='RHSChannelBoards' style={{padding: 12}}>
                <div style={{display: 'flex', alignItems: 'center', marginBottom: 8}}>
                    <h3 style={{margin: 0, flex: 1, fontSize: 13, textTransform: 'uppercase', color: 'rgba(var(--center-channel-color-rgb), 0.56)'}}>
                        <FormattedMessage id='rhs-pages.linked-pages' defaultMessage='Linked pages'/>
                    </h3>
                    <Button
                        onClick={() => dispatch(setLinkPageToChannel(channel.id))}
                        emphasis='secondary'
                        size='small'
                    >
                        <FormattedMessage id='generic.add' defaultMessage='Add'/>
                    </Button>
                </div>

                <ul style={{listStyle: 'none', padding: 0, margin: 0}}>
                    {linked.map((p) => (
                        <li key={p.id} style={{display: 'flex', alignItems: 'center'}}>
                            <button
                                onClick={() => onSelect(p.id)}
                                style={{
                                    display: 'flex', alignItems: 'center',
                                    flex: 1, padding: '8px 4px',
                                    background: 'transparent', border: 0,
                                    textAlign: 'left', cursor: 'pointer',
                                    fontSize: 14, borderRadius: 4,
                                    color: 'inherit',
                                }}
                            >
                                <span style={{marginRight: 8}}>{p.icon || '📄'}</span>
                                <span style={{flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                                    {p.title || '(Untitled)'}
                                </span>
                            </button>
                            <button
                                onClick={() => onUnlink(p.id)}
                                disabled={busyPageId === p.id}
                                title='Unlink'
                                style={{
                                    background: 'transparent', border: 0, cursor: 'pointer',
                                    color: '#9ca3af', padding: '0 6px', fontSize: 16,
                                }}
                            >
                                {busyPageId === p.id ? '…' : '×'}
                            </button>
                        </li>
                    ))}
                </ul>
            </div>
        </div>
    )
}

const IntlRHSChannelPages = () => {
    const language = useAppSelector<string>(getLanguage)
    return (
        <IntlProvider
            locale={language.split(/[_]/)[0]}
            messages={getMessages(language)}
        >
            <RHSChannelPages/>
        </IntlProvider>
    )
}

export default IntlRHSChannelPages
