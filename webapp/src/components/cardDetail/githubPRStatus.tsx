// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Card} from '../../blocks/card'
import {GitHubPRDetails, GitHubConnectedResponse} from '../../github'
import octoClient from '../../octoClient'
import IconButton from '../../widgets/buttons/iconButton'
import CompassIcon from '../../widgets/icons/compassIcon'
import CloseIcon from '../../widgets/icons/close'
import Label from '../../widgets/label'

import './githubPRStatus.scss'

// Well-known property ID for GitHub PRs JSON data (synced by external cron)
// Exported so it can be used in other components (Issue 9)
export const GITHUB_PRS_PROPERTY_ID = 'agithubprs1prp7x9jkxd1ec66j'

// PR status as synced by the cron
type PRStatus = 'NEW' | 'CI' | 'FAILED' | 'READY' | 'MERGED'

// Shape of each PR entry stored in the card property JSON
interface PropertyPR {
    number: number
    title: string
    url: string
    status: PRStatus
    repo: string
    branch?: string
}

type Props = {
    card: Card
    readonly: boolean
    hasBranch?: boolean
}

// Parse PR URL to extract owner, repo, and number
// Supports: https://github.com/owner/repo/pull/123
const parsePRUrl = (url: string): {owner: string; repo: string; number: number} | null => {
    const match = url.match(/github\.com\/([^/]+)\/([^/]+)\/pull\/(\d+)/)
    if (match) {
        return {
            owner: match[1],
            repo: match[2],
            number: parseInt(match[3], 10),
        }
    }
    return null
}

// Status badge color mapping
const getStatusColor = (status: PRStatus): string => {
    switch (status) {
    case 'NEW':
        return 'propColorGray'
    case 'CI':
        return 'propColorBlue'
    case 'FAILED':
        return 'propColorRed'
    case 'READY':
        return 'propColorGreen'
    case 'MERGED':
        return 'propColorPurple'
    default:
        return 'propColorGray'
    }
}

// Valid PR status values
const VALID_PR_STATUSES: PRStatus[] = ['NEW', 'CI', 'FAILED', 'READY', 'MERGED']

// Sanitize URL to prevent XSS — only allow http(s) schemes
const sanitizePRUrl = (url: string): string => {
    try {
        const parsed = new URL(url)
        if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
            return parsed.href
        }
    } catch {
        // Invalid URL
    }
    return ''
}

// Parse the JSON property value into an array of PRs
const parsePRsFromProperty = (card: Card): PropertyPR[] => {
    const raw = card.fields?.properties?.[GITHUB_PRS_PROPERTY_ID]
    if (!raw || typeof raw !== 'string') {
        return []
    }
    try {
        const parsed = JSON.parse(raw)
        if (Array.isArray(parsed)) {
            return parsed
                .filter((pr: any) =>
                    pr &&
                    typeof pr.number === 'number' &&
                    typeof pr.url === 'string' &&
                    typeof pr.status === 'string' &&
                    VALID_PR_STATUSES.includes(pr.status) &&
                    typeof pr.repo === 'string' &&
                    pr.repo.length > 0,
                )
                .map((pr: any) => ({
                    ...pr,
                    url: sanitizePRUrl(pr.url),
                }))
                .filter((pr: PropertyPR) => pr.url.length > 0)
        }
    } catch {
        // Invalid JSON — ignore
    }
    return []
}

// Sub-component: renders PRs from card property data (cron-synced)
const PropertyPRList = ({prs}: {prs: PropertyPR[]}): JSX.Element => {
    return (
        <div className='GitHubPRStatus__property-list'>
            {prs.map((pr) => (
                <div
                    key={`${pr.repo}-${pr.number}`}
                    className='GitHubPRStatus__property-pr'
                >
                    <a
                        href={pr.url}
                        target='_blank'
                        rel='noopener noreferrer'
                        className='GitHubPRStatus__property-pr-title'
                        title={`${pr.repo}#${pr.number}`}
                    >
                        {pr.title || `#${pr.number}`}
                    </a>
                    <Label color={getStatusColor(pr.status)}>
                        <span className='Label-text'>{pr.status}</span>
                    </Label>
                </div>
            ))}
        </div>
    )
}

