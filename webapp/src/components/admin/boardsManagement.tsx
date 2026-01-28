// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useRef} from 'react'

import {Board, IPropertyOption} from '../../blocks/board'
import {BoardView} from '../../blocks/boardView'
import octoClient from '../../octoClient'
import {Utils} from '../../utils'

import './boardsManagement.scss'

type Props = {
    id: string
    label: string
    helpText?: string
    value: any
    disabled: boolean
    config: any
    license: any
    setByEnv: boolean
    onChange: (id: string, value: any) => void
    setSaveNeeded: () => void
    registerSaveAction: (action: () => Promise<void>) => void
    unRegisterSaveAction: (action: () => Promise<void>) => void
}

type BoardWithData = {
    board: Board
    views: BoardView[]
    code: string
    originalCode: string  // Track original value for dirty check
    isEditingCode: boolean
    isCodeDirty: boolean  // Track if code actually changed
    codeError: string
    statuses: IPropertyOption[]
    transitionMatrix: TransitionMatrix
    rulesLoadError: boolean  // Track if rules failed to load
    hasStatusChanges: boolean
}

type TransitionMatrix = {
    [fromStatusId: string]: {
        [toStatusId: string]: boolean
    }
}

type StatusTransitionRule = {
    id: string
    boardId: string
    fromStatus: string
    toStatus: string
    allowed: boolean
    createAt: number
    updateAt: number
}

