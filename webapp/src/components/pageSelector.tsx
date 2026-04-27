// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useMemo, useState} from 'react'
import {FormattedMessage, IntlProvider, useIntl} from 'react-intl'

import {useAppDispatch, useAppSelector} from '../store/hooks'
import {getCurrentTeamId} from '../store/teams'
import {getLanguage} from '../store/language'
import {createPage, getLinkPageToChannel, setLinkPageToChannel} from '../store/pages'
import type {AppDispatch} from '../store'
import {getMessages} from '../i18n'

import client from '../octoClient'
import type {Page} from '../blocks/page'

import Dialog from '../components/dialog'
import Button from '../widgets/buttons/button'
import SearchIcon from '../widgets/icons/search'

// Page selector — modal dialog for linking pages to a channel. Mirrors
// BoardSelector. Triggered from RHSChannelPages empty-state primary button
// (which dispatches setLinkPageToChannel(channelId)). Closes by clearing
// that state.
//
// Search is in-memory across the team's pages — Pages are not so numerous
// that a server-side search is needed for Phase 1.

const windowAny = (window as unknown) as {frontendBaseURL?: string}

const PageSelector = (): JSX.Element | null => {
    const dispatch: AppDispatch = useAppDispatch()
    const intl = useIntl()
    const teamId = useAppSelector(getCurrentTeamId)
    const channelId = useAppSelector(getLinkPageToChannel)

    const [allPages, setAllPages] = useState<Page[]>([])
    const [loading, setLoading] = useState(false)
    const [searchQuery, setSearchQuery] = useState<string>('')
    const [busyPageId, setBusyPageId] = useState<string>('')

    useEffect(() => {
        if (!channelId || !teamId) {
            return
        }
        setLoading(true)
        setSearchQuery('')
        ;(async () => {
            try {
                const resp = await client.getPagesForTeam(teamId)
                if (resp.ok) {
                    setAllPages((await resp.json()) as Page[])
                } else {
                    setAllPages([])
                }
            } catch {
                setAllPages([])
            }
            setLoading(false)
        })()
    }, [channelId, teamId])

    const closeDialog = useCallback(() => {
        dispatch(setLinkPageToChannel(''))
        setSearchQuery('')
    }, [dispatch])

    const linkPage = useCallback(async (page: Page) => {
        if (!channelId) {
            return
        }
        setBusyPageId(page.id)
        try {
            await client.linkPageToChannel(page.id, channelId)
            closeDialog()
        } finally {
            setBusyPageId('')
        }
    }, [channelId, closeDialog])

    const createNewPage = useCallback(async () => {
        if (!teamId || !channelId) {
            return
        }
        setBusyPageId('__new__')
        try {
            const result = await dispatch(createPage({teamId, parentId: '', title: ''}))
            if (createPage.fulfilled.match(result)) {
                // link the freshly created page to the channel
                await client.linkPageToChannel(result.payload.id, channelId)
                closeDialog()
                const base = windowAny.frontendBaseURL || '/boards'
                window.location.href = `${base}/team/${teamId}/pages/${result.payload.id}`
            }
        } finally {
            setBusyPageId('')
        }
    }, [teamId, channelId, dispatch, closeDialog])

    const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
        if (event.key === 'Escape') {
            closeDialog()
        }
    }

    const filtered = useMemo(() => {
        const q = searchQuery.trim().toLowerCase()
        if (!q) {
            return allPages
        }
        return allPages.filter((p) => (p.title || '').toLowerCase().includes(q))
    }, [allPages, searchQuery])

    if (!teamId || !channelId) {
        return null
    }

    return (
        <div className='focalboard-body' onKeyDown={handleKeyDown}>
            <Dialog
                className='BoardSelector'
                onClose={closeDialog}
                title={
                    <FormattedMessage id='pageSelector.title' defaultMessage='Link pages'/>
                }
                toolbar={
                    <Button
                        onClick={createNewPage}
                        emphasis='secondary'
                        disabled={busyPageId === '__new__'}
                    >
                        {busyPageId === '__new__' ? (
                            <FormattedMessage id='pageSelector.creating' defaultMessage='Creating…'/>
                        ) : (
                            <FormattedMessage id='pageSelector.create-a-page' defaultMessage='Create a page'/>
                        )}
                    </Button>
                }
            >
                <div className='BoardSelectorBody'>
                    <div className='head'>
                        <div className='queryWrapper'>
                            <SearchIcon/>
                            <input
                                className='searchQuery'
                                placeholder={intl.formatMessage({
                                    id: 'pageSelector.search-for-pages',
                                    defaultMessage: 'Search for pages',
                                })}
                                type='text'
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                autoFocus={true}
                                maxLength={100}
                            />
                        </div>
                    </div>
                    <div className='searchResults'>
                        {loading && (
                            <div style={{padding: 24, textAlign: 'center', color: '#9ca3af', fontSize: 13}}>
                                {'Loading…'}
                            </div>
                        )}
                        {!loading && filtered.length === 0 && (
                            <div style={{padding: 24, textAlign: 'center', color: '#9ca3af', fontSize: 13}}>
                                {searchQuery ? (
                                    <FormattedMessage
                                        id='pageSelector.no-results'
                                        defaultMessage='No pages match "{q}"'
                                        values={{q: searchQuery}}
                                    />
                                ) : (
                                    <FormattedMessage
                                        id='pageSelector.no-pages'
                                        defaultMessage='No pages in this team yet. Click "Create a page" above to start.'
                                    />
                                )}
                            </div>
                        )}
                        {!loading && filtered.map((p) => (
                            <button
                                key={p.id}
                                disabled={busyPageId === p.id}
                                onClick={() => linkPage(p)}
                                className='BoardSelectorItem'
                                style={{
                                    display: 'flex', alignItems: 'center', width: '100%',
                                    padding: '10px 14px', background: 'transparent', border: 0,
                                    borderTop: '1px solid rgba(var(--center-channel-color-rgb), 0.08)',
                                    textAlign: 'left', cursor: 'pointer', fontSize: 14,
                                    color: 'inherit',
                                }}
                            >
                                <span style={{marginRight: 10, fontSize: 18}}>{p.icon || '📄'}</span>
                                <div style={{flex: 1, overflow: 'hidden'}}>
                                    <div style={{fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                                        {p.title || '(Untitled)'}
                                    </div>
                                </div>
                                <Button emphasis='primary' size='small'>
                                    {busyPageId === p.id ? (
                                        <FormattedMessage id='pageSelector.linking' defaultMessage='Linking…'/>
                                    ) : (
                                        <FormattedMessage id='pageSelector.link' defaultMessage='Link'/>
                                    )}
                                </Button>
                            </button>
                        ))}
                    </div>
                </div>
            </Dialog>
        </div>
    )
}

const IntlPageSelector = (): JSX.Element => {
    const language = useAppSelector<string>(getLanguage)
    return (
        <IntlProvider
            locale={language.split(/[_]/)[0]}
            messages={getMessages(language)}
        >
            <PageSelector/>
        </IntlProvider>
    )
}

export default IntlPageSelector
