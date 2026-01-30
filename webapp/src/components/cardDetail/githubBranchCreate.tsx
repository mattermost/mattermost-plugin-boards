// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Card} from '../../blocks/card'
import {GitHubRepository, GitHubConnectedResponse, GitHubBranch, GitHubBranchInfo} from '../../github'
import octoClient from '../../octoClient'
import {sendFlashMessage} from '../flashMessages'
import IconButton from '../../widgets/buttons/iconButton'
import CompassIcon from '../../widgets/icons/compassIcon'
import CloseIcon from '../../widgets/icons/close'
import Button from '../../widgets/buttons/button'
import {UserSettings} from '../../userSettings'

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
    const [branches, setBranches] = useState<GitHubBranchInfo[]>([])
    const [loadingBranches, setLoadingBranches] = useState(false)
    const [baseBranch, setBaseBranch] = useState<string>('')
    const [showBaseBranchPicker, setShowBaseBranchPicker] = useState(false)
    const [showBranchPicker, setShowBranchPicker] = useState(false)
    const [chooseExistingMode, setChooseExistingMode] = useState(false)

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

    // Load saved branch from card fields on mount/card change
    useEffect(() => {
        const saved = card.fields.githubBranch
        if (saved && typeof saved.ref === 'string' && typeof saved.url === 'string') {
            const branch: GitHubBranch = {
                ref: saved.ref,
                url: saved.url,
                object: {sha: '', type: 'commit'},
            }
            setCreatedBranch(branch)
            onBranchCreated?.(branch)
        }
    }, [card.id, card.fields.githubBranch, onBranchCreated])

    // Reset all state when card changes to prevent data leaking between cards
    useEffect(() => {
        setShowForm(false)
        setSelectedRepo(null)
        setBranchName('')
        // Only clear if card doesn't have a saved branch
        if (!card.fields.githubBranch) {
            setCreatedBranch(null)
            onBranchCreated?.(null)
        }
    }, [card.id, card.fields.githubBranch, onBranchCreated])

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
                // Try to restore last selected repository from localStorage
                const lastRepoFullName = UserSettings.lastGitHubRepo
                const lastRepo = lastRepoFullName ? repos.find((r) => r.full_name === lastRepoFullName) : null
                setSelectedRepo(lastRepo || repos[0])
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

    const loadBranches = async (repo: GitHubRepository) => {
        try {
            setLoadingBranches(true)
            const branchList = await octoClient.getGitHubBranches(repo.owner, repo.name)
            setBranches(branchList)
            // Always reset base branch when repository changes to avoid
            // carrying over a branch name that doesn't exist in the new repo.
            setBaseBranch(repo.default_branch || '')
        } catch (error) {
            console.error('Failed to load branches:', error)
            sendFlashMessage({
                content: intl.formatMessage({
                    id: 'GitHubBranchCreate.branchesError',
                    defaultMessage: 'Failed to load branches',
                }),
                severity: 'low',
            })
        } finally {
            setLoadingBranches(false)
        }
    }

    // Load branches when repository changes
    useEffect(() => {
        if (selectedRepo) {
            loadBranches(selectedRepo)
        }
    }, [selectedRepo])

    const handleOpenForm = () => {
        setShowForm(true)
        setBranchName(getDefaultBranchName())
        loadRepositories()
    }

    const handleCloseForm = () => {
        setShowForm(false)
        setBranchName('')
        setSelectedRepo(null)
        setBranches([])
        setBaseBranch('')
        setShowBaseBranchPicker(false)
        setShowBranchPicker(false)
        setChooseExistingMode(false)
    }

    // Check if the current branch name matches an existing branch
    const isExistingBranch = useCallback(
        (name: string): GitHubBranchInfo | undefined => {
            return branches.find((b) => b.name === name.trim())
        },
        [branches],
    )

    const connectExistingBranch = async (existingBranch: GitHubBranchInfo) => {
        if (!selectedRepo) {
            return
        }

        try {
            setCreating(true)

            // Build a GitHubBranch object from the existing branch info
            const branch: GitHubBranch = {
                ref: `refs/heads/${existingBranch.name}`,
                url: `https://api.github.com/repos/${selectedRepo.owner}/${selectedRepo.name}/git/refs/heads/${encodeURIComponent(existingBranch.name)}`,
                object: {sha: existingBranch.sha, type: 'commit'},
            }

            // Save branch info to card fields
            const blockPatch = {
                updatedFields: {
                    githubBranch: {
                        ref: branch.ref,
                        url: branch.url,
                        repo: selectedRepo.full_name,
                        connectedAt: new Date().toISOString(),
                    },
                },
            }

            try {
                await octoClient.patchBlock(card.boardId, card.id, blockPatch)
            } catch (saveError) {
                console.error('Failed to save branch to card:', saveError)
                sendFlashMessage({
                    content: intl.formatMessage({
                        id: 'GitHubBranchCreate.connectSaveError',
                        defaultMessage: 'Failed to connect branch to card. Try refreshing.',
                    }),
                    severity: 'low',
                })
                return
            }

            setCreatedBranch(branch)
            onBranchCreated?.(branch)
            setShowForm(false)
            setChooseExistingMode(false)

            sendFlashMessage({
                content: intl.formatMessage({
                    id: 'GitHubBranchCreate.connectSuccess',
                    defaultMessage: 'Branch "{branchName}" connected successfully',
                }, {branchName: existingBranch.name}),
                severity: 'high',
            })
        } catch (error) {
            console.error('Failed to connect branch:', error)
            sendFlashMessage({
                content: intl.formatMessage({
                    id: 'GitHubBranchCreate.connectError',
                    defaultMessage: 'Failed to connect branch',
                }),
                severity: 'low',
            })
        } finally {
            setCreating(false)
        }
    }

    const handleCreateOrConnectBranch = async () => {
        if (!selectedRepo || !branchName.trim()) {
            return
        }

        // In "choose existing" mode, check if branch name matches an existing branch
        if (chooseExistingMode) {
            const existing = isExistingBranch(branchName)
            if (existing) {
                // Connect existing branch — no GitHub API create call
                await connectExistingBranch(existing)
                return
            }
            // Name was edited and no longer matches — fall through to create
        }

        try {
            setCreating(true)
            const branch = await octoClient.createGitHubBranch({
                owner: selectedRepo.owner,
                repo: selectedRepo.name,
                branch_name: branchName.trim(),
                base_branch: baseBranch || undefined,
            })

            if (branch) {
                // Save branch info to card fields before updating UI
                const blockPatch = {
                    updatedFields: {
                        githubBranch: {
                            ref: branch.ref,
                            url: branch.url,
                            repo: selectedRepo.full_name,
                            createdAt: new Date().toISOString(),
                        },
                    },
                }

                try {
                    await octoClient.patchBlock(card.boardId, card.id, blockPatch)
                } catch (saveError) {
                    console.error('Failed to save branch to card:', saveError)
                    sendFlashMessage({
                        content: intl.formatMessage({
                            id: 'GitHubBranchCreate.saveError',
                            defaultMessage: 'Branch created on GitHub but failed to save to card. Try refreshing.',
                        }),
                        severity: 'low',
                    })
                }

                setCreatedBranch(branch)
                onBranchCreated?.(branch)
                setShowForm(false)
                setChooseExistingMode(false)

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
                    <CompassIcon icon='github-circle'/>
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
                                                // Save selected repository to localStorage
                                                if (repo) {
                                                    UserSettings.lastGitHubRepo = repo.full_name
                                                }
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
                                        <div className='GitHubBranchCreate__form-field-header'>
                                            <label htmlFor='branch-name'>
                                                <FormattedMessage
                                                    id='GitHubBranchCreate.branchName'
                                                    defaultMessage='Branch Name'
                                                />
                                            </label>
                                            {selectedRepo && !loadingBranches && branches.length > 0 && (
                                                <button
                                                    type='button'
                                                    className='GitHubBranchCreate__choose-existing-link'
                                                    onClick={() => {
                                                        const newState = !showBranchPicker
                                                        setShowBranchPicker(newState)
                                                        if (!newState) {
                                                            // Closing picker — exit choose existing mode if branch was not selected
                                                            setChooseExistingMode(false)
                                                        }
                                                    }}
                                                >
                                                    <FormattedMessage
                                                        id={showBranchPicker ? 'GitHubBranchCreate.newBranch' : 'GitHubBranchCreate.chooseExisting'}
                                                        defaultMessage={showBranchPicker ? 'new branch' : 'choose existing'}
                                                    />
                                                </button>
                                            )}
                                        </div>
                                        <input
                                            id='branch-name'
                                            type='text'
                                            className='GitHubBranchCreate__form-input'
                                            placeholder={intl.formatMessage({
                                                id: 'GitHubBranchCreate.branchPlaceholder',
                                                defaultMessage: 'e.g., feature/my-branch',
                                            })}
                                            value={branchName}
                                            onChange={(e) => {
                                                setBranchName(e.target.value)
                                                // Update chooseExistingMode based on whether typed name matches an existing branch
                                                if (chooseExistingMode) {
                                                    const exists = branches.some((b) => b.name === e.target.value.trim())
                                                    setChooseExistingMode(exists)
                                                }
                                            }}
                                            onKeyDown={(e) => {
                                                if (e.key === 'Enter' && !creating) {
                                                    handleCreateOrConnectBranch()
                                                } else if (e.key === 'Escape') {
                                                    handleCloseForm()
                                                }
                                            }}
                                            autoFocus={true}
                                        />
                                        {showBranchPicker && selectedRepo && branches.length > 0 && (
                                            <div className='GitHubBranchCreate__branch-picker'>
                                                <div className='GitHubBranchCreate__branch-picker-header'>
                                                    <FormattedMessage
                                                        id='GitHubBranchCreate.selectBranch'
                                                        defaultMessage='Select an existing branch'
                                                    />
                                                </div>
                                                <div className='GitHubBranchCreate__branch-list'>
                                                    {branches.map((branch) => (
                                                        <button
                                                            key={branch.name}
                                                            type='button'
                                                            className='GitHubBranchCreate__branch-item'
                                                            onClick={() => {
                                                                setBranchName(branch.name)
                                                                setShowBranchPicker(false)
                                                                setChooseExistingMode(true)
                                                            }}
                                                        >
                                                            {branch.name}
                                                        </button>
                                                    ))}
                                                </div>
                                            </div>
                                        )}
                                        {selectedRepo && !showBaseBranchPicker && (
                                            <div className='GitHubBranchCreate__form-hint'>
                                                <FormattedMessage
                                                    id='GitHubBranchCreate.baseBranchLabel'
                                                    defaultMessage='Base: '
                                                />
                                                <button
                                                    type='button'
                                                    className='GitHubBranchCreate__base-branch-link'
                                                    onClick={() => setShowBaseBranchPicker(true)}
                                                >
                                                    {baseBranch || selectedRepo.default_branch}
                                                </button>
                                            </div>
                                        )}
                                        {showBaseBranchPicker && selectedRepo && branches.length > 0 && (
                                            <div className='GitHubBranchCreate__form-hint'>
                                                <label htmlFor='base-branch-select'>
                                                    <FormattedMessage
                                                        id='GitHubBranchCreate.baseBranchSelect'
                                                        defaultMessage='BASE BRANCH'
                                                    />
                                                </label>
                                                <select
                                                    id='base-branch-select'
                                                    className='GitHubBranchCreate__base-branch-select'
                                                    value={baseBranch || selectedRepo.default_branch}
                                                    onChange={(e) => {
                                                        setBaseBranch(e.target.value)
                                                        setShowBaseBranchPicker(false)
                                                    }}
                                                    autoFocus={true}
                                                >
                                                    {branches.map((branch) => (
                                                        <option key={branch.name} value={branch.name}>
                                                            {branch.name}
                                                        </option>
                                                    ))}
                                                </select>
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
                                            onClick={handleCreateOrConnectBranch}
                                            filled={true}
                                            disabled={!selectedRepo || !branchName.trim() || creating}
                                        >
                                            {creating ? (
                                                chooseExistingMode && isExistingBranch(branchName) ? (
                                                    <FormattedMessage
                                                        id='GitHubBranchCreate.connecting'
                                                        defaultMessage='Connecting...'
                                                    />
                                                ) : (
                                                    <FormattedMessage
                                                        id='GitHubBranchCreate.creating'
                                                        defaultMessage='Creating...'
                                                    />
                                                )
                                            ) : (
                                                chooseExistingMode && isExistingBranch(branchName) ? (
                                                    <FormattedMessage
                                                        id='GitHubBranchCreate.connect'
                                                        defaultMessage='Connect Branch'
                                                    />
                                                ) : (
                                                    <FormattedMessage
                                                        id='GitHubBranchCreate.create'
                                                        defaultMessage='Create Branch'
                                                    />
                                                )
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
