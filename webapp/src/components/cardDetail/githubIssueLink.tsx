// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Card} from '../../blocks/card'
import {GitHubIssue, GitHubConnectedResponse} from '../../github'
import octoClient from '../../octoClient'
import {sendFlashMessage} from '../flashMessages'
import IconButton from '../../widgets/buttons/iconButton'
import LinkIcon from '../../widgets/icons/Link'
import SearchIcon from '../../widgets/icons/search'
import CloseIcon from '../../widgets/icons/close'
import Label from '../../widgets/label'

import './githubIssueLink.scss'

type Props = {
    card: Card
    readonly: boolean
}

const GitHubIssueLink = (props: Props): JSX.Element => {
    const {card, readonly} = props
    const intl = useIntl()

    const [connectionStatus, setConnectionStatus] = useState<GitHubConnectedResponse | null>(null)
    const [loading, setLoading] = useState(true)
    const [showSearch, setShowSearch] = useState(false)
    const [searchQuery, setSearchQuery] = useState('')
    const [searchResults, setSearchResults] = useState<GitHubIssue[]>([])
    const [searching, setSearching] = useState(false)
    const [linkedIssue, setLinkedIssue] = useState<GitHubIssue | null>(null)

    // Check GitHub connection status
    useEffect(() => {
        loadConnectionStatus()
    }, [])

    const loadConnectionStatus = async () => {
        try {
            setLoading(true)
            const status = await octoClient.getGitHubConnected()
            setConnectionStatus(status || null)
        } catch (error) {
            console.error('Failed to check GitHub connection:', error)
        } finally {
            setLoading(false)
        }
    }

    // Search for GitHub issues
    const handleSearch = useCallback(async () => {
        if (!searchQuery.trim()) {
            setSearchResults([])
            return
        }

        try {
            setSearching(true)
            const results = await octoClient.searchGitHubIssues(searchQuery)
            setSearchResults(results)
        } catch (error) {
            console.error('Failed to search GitHub issues:', error)
            sendFlashMessage({
                content: intl.formatMessage({
                    id: 'GitHubIssueLink.searchError',
                    defaultMessage: 'Failed to search GitHub issues',
                }),
                severity: 'low',
            })
        } finally {
            setSearching(false)
        }
    }, [searchQuery, intl])

    const handleLinkIssue = useCallback((issue: GitHubIssue) => {
        setLinkedIssue(issue)
        setShowSearch(false)
        setSearchQuery('')
        setSearchResults([])
        sendFlashMessage({
            content: intl.formatMessage({
                id: 'GitHubIssueLink.linked',
                defaultMessage: 'Linked to GitHub issue #{number}',
            }, {number: issue.number}),
            severity: 'high',
        })
    }, [intl])

    const handleUnlinkIssue = useCallback(() => {
        setLinkedIssue(null)
        sendFlashMessage({
            content: intl.formatMessage({
                id: 'GitHubIssueLink.unlinked',
                defaultMessage: 'Unlinked from GitHub issue',
            }),
            severity: 'high',
        })
    }, [intl])

    if (loading) {
        return (
            <div className='GitHubIssueLink'>
                <div className='GitHubIssueLink__loading'>
                    <FormattedMessage
                        id='GitHubIssueLink.loading'
                        defaultMessage='Loading GitHub connection...'
                    />
                </div>
            </div>
        )
    }

    if (!connectionStatus?.connected) {
        return (
            <div className='GitHubIssueLink'>
                <div className='GitHubIssueLink__disconnected'>
                    <div className='GitHubIssueLink__disconnected-icon'>
                        <LinkIcon/>
                    </div>
                    <div className='GitHubIssueLink__disconnected-text'>
                        <FormattedMessage
                            id='GitHubIssueLink.notConnected'
                            defaultMessage='Connect your GitHub account to link issues'
                        />
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className='GitHubIssueLink'>
            <div className='GitHubIssueLink__header'>
                <div className='GitHubIssueLink__title'>
                    <LinkIcon/>
                    <FormattedMessage
                        id='GitHubIssueLink.title'
                        defaultMessage='GitHub Issue'
                    />
                </div>
                {connectionStatus.github_username && (
                    <div className='GitHubIssueLink__username'>
                        @{connectionStatus.github_username}
                    </div>
                )}
            </div>

            {/* Linked Issue Display */}
            {linkedIssue && (
                <div className='GitHubIssueLink__linked'>
                    <div className='GitHubIssueLink__issue'>
                        <div className='GitHubIssueLink__issue-header'>
                            <a
                                href={linkedIssue.html_url}
                                target='_blank'
                                rel='noopener noreferrer'
                                className='GitHubIssueLink__issue-number'
                            >
                                #{linkedIssue.number}
                            </a>
                            <div className='GitHubIssueLink__issue-state'>
                                <Label color={linkedIssue.state === 'open' ? 'propColorGreen' : 'propColorGray'}>
                                    <span className='Label-text'>{linkedIssue.state}</span>
                                </Label>
                            </div>
                            {!readonly && (
                                <IconButton
                                    className='GitHubIssueLink__issue-unlink'
                                    onClick={handleUnlinkIssue}
                                    icon={<CloseIcon/>}
                                    title={intl.formatMessage({
                                        id: 'GitHubIssueLink.unlink',
                                        defaultMessage: 'Unlink issue',
                                    })}
                                    size='small'
                                />
                            )}
                        </div>
                        <div className='GitHubIssueLink__issue-title'>
                            {linkedIssue.title}
                        </div>
                        {linkedIssue.labels && linkedIssue.labels.length > 0 && (
                            <div className='GitHubIssueLink__issue-labels'>
                                {linkedIssue.labels.map((label) => (
                                    <span
                                        key={label.id}
                                        className='GitHubIssueLink__label'
                                        style={{backgroundColor: `#${label.color}`}}
                                    >
                                        {label.name}
                                    </span>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Search Interface */}
            {!linkedIssue && !readonly && (
                <div className='GitHubIssueLink__search'>
                    {!showSearch ? (
                        <button
                            className='GitHubIssueLink__search-button'
                            onClick={() => setShowSearch(true)}
                        >
                            <SearchIcon/>
                            <FormattedMessage
                                id='GitHubIssueLink.searchButton'
                                defaultMessage='Link to GitHub issue'
                            />
                        </button>
                    ) : (
                        <div className='GitHubIssueLink__search-panel'>
                            <div className='GitHubIssueLink__search-input-wrapper'>
                                <SearchIcon/>
                                <input
                                    type='text'
                                    className='GitHubIssueLink__search-input'
                                    placeholder={intl.formatMessage({
                                        id: 'GitHubIssueLink.searchPlaceholder',
                                        defaultMessage: 'Search GitHub issues...',
                                    })}
                                    value={searchQuery}
                                    onChange={(e) => setSearchQuery(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') {
                                            handleSearch()
                                        } else if (e.key === 'Escape') {
                                            setShowSearch(false)
                                            setSearchQuery('')
                                            setSearchResults([])
                                        }
                                    }}
                                    autoFocus={true}
                                />
                                <IconButton
                                    className='GitHubIssueLink__search-close'
                                    onClick={() => {
                                        setShowSearch(false)
                                        setSearchQuery('')
                                        setSearchResults([])
                                    }}
                                    icon={<CloseIcon/>}
                                    title={intl.formatMessage({
                                        id: 'GitHubIssueLink.closeSearch',
                                        defaultMessage: 'Close search',
                                    })}
                                    size='small'
                                />
                            </div>

                            {searching && (
                                <div className='GitHubIssueLink__search-loading'>
                                    <FormattedMessage
                                        id='GitHubIssueLink.searching'
                                        defaultMessage='Searching...'
                                    />
                                </div>
                            )}

                            {!searching && searchResults.length > 0 && (
                                <div className='GitHubIssueLink__search-results'>
                                    {searchResults.map((issue) => (
                                        <div
                                            key={issue.number}
                                            className='GitHubIssueLink__search-result'
                                            onClick={() => handleLinkIssue(issue)}
                                            role='button'
                                            tabIndex={0}
                                            onKeyDown={(e) => {
                                                if (e.key === 'Enter' || e.key === ' ') {
                                                    e.preventDefault()
                                                    handleLinkIssue(issue)
                                                }
                                            }}
                                        >
                                            <div className='GitHubIssueLink__search-result-header'>
                                                <span className='GitHubIssueLink__search-result-number'>
                                                    #{issue.number}
                                                </span>
                                                <Label color={issue.state === 'open' ? 'propColorGreen' : 'propColorGray'}>
                                                    <span className='Label-text'>{issue.state}</span>
                                                </Label>
                                            </div>
                                            <div className='GitHubIssueLink__search-result-title'>
                                                {issue.title}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}

                            {!searching && searchQuery && searchResults.length === 0 && (
                                <div className='GitHubIssueLink__search-empty'>
                                    <FormattedMessage
                                        id='GitHubIssueLink.noResults'
                                        defaultMessage='No issues found'
                                    />
                                </div>
                            )}
                        </div>
                    )}
                </div>
            )}

            {!linkedIssue && readonly && (
                <div className='GitHubIssueLink__empty'>
                    <FormattedMessage
                        id='GitHubIssueLink.noIssueLinked'
                        defaultMessage='No GitHub issue linked'
                    />
                </div>
            )}
        </div>
    )
}

export default GitHubIssueLink

