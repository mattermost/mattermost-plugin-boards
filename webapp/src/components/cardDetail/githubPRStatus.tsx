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

const GitHubPRStatus = (props: Props): JSX.Element | null => {
    // card is available for future use (e.g., auto-detect PR by branch name)
    const {readonly, hasBranch} = props
    const intl = useIntl()

    const [connectionStatus, setConnectionStatus] = useState<GitHubConnectedResponse | null>(null)
    const [loading, setLoading] = useState(true)
    const [showForm, setShowForm] = useState(false)
    const [prUrl, setPrUrl] = useState('')
    const [prDetails, setPrDetails] = useState<GitHubPRDetails | null>(null)
    const [loadingPR, setLoadingPR] = useState(false)
    const [error, setError] = useState<string | null>(null)

    // Check GitHub connection status
    useEffect(() => {
        loadConnectionStatus()
    }, [])

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

    // Don't show anything while loading, if not connected, or if no branch has been created
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

export default GitHubPRStatus
