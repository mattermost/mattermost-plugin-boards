// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Card} from '../../blocks/card'
import {GitHubRepository, GitHubConnectedResponse, GitHubBranch} from '../../github'
import octoClient from '../../octoClient'
import {sendFlashMessage} from '../flashMessages'
import IconButton from '../../widgets/buttons/iconButton'
import CompassIcon from '../../widgets/icons/compassIcon'
import CloseIcon from '../../widgets/icons/close'
import Button from '../../widgets/buttons/button'

import './githubBranchCreate.scss'

type Props = {
    card: Card
    readonly: boolean
    onBranchCreated?: (branch: GitHubBranch | null) => void
}

const GitHubBranchCreate = (props: Props): JSX.Element | null => {
    const {card, readonly, onBranchCreated} = props
    const intl = useIntl()

    const [connectionStatus, setConnectionStatus] = useState<GitHubConnectedResponse | null>(null)
    const [loading, setLoading] = useState(true)
    const [showForm, setShowForm] = useState(false)
    const [repositories, setRepositories] = useState<GitHubRepository[]>([])
    const [loadingRepos, setLoadingRepos] = useState(false)
    const [selectedRepo, setSelectedRepo] = useState<GitHubRepository | null>(null)
    const [branchName, setBranchName] = useState('')
    const [creating, setCreating] = useState(false)
    const [createdBranch, setCreatedBranch] = useState<GitHubBranch | null>(null)

    // Generate default branch name from card code
    const getDefaultBranchName = useCallback(() => {
        if (card.code) {
            // Convert card code to lowercase and create branch-friendly slug
            let slug = card.title
                .toLowerCase()
                .replace(/[^a-z0-9]+/g, '-')
                .replace(/^-|-$/g, '')
                .substring(0, 40)
            // Fallback if slug is empty (title was blank or all non-alphanumerics)
            if (!slug) {
                slug = 'task'
            }
            return `${card.code.toLowerCase()}/${slug}`
        }
        return ''
    }, [card.code, card.title])

    // Check GitHub connection status
    useEffect(() => {
        loadConnectionStatus()
    }, [])

    // Reset all state when card changes to prevent data leaking between cards
    useEffect(() => {
        setShowForm(false)
        setSelectedRepo(null)
        setBranchName('')
        setCreatedBranch(null)
        onBranchCreated?.(null)
    }, [card.id, onBranchCreated])

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

    const loadRepositories = async () => {
        try {
            setLoadingRepos(true)
            const repos = await octoClient.getGitHubRepositories()
            setRepositories(repos)
            if (repos.length > 0 && !selectedRepo) {
                setSelectedRepo(repos[0])
            }
        } catch (error) {
            console.error('Failed to load repositories:', error)
            sendFlashMessage({
                content: intl.formatMessage({
                    id: 'GitHubBranchCreate.repoError',
                    defaultMessage: 'Failed to load GitHub repositories',
                }),
                severity: 'low',
            })
        } finally {
            setLoadingRepos(false)
        }
    }

    const handleOpenForm = () => {
        setShowForm(true)
        setBranchName(getDefaultBranchName())
        loadRepositories()
    }

    const handleCloseForm = () => {
        setShowForm(false)
        setBranchName('')
        setSelectedRepo(null)
    }

    const handleCreateBranch = async () => {
        if (!selectedRepo || !branchName.trim()) {
            return
        }

        try {
            setCreating(true)
            const branch = await octoClient.createGitHubBranch({
                owner: selectedRepo.owner,
                repo: selectedRepo.name,
                branch_name: branchName.trim(),
            })

            if (branch) {
                setCreatedBranch(branch)
                onBranchCreated?.(branch)
                setShowForm(false)
                sendFlashMessage({
                    content: intl.formatMessage({
                        id: 'GitHubBranchCreate.success',
                        defaultMessage: 'Branch "{branchName}" created successfully',
                    }, {branchName: branchName.trim()}),
                    severity: 'high',
                })
            } else {
                sendFlashMessage({
                    content: intl.formatMessage({
                        id: 'GitHubBranchCreate.error',
                        defaultMessage: 'Failed to create branch',
                    }),
                    severity: 'low',
                })
            }
        } catch (error) {
            console.error('Failed to create branch:', error)
            sendFlashMessage({
                content: intl.formatMessage({
                    id: 'GitHubBranchCreate.error',
                    defaultMessage: 'Failed to create branch',
                }),
                severity: 'low',
            })
        } finally {
            setCreating(false)
        }
    }

    // Don't show anything while loading
    if (loading) {
        return null
    }

    // Show connect prompt if not connected to GitHub
    if (!connectionStatus?.connected) {
        return (
            <div className='GitHubBranchCreate'>
                <div className='GitHubBranchCreate__header'>
                    <div className='GitHubBranchCreate__title'>
                        <CompassIcon icon='github-circle'/>
                        <FormattedMessage
                            id='GitHubBranchCreate.title'
                            defaultMessage='GitHub Branch'
                        />
                    </div>
                </div>
                <div className='GitHubBranchCreate__connect-prompt'>
                    <FormattedMessage
                        id='GitHubBranchCreate.connectPrompt'
                        defaultMessage='Run /github connect in Mattermost to create branches'
                    />
                </div>
            </div>
        )
    }

    return (
        <div className='GitHubBranchCreate'>
            <div className='GitHubBranchCreate__header'>
                <div className='GitHubBranchCreate__title'>
                    <CompassIcon icon='source-branch'/>
                    <FormattedMessage
                        id='GitHubBranchCreate.title'
                        defaultMessage='GitHub Branch'
                    />
                </div>
            </div>

            {/* Created Branch Display */}
            {createdBranch && (
                <div className='GitHubBranchCreate__created'>
                    <div className='GitHubBranchCreate__branch'>
                        <div className='GitHubBranchCreate__branch-header'>
                            <CompassIcon icon='source-branch'/>
                            <span className='GitHubBranchCreate__branch-name'>
                                {createdBranch.ref.replace('refs/heads/', '')}
                            </span>
                        </div>
                        <a
                            href={createdBranch.url.replace('api.github.com/repos', 'github.com').replace('/git/refs/heads/', '/tree/')}
                            target='_blank'
                            rel='noopener noreferrer'
                            className='GitHubBranchCreate__branch-link'
                        >
                            <FormattedMessage
                                id='GitHubBranchCreate.viewOnGitHub'
                                defaultMessage='View on GitHub'
                            />
                        </a>
                    </div>
                </div>
            )}

            {/* Create Branch Button/Form */}
            {!createdBranch && !readonly && (
                <div className='GitHubBranchCreate__create'>
                    {!showForm ? (
                        <button
                            type='button'
                            className='GitHubBranchCreate__create-button'
                            onClick={handleOpenForm}
                        >
                            <CompassIcon icon='plus'/>
                            <FormattedMessage
                                id='GitHubBranchCreate.createButton'
                                defaultMessage='Create branch for this card'
                            />
                        </button>
                    ) : (
                        <div className='GitHubBranchCreate__form'>
                            <div className='GitHubBranchCreate__form-header'>
                                <FormattedMessage
                                    id='GitHubBranchCreate.formTitle'
                                    defaultMessage='Create GitHub Branch'
                                />
                                <IconButton
                                    className='GitHubBranchCreate__form-close'
                                    onClick={handleCloseForm}
                                    icon={<CloseIcon/>}
                                    title={intl.formatMessage({
                                        id: 'GitHubBranchCreate.closeForm',
                                        defaultMessage: 'Close',
                                    })}
                                    size='small'
                                />
                            </div>

                            {loadingRepos ? (
                                <div className='GitHubBranchCreate__form-loading'>
                                    <FormattedMessage
                                        id='GitHubBranchCreate.loadingRepos'
                                        defaultMessage='Loading repositories...'
                                    />
                                </div>
                            ) : (
                                <>
                                    <div className='GitHubBranchCreate__form-field'>
                                        <label htmlFor='repo-select'>
                                            <FormattedMessage
                                                id='GitHubBranchCreate.repository'
                                                defaultMessage='Repository'
                                            />
                                        </label>
                                        <select
                                            id='repo-select'
                                            className='GitHubBranchCreate__form-select'
                                            value={selectedRepo?.full_name || ''}
                                            onChange={(e) => {
                                                const repo = repositories.find((r) => r.full_name === e.target.value)
                                                setSelectedRepo(repo || null)
                                            }}
                                        >
                                            {repositories.map((repo) => (
                                                <option key={repo.id} value={repo.full_name}>
                                                    {repo.full_name}
                                                </option>
                                            ))}
                                        </select>
                                    </div>

                                    <div className='GitHubBranchCreate__form-field'>
                                        <label htmlFor='branch-name'>
                                            <FormattedMessage
                                                id='GitHubBranchCreate.branchName'
                                                defaultMessage='Branch Name'
                                            />
                                        </label>
                                        <input
                                            id='branch-name'
                                            type='text'
                                            className='GitHubBranchCreate__form-input'
                                            placeholder={intl.formatMessage({
                                                id: 'GitHubBranchCreate.branchPlaceholder',
                                                defaultMessage: 'e.g., feature/my-branch',
                                            })}
                                            value={branchName}
                                            onChange={(e) => setBranchName(e.target.value)}
                                            onKeyDown={(e) => {
                                                if (e.key === 'Enter' && !creating) {
                                                    handleCreateBranch()
                                                } else if (e.key === 'Escape') {
                                                    handleCloseForm()
                                                }
                                            }}
                                            autoFocus={true}
                                        />
                                        {selectedRepo && (
                                            <div className='GitHubBranchCreate__form-hint'>
                                                <FormattedMessage
                                                    id='GitHubBranchCreate.baseBranch'
                                                    defaultMessage='Base: {branch}'
                                                    values={{branch: selectedRepo.default_branch}}
                                                />
                                            </div>
                                        )}
                                    </div>

                                    <div className='GitHubBranchCreate__form-actions'>
                                        <Button
                                            onClick={handleCloseForm}
                                            emphasis='tertiary'
                                        >
                                            <FormattedMessage
                                                id='GitHubBranchCreate.cancel'
                                                defaultMessage='Cancel'
                                            />
                                        </Button>
                                        <Button
                                            onClick={handleCreateBranch}
                                            filled={true}
                                            disabled={!selectedRepo || !branchName.trim() || creating}
                                        >
                                            {creating ? (
                                                <FormattedMessage
                                                    id='GitHubBranchCreate.creating'
                                                    defaultMessage='Creating...'
                                                />
                                            ) : (
                                                <FormattedMessage
                                                    id='GitHubBranchCreate.create'
                                                    defaultMessage='Create Branch'
                                                />
                                            )}
                                        </Button>
                                    </div>
                                </>
                            )}
                        </div>
                    )}
                </div>
            )}

            {!createdBranch && readonly && (
                <div className='GitHubBranchCreate__empty'>
                    <FormattedMessage
                        id='GitHubBranchCreate.noBranch'
                        defaultMessage='No branch created'
                    />
                </div>
            )}
        </div>
    )
}

export default GitHubBranchCreate