const BoardsManagement = (props: Props) => {
    const [boards, setBoards] = useState<BoardWithData[]>([])
    const [activeTab, setActiveTab] = useState<string>('')
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState('')
    const saveActionRef = useRef<(() => Promise<void>) | null>(null)

    useEffect(() => {
        loadBoards()
    }, [])

    useEffect(() => {
        // Register save action with the admin console
        const saveAction = async () => {
            await handleSaveAll()
        }
        saveActionRef.current = saveAction
        props.registerSaveAction(saveAction)

        return () => {
            if (saveActionRef.current) {
                props.unRegisterSaveAction(saveActionRef.current)
            }
        }
    }, [boards])

    // Initialize a full transition matrix with all transitions allowed by default
    const initializeTransitionMatrix = (statuses: IPropertyOption[]): TransitionMatrix => {
        const matrix: TransitionMatrix = {}
        statuses.forEach(fromStatus => {
            matrix[fromStatus.id] = {}
            statuses.forEach(toStatus => {
                matrix[fromStatus.id][toStatus.id] = true
            })
        })
        return matrix
    }

    const loadBoards = async () => {
        try {
            setLoading(true)
            setError('')

            // Get all teams first
            const teams = await octoClient.getTeams()

            if (!teams || teams.length === 0) {
                setError('No teams found')
                setBoards([])
                return
            }

            // Get boards from all teams
            const allBoards: Board[] = []
            for (const team of teams) {
                try {
                    const regularBoards = await octoClient.getBoardsForTeam(team.id)
                    if (regularBoards && regularBoards.length > 0) {
                        const nonTemplateBoards = regularBoards.filter(board => !board.isTemplate)
                        allBoards.push(...nonTemplateBoards)
                    }
                } catch (teamErr) {
                    Utils.logError(`Failed to load boards for team ${team.id}: ${teamErr}`)
                }
            }

            if (allBoards.length === 0) {
                setBoards([])
                return
            }

            // Remove duplicates by board ID
            const uniqueBoards = Array.from(
                new Map(allBoards.map(board => [board.id, board])).values()
            )

            // Load data for each board
            const boardsWithData: BoardWithData[] = await Promise.all(
                uniqueBoards.map(async (board) => {
                    // Load status property and transition rules
                    const statusProperty = board.cardProperties.find(
                        prop => prop.type === 'select' && prop.name.toLowerCase() === 'status'
                    )
                    const statuses = statusProperty?.options || []

                    // Initialize with all transitions allowed by default
                    const transitionMatrix = initializeTransitionMatrix(statuses)
                    let rulesLoadError = false

                    if (statuses.length > 0) {
                        try {
                            const rules = await octoClient.getStatusTransitionRules(board.id)
                            // Apply loaded rules on top of the default matrix
                            rules.forEach(rule => {
                                if (transitionMatrix[rule.fromStatus] && transitionMatrix[rule.fromStatus][rule.toStatus] !== undefined) {
                                    transitionMatrix[rule.fromStatus][rule.toStatus] = rule.allowed
                                }
                            })
                        } catch (err) {
                            Utils.logError(`Failed to load status transition rules for board ${board.id}: ${err}`)
                            rulesLoadError = true
                        }
                    }

                    const currentCode = board.code || ''
                    return {
                        board,
                        views: [],
                        code: currentCode,
                        originalCode: currentCode,
                        isEditingCode: false,
                        isCodeDirty: false,
                        codeError: '',
                        statuses,
                        transitionMatrix,
                        rulesLoadError,
                        hasStatusChanges: false
                    }
                })
            )

            setBoards(boardsWithData)
            if (boardsWithData.length > 0 && !activeTab) {
                setActiveTab(boardsWithData[0].board.id)
            }
        } catch (err) {
            setError(`Failed to load boards: ${err}`)
            Utils.logError(`Failed to load boards: ${err}`)
        } finally {
            setLoading(false)
        }
    }

    const validateCode = (code: string): string => {
        if (!code) {
            return 'Code cannot be empty'
        }
        if (code.length > 10) {
            return 'Code must be 10 characters or less'
        }
        if (!/^[A-Za-z]/.test(code)) {
            return 'Code must start with a letter'
        }
        if (!/^[A-Za-z0-9]+$/.test(code)) {
            return 'Code must contain only letters and numbers'
        }
        return ''
    }

    const handleCodeChange = (boardId: string, newCode: string) => {
        const validationError = validateCode(newCode)
        setBoards(prev => prev.map(b => {
            if (b.board.id !== boardId) return b
            const isDirty = newCode !== b.originalCode
            return {...b, code: newCode, codeError: validationError, isCodeDirty: isDirty}
        }))
        props.setSaveNeeded()
    }

    const handleEditCodeClick = (boardId: string) => {
        setBoards(prev => prev.map(b =>
            b.board.id === boardId
                ? {...b, isEditingCode: true, codeError: ''}
                : b
        ))
    }

    const handleCancelCodeClick = (boardId: string) => {
        setBoards(prev => prev.map(b =>
            b.board.id === boardId
                ? {...b, code: b.originalCode, isEditingCode: false, isCodeDirty: false, codeError: ''}
                : b
        ))
    }

    const handleMatrixChange = (boardId: string, fromStatusId: string, toStatusId: string, allowed: boolean) => {
        setBoards(prev => prev.map(b => {
            if (b.board.id !== boardId) return b
            return {
                ...b,
                transitionMatrix: {
                    ...b.transitionMatrix,
                    [fromStatusId]: {
                        ...b.transitionMatrix[fromStatusId],
                        [toStatusId]: allowed
                    }
                },
                hasStatusChanges: true
            }
        }))
        props.setSaveNeeded()
    }

    const handleSaveAll = async () => {
        try {
            setSaving(true)
            setError('')

            for (const boardData of boards) {
                // Save board code only if it was actually changed (dirty)
                if (boardData.isCodeDirty) {
                    const validationError = validateCode(boardData.code)
                    if (validationError) {
                        setError(`Board "${boardData.board.title}": ${validationError}`)
                        return
                    }

                    const response = await octoClient.patchBoard(boardData.board.id, {code: boardData.code})
                    if (!response.ok) {
                        const errorText = await response.text().catch(() => '')
                        setError(`Failed to save code for board "${boardData.board.title}" (${response.status}): ${errorText || 'Unknown error'}`)
                        return
                    }
                }

                // Save status transition rules if they were changed
                // Block save if rules failed to load to prevent accidental overwrites
                if (boardData.hasStatusChanges && boardData.statuses.length > 0) {
                    if (boardData.rulesLoadError) {
                        setError(`Cannot save status transitions for board "${boardData.board.title}": Failed to load existing rules. Please refresh and try again.`)
                        return
                    }

                    const rules: StatusTransitionRule[] = []
                    Object.keys(boardData.transitionMatrix).forEach(fromStatusId => {
                        Object.keys(boardData.transitionMatrix[fromStatusId]).forEach(toStatusId => {
                            rules.push({
                                id: '',
                                boardId: boardData.board.id,
                                fromStatus: fromStatusId,
                                toStatus: toStatusId,
                                allowed: boardData.transitionMatrix[fromStatusId][toStatusId],
                                createAt: 0,
                                updateAt: 0
                            })
                        })
                    })

                    const response = await octoClient.saveStatusTransitionRules(boardData.board.id, rules)
                    if (!response.ok) {
                        const errorText = await response.text().catch(() => '')
                        setError(`Failed to save status transitions for board "${boardData.board.title}" (${response.status}): ${errorText || 'Unknown error'}`)
                        return
                    }
                }
            }

            // Reload boards to get fresh data
            await loadBoards()
        } catch (err) {
            setError(`Failed to save: ${err}`)
            Utils.logError(`Failed to save: ${err}`)
        } finally {
            setSaving(false)
        }
    }

    // Wrap in focalboard-body to apply styles (webpack loader prefixes all selectors with .focalboard-body)
    if (loading) {
        return <div className='focalboard-body'><div className='BoardsManagement loading'>Loading boards...</div></div>
    }

    if (error && boards.length === 0) {
        return <div className='focalboard-body'><div className='BoardsManagement error'>{error}</div></div>
    }

    if (boards.length === 0) {
        return (
            <div className='focalboard-body'>
                <div className='BoardsManagement'>
                    <div className='BoardsManagement__header'>
                        <h3 className='BoardsManagement__title'>Boards Management</h3>
                        <p className='BoardsManagement__help-text'>{props.helpText}</p>
                    </div>
                    <div className='BoardsManagement__empty'>
                        No boards found
                    </div>
                </div>
            </div>
        )
    }

    const activeBoardData = boards.find(b => b.board.id === activeTab)

    return (
        <div className='focalboard-body'>
            <div className='BoardsManagement'>
                <div className='BoardsManagement__header'>
                    <h3 className='BoardsManagement__title'>Boards Management</h3>
                    <p className='BoardsManagement__help-text'>{props.helpText}</p>
                </div>

                {error && (
                    <div className='BoardsManagement__error'>
                        {error}
                    </div>
                )}

                <div className='BoardsManagement__tabs'>
                    {boards.map(({board}) => (
                        <button
                            key={board.id}
                            type='button'
                            className={`BoardsManagement__tab ${activeTab === board.id ? 'active' : ''}`}
                            onClick={() => setActiveTab(board.id)}
                            disabled={saving}
                        >
                            {board.icon && <span className='BoardsManagement__tab-icon'>{board.icon}</span>}
                            {board.title}
                        </button>
                    ))}
                </div>

                {activeBoardData && (
                    <div className='BoardsManagement__content'>
                        {/* Board Code Section */}
                        <div className='BoardsManagement__section'>
                            <h4 className='BoardsManagement__section-title'>Board Code</h4>
                            <p className='BoardsManagement__section-help'>
                            Set a unique alphanumeric code for this board (1-10 characters, starting with a letter).
                            </p>
                            <div className='BoardsManagement__code-input'>
                                {activeBoardData.isEditingCode ? (
                                    <div className='BoardsManagement__code-edit'>
                                        <input
                                            type='text'
                                            value={activeBoardData.code}
                                            onChange={(e) => handleCodeChange(activeBoardData.board.id, e.target.value.toUpperCase())}
                                            maxLength={10}
                                            className={activeBoardData.codeError ? 'error' : ''}
                                            disabled={props.disabled || saving}
                                            placeholder='Enter code'
                                            aria-label='Board code'
                                        />
                                        <button
                                            type='button'
                                            onClick={() => handleCancelCodeClick(activeBoardData.board.id)}
                                            disabled={props.disabled || saving}
                                            className='BoardsManagement__button BoardsManagement__button--secondary'
                                        >
                                        Cancel
                                        </button>
                                        {activeBoardData.codeError && (
                                            <span className='BoardsManagement__error-text'>{activeBoardData.codeError}</span>
                                        )}
                                        <p className='BoardsManagement__hint'>
                                        Changes will be saved when you click the Save button at the bottom of the page.
                                        </p>
                                    </div>
                                ) : (
                                    <div className='BoardsManagement__code-display'>
                                        <span className='BoardsManagement__code-value'>
                                            {activeBoardData.code || <em>Not set</em>}
                                        </span>
                                        <button
                                            type='button'
                                            onClick={() => handleEditCodeClick(activeBoardData.board.id)}
                                            disabled={props.disabled || saving}
                                            className='BoardsManagement__button BoardsManagement__button--link'
                                        >
                                        Edit
                                        </button>
                                    </div>
                                )}
                            </div>
                        </div>

                        {/* Status Transitions Section */}
                        {activeBoardData.statuses.length > 0 ? (
                            <div className='BoardsManagement__section'>
                                <h4 className='BoardsManagement__section-title'>Status Transition Rules</h4>
                                <p className='BoardsManagement__section-help'>
                                Configure which status transitions are allowed for cards in this board.
                                Check a box to allow transitioning from the row status to the column status.
                                </p>
                                {activeBoardData.rulesLoadError && (
                                    <div className='BoardsManagement__warning'>
                                    ⚠️ Failed to load existing rules. Editing is disabled to prevent accidental overwrites.
                                    Please refresh the page to try again.
                                    </div>
                                )}
                                <div className='BoardsManagement__matrix-wrapper'>
                                    <table className='BoardsManagement__matrix'>
                                        <thead>
                                            <tr>
                                                <th className='BoardsManagement__corner'>From → To</th>
                                                {activeBoardData.statuses.map(toStatus => (
                                                    <th key={toStatus.id} className='BoardsManagement__header-cell'>
                                                        <div className={`BoardsManagement__status-badge propColor${toStatus.color}`}>
                                                            {toStatus.value}
                                                        </div>
                                                    </th>
                                                ))}
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {activeBoardData.statuses.map(fromStatus => (
                                                <tr key={fromStatus.id}>
                                                    <td className='BoardsManagement__row-header'>
                                                        <div className={`BoardsManagement__status-badge propColor${fromStatus.color}`}>
                                                            {fromStatus.value}
                                                        </div>
                                                    </td>
                                                    {activeBoardData.statuses.map(toStatus => (
                                                        <td key={toStatus.id} className='BoardsManagement__cell'>
                                                            <input
                                                                type='checkbox'
                                                                checked={activeBoardData.transitionMatrix[fromStatus.id]?.[toStatus.id] ?? true}
                                                                onChange={(e) => handleMatrixChange(
                                                                    activeBoardData.board.id,
                                                                    fromStatus.id,
                                                                    toStatus.id,
                                                                    e.target.checked
                                                                )}
                                                                disabled={props.disabled || saving || activeBoardData.rulesLoadError}
                                                                aria-label={`Allow transition from ${fromStatus.value} to ${toStatus.value}`}
                                                            />
                                                        </td>
                                                    ))}
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        ) : (
                            <div className='BoardsManagement__section'>
                                <h4 className='BoardsManagement__section-title'>Status Transition Rules</h4>
                                <p className='BoardsManagement__section-help BoardsManagement__section-help--muted'>
                                This board has no Status property or no status options defined.
                                </p>
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div>
    )
}

export default BoardsManagement