const GitHubPRStatus = (props: Props): JSX.Element | null => {
    const {card, readonly, hasBranch} = props
    const intl = useIntl()

    const [connectionStatus, setConnectionStatus] = useState<GitHubConnectedResponse | null>(null)
    const [loading, setLoading] = useState(true)
    const [showForm, setShowForm] = useState(false)
    const [prUrl, setPrUrl] = useState('')
    const [prDetails, setPrDetails] = useState<GitHubPRDetails | null>(null)
    const [loadingPR, setLoadingPR] = useState(false)
    const [error, setError] = useState<string | null>(null)

    // PRs from card property (cron-synced)
    const propertyPRs = parsePRsFromProperty(card)
    const hasPropertyPRs = propertyPRs.length > 0

    // Check GitHub connection status (only needed for manual tracking)
    useEffect(() => {
        if (!hasPropertyPRs) {
            loadConnectionStatus()
        } else {
            setLoading(false)
        }
    }, [hasPropertyPRs])

    // Reset all state when card changes to prevent data leaking between cards
    useEffect(() => {
        setShowForm(false)
        setPrUrl('')
        setPrDetails(null)
        setError(null)
    }, [card.id])

    const loadConnectionStatus = async () => {
        try {
            setLoading(true)
            const status = await octoClient.getGitHubConnected()
            setConnectionStatus(status || null)
        } catch (err) {
            console.error('Failed to check GitHub connection:', err)
        } finally {
            setLoading(false)
        }
    }

    const handleLoadPR = useCallback(async () => {
        if (!prUrl.trim()) {
            return
        }

        const parsed = parsePRUrl(prUrl.trim())
        if (!parsed) {
            setError(intl.formatMessage({
                id: 'GitHubPRStatus.invalidUrl',
                defaultMessage: 'Invalid GitHub PR URL. Use format: https://github.com/owner/repo/pull/123',
            }))
            return
        }

        try {
            setLoadingPR(true)
            setError(null)
            const pr = await octoClient.getGitHubPR(parsed.owner, parsed.repo, parsed.number)
            if (pr) {
                setPrDetails(pr)
                setShowForm(false)
                setPrUrl('')
            } else {
                setError(intl.formatMessage({
                    id: 'GitHubPRStatus.notFound',
                    defaultMessage: 'PR not found or you do not have access',
                }))
            }
        } catch (err) {
            console.error('Failed to load PR:', err)
            setError(intl.formatMessage({
                id: 'GitHubPRStatus.loadError',
                defaultMessage: 'Failed to load PR details',
            }))
        } finally {
            setLoadingPR(false)
        }
    }, [prUrl, intl])

    const handleClearPR = useCallback(() => {
        setPrDetails(null)
    }, [])

    const getPRStateColor = (pr: GitHubPRDetails): string => {
        if (pr.merged) {
            return 'propColorPurple'
        }
        if (pr.state === 'closed') {
            return 'propColorRed'
        }
        return 'propColorGreen'
    }

    const getPRStateText = (pr: GitHubPRDetails): string => {
        if (pr.merged) {
            return 'merged'
        }
        return pr.state
    }

    // If we have property-based PRs, always show them (no GitHub connection needed)
    if (hasPropertyPRs) {
        return (
            <div className='GitHubPRStatus'>
                <div className='GitHubPRStatus__header'>
                    <div className='GitHubPRStatus__title'>
                        <CompassIcon icon='source-pull'/>
                        <FormattedMessage
                            id='GitHubPRStatus.title'
                            defaultMessage='Pull Request'
                        />
                    </div>
                </div>
                <PropertyPRList prs={propertyPRs}/>
            </div>
        )
    }

    // Fallback: manual PR tracking (requires GitHub connection + branch)
    if (loading || !connectionStatus?.connected || !hasBranch) {
        return null
    }

    return (
        <div className='GitHubPRStatus'>
            <div className='GitHubPRStatus__header'>
                <div className='GitHubPRStatus__title'>
                    <CompassIcon icon='source-pull'/>
                    <FormattedMessage
                        id='GitHubPRStatus.title'
                        defaultMessage='Pull Request'
                    />
                </div>
            </div>

            {/* PR Details Display */}
            {prDetails && (
                <div className='GitHubPRStatus__pr'>
                    <div className='GitHubPRStatus__pr-header'>
                        <a
                            href={prDetails.html_url}
                            target='_blank'
                            rel='noopener noreferrer'
                            className='GitHubPRStatus__pr-number'
                        >
                            #{prDetails.number}
                        </a>
                        <div className='GitHubPRStatus__pr-state'>
                            <Label color={getPRStateColor(prDetails)}>
                                <span className='Label-text'>{getPRStateText(prDetails)}</span>
                            </Label>
                        </div>
                        {!readonly && (
                            <IconButton
                                className='GitHubPRStatus__pr-clear'
                                onClick={handleClearPR}
                                icon={<CloseIcon/>}
                                title={intl.formatMessage({
                                    id: 'GitHubPRStatus.clear',
                                    defaultMessage: 'Clear PR',
                                })}
                                size='small'
                            />
                        )}
                    </div>
                    <div className='GitHubPRStatus__pr-title'>
                        {prDetails.title}
                    </div>
                    <div className='GitHubPRStatus__pr-meta'>
                        <span className='GitHubPRStatus__pr-branch'>
                            <CompassIcon icon='source-branch'/>
                            {prDetails.head.ref}
                        </span>
                        <span className='GitHubPRStatus__pr-arrow'>→</span>
                        <span className='GitHubPRStatus__pr-branch'>
                            {prDetails.base.ref}
                        </span>
                    </div>
                    {prDetails.requested_reviewers && prDetails.requested_reviewers.length > 0 && (
                        <div className='GitHubPRStatus__pr-reviewers'>
                            <span className='GitHubPRStatus__pr-reviewers-label'>
                                <FormattedMessage
                                    id='GitHubPRStatus.reviewers'
                                    defaultMessage='Reviewers:'
                                />
                            </span>
                            {prDetails.requested_reviewers.map((reviewer) => (
                                <a
                                    key={reviewer.id}
                                    href={reviewer.html_url}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                    className='GitHubPRStatus__pr-reviewer'
                                >
                                    <img
                                        src={reviewer.avatar_url}
                                        alt={reviewer.login}
                                        className='GitHubPRStatus__pr-reviewer-avatar'
                                    />
                                    @{reviewer.login}
                                </a>
                            ))}
                        </div>
                    )}
                    {prDetails.labels && prDetails.labels.length > 0 && (
                        <div className='GitHubPRStatus__pr-labels'>
                            {prDetails.labels.map((label) => (
                                <span
                                    key={label.id}
                                    className='GitHubPRStatus__label'
                                    style={{backgroundColor: `#${label.color}`}}
                                >
                                    {label.name}
                                </span>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {/* Add PR Form */}
            {!prDetails && !readonly && (
                <div className='GitHubPRStatus__add'>
                    {!showForm ? (
                        <button
                            type='button'
                            className='GitHubPRStatus__add-button'
                            onClick={() => setShowForm(true)}
                        >
                            <CompassIcon icon='plus'/>
                            <FormattedMessage
                                id='GitHubPRStatus.addButton'
                                defaultMessage='Track a Pull Request'
                            />
                        </button>
                    ) : (
                        <div className='GitHubPRStatus__form'>
                            <div className='GitHubPRStatus__form-header'>
                                <FormattedMessage
                                    id='GitHubPRStatus.formTitle'
                                    defaultMessage='Add Pull Request'
                                />
                                <IconButton
                                    className='GitHubPRStatus__form-close'
                                    onClick={() => {
                                        setShowForm(false)
                                        setPrUrl('')
                                        setError(null)
                                    }}
                                    icon={<CloseIcon/>}
                                    title={intl.formatMessage({
                                        id: 'GitHubPRStatus.closeForm',
                                        defaultMessage: 'Close',
                                    })}
                                    size='small'
                                />
                            </div>
                            <div className='GitHubPRStatus__form-field'>
                                <input
                                    type='text'
                                    className='GitHubPRStatus__form-input'
                                    placeholder={intl.formatMessage({
                                        id: 'GitHubPRStatus.urlPlaceholder',
                                        defaultMessage: 'https://github.com/owner/repo/pull/123',
                                    })}
                                    value={prUrl}
                                    onChange={(e) => {
                                        setPrUrl(e.target.value)
                                        setError(null)
                                    }}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter' && !loadingPR) {
                                            handleLoadPR()
                                        } else if (e.key === 'Escape') {
                                            setShowForm(false)
                                            setPrUrl('')
                                            setError(null)
                                        }
                                    }}
                                    autoFocus={true}
                                />
                            </div>
                            {error && (
                                <div className='GitHubPRStatus__form-error'>
                                    {error}
                                </div>
                            )}
                            <div className='GitHubPRStatus__form-actions'>
                                <button
                                    type='button'
                                    className='Button tertiary'
                                    onClick={() => {
                                        setShowForm(false)
                                        setPrUrl('')
                                        setError(null)
                                    }}
                                >
                                    <FormattedMessage
                                        id='GitHubPRStatus.cancel'
                                        defaultMessage='Cancel'
                                    />
                                </button>
                                <button
                                    type='button'
                                    className='Button filled'
                                    onClick={handleLoadPR}
                                    disabled={!prUrl.trim() || loadingPR}
                                >
                                    {loadingPR ? (
                                        <FormattedMessage
                                            id='GitHubPRStatus.loading'
                                            defaultMessage='Loading...'
                                        />
                                    ) : (
                                        <FormattedMessage
                                            id='GitHubPRStatus.add'
                                            defaultMessage='Add'
                                        />
                                    )}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}

            {!prDetails && readonly && (
                <div className='GitHubPRStatus__empty'>
                    <FormattedMessage
                        id='GitHubPRStatus.noPR'
                        defaultMessage='No PR tracked'
                    />
                </div>
            )}
        </div>
    )
}

export {GITHUB_PRS_PROPERTY_ID}
export default GitHubPRStatus
