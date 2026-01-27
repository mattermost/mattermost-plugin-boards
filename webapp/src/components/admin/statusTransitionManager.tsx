// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useRef} from 'react'

import {Board, IPropertyOption} from '../../blocks/board'
import octoClient from '../../octoClient'
import {Utils} from '../../utils'

import './statusTransitionManager.scss'

type StatusTransitionRule = {
    id: string
    boardId: string
    fromStatus: string
    toStatus: string
    allowed: boolean
    createAt: number
    updateAt: number
}

type TransitionMatrix = {
    [fromStatusId: string]: {
        [toStatusId: string]: boolean
    }
}

const StatusTransitionManager = () => {
    const [boards, setBoards] = useState<Board[]>([])
    const [selectedBoardId, setSelectedBoardId] = useState<string>('')
    const [statuses, setStatuses] = useState<IPropertyOption[]>([])
    const [matrix, setMatrix] = useState<TransitionMatrix>({})
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState('')
    const [hasChanges, setHasChanges] = useState(false)
    const currentRequestRef = useRef<string>('')

    useEffect(() => {
        loadBoards()
    }, [])

    useEffect(() => {
        if (selectedBoardId) {
            loadBoardData(selectedBoardId)
        }
    }, [selectedBoardId])

    const loadBoards = async () => {
        try {
            setLoading(true)
            setError('')

            const teams = await octoClient.getTeams()
            if (!teams || teams.length === 0) {
                setError('No teams found')
                setBoards([])
                return
            }

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

            const uniqueBoards = Array.from(
                new Map(allBoards.map(board => [board.id, board])).values()
            )

            setBoards(uniqueBoards)
            if (uniqueBoards.length > 0 && !selectedBoardId) {
                setSelectedBoardId(uniqueBoards[0].id)
            }
        } catch (err) {
            setError(`Failed to load boards: ${err}`)
            Utils.logError(`Failed to load boards: ${err}`)
        } finally {
            setLoading(false)
        }
    }

    const loadBoardData = async (boardId: string) => {
        // Set this request as the current one to prevent race conditions
        const requestId = boardId + Date.now()
        currentRequestRef.current = requestId

        try {
            setLoading(true)
            setError('')

            const board = boards.find(b => b.id === boardId)
            if (!board) {
                setError('Board not found')
                return
            }

            const statusProperty = board.cardProperties.find(
                prop => prop.type === 'select' && prop.name.toLowerCase() === 'status'
            )

            if (!statusProperty || !statusProperty.options || statusProperty.options.length === 0) {
                setStatuses([])
                setMatrix({})
                setError('This board has no Status property or no status options defined')
                return
            }

            const rules = await octoClient.getStatusTransitionRules(boardId)

            // Only update state if this is still the current request
            if (currentRequestRef.current !== requestId) {
                return
            }

            const newMatrix: TransitionMatrix = {}

            statusProperty.options.forEach(fromStatus => {
                newMatrix[fromStatus.id] = {}
                statusProperty.options.forEach(toStatus => {
                    const rule = rules.find(
                        r => r.fromStatus === fromStatus.id && r.toStatus === toStatus.id
                    )
                    newMatrix[fromStatus.id][toStatus.id] = rule ? rule.allowed : true
                })
            })

            setStatuses(statusProperty.options)
            setMatrix(newMatrix)
            setHasChanges(false)
        } catch (err) {
            setError(`Failed to load board data: ${err}`)
            Utils.logError(`Failed to load board data: ${err}`)
        } finally {
            setLoading(false)
        }
    }

    const handleMatrixChange = (fromStatusId: string, toStatusId: string, allowed: boolean) => {
        setMatrix(prev => ({
            ...prev,
            [fromStatusId]: {
                ...prev[fromStatusId],
                [toStatusId]: allowed
            }
        }))
        setHasChanges(true)
    }

    const handleSave = async () => {
        if (!selectedBoardId) {
            return
        }

        try {
            setSaving(true)
            setError('')

            const rules: StatusTransitionRule[] = []
            Object.keys(matrix).forEach(fromStatusId => {
                Object.keys(matrix[fromStatusId]).forEach(toStatusId => {
                    rules.push({
                        id: '',
                        boardId: selectedBoardId,
                        fromStatus: fromStatusId,
                        toStatus: toStatusId,
                        allowed: matrix[fromStatusId][toStatusId],
                        createAt: 0,
                        updateAt: 0
                    })
                })
            })

            const response = await octoClient.saveStatusTransitionRules(selectedBoardId, rules)
            if (!response.ok) {
                const errorText = await response.text()
                throw new Error(`Server returned ${response.status}: ${errorText || response.statusText}`)
            }
            setHasChanges(false)
        } catch (err) {
            setError(`Failed to save rules: ${err}`)
            Utils.logError(`Failed to save rules: ${err}`)
        } finally {
            setSaving(false)
        }
    }

    const handleCancel = () => {
        if (selectedBoardId) {
            loadBoardData(selectedBoardId)
        }
    }

    if (loading && boards.length === 0) {
        return (
            <div className='StatusTransitionManager'>
                <div className='StatusTransitionManager__loading'>
                    Loading boards...
                </div>
            </div>
        )
    }

    if (boards.length === 0) {
        return (
            <div className='StatusTransitionManager'>
                <div className='StatusTransitionManager__empty'>
                    No boards found
                </div>
            </div>
        )
    }

    return (
        <div className='StatusTransitionManager'>
            <div className='StatusTransitionManager__header'>
                <div className='StatusTransitionManager__board-selector'>
                    <label htmlFor='board-select'>Select Board:</label>
                    <select
                        id='board-select'
                        value={selectedBoardId}
                        onChange={(e) => setSelectedBoardId(e.target.value)}
                        disabled={saving}
                    >
                        {boards.map(board => (
                            <option key={board.id} value={board.id}>
                                {board.icon ? `${board.icon} ` : ''}{board.title}
                            </option>
                        ))}
                    </select>
                </div>
            </div>

            {error && (
                <div className='StatusTransitionManager__error'>
                    {error}
                </div>
            )}

            {loading && selectedBoardId && (
                <div className='StatusTransitionManager__loading'>
                    Loading board data...
                </div>
            )}

            {!loading && statuses.length > 0 && (
                <>
                    <div className='StatusTransitionManager__description'>
                        <p>
                            Configure which status transitions are allowed for cards in this board.
                            Check a box to allow transitioning from the row status to the column status.
                        </p>
                    </div>

                    <div className='StatusTransitionManager__matrix-wrapper'>
                        <table className='StatusTransitionManager__matrix'>
                            <thead>
                                <tr>
                                    <th className='StatusTransitionManager__corner'>From → To</th>
                                    {statuses.map(toStatus => (
                                        <th key={toStatus.id} className='StatusTransitionManager__header-cell'>
                                            <div
                                                className={`StatusTransitionManager__status-badge propColor${toStatus.color}`}
                                            >
                                                {toStatus.value}
                                            </div>
                                        </th>
                                    ))}
                                </tr>
                            </thead>
                            <tbody>
                                {statuses.map(fromStatus => (
                                    <tr key={fromStatus.id}>
                                        <td className='StatusTransitionManager__row-header'>
                                            <div
                                                className={`StatusTransitionManager__status-badge propColor${fromStatus.color}`}
                                            >
                                                {fromStatus.value}
                                            </div>
                                        </td>
                                        {statuses.map(toStatus => (
                                            <td key={toStatus.id} className='StatusTransitionManager__cell'>
                                                <input
                                                    type='checkbox'
                                                    checked={matrix[fromStatus.id]?.[toStatus.id] ?? true}
                                                    onChange={(e) => handleMatrixChange(
                                                        fromStatus.id,
                                                        toStatus.id,
                                                        e.target.checked
                                                    )}
                                                    disabled={saving}
                                                    aria-label={`Allow transition from ${fromStatus.value} to ${toStatus.value}`}
                                                />
                                            </td>
                                        ))}
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>

                    <div className='StatusTransitionManager__actions'>
                        <button
                            type='button'
                            className='StatusTransitionManager__button StatusTransitionManager__button--primary'
                            onClick={handleSave}
                            disabled={!hasChanges || saving}
                        >
                            {saving ? 'Saving...' : 'Save Changes'}
                        </button>
                        <button
                            type='button'
                            className='StatusTransitionManager__button StatusTransitionManager__button--secondary'
                            onClick={handleCancel}
                            disabled={!hasChanges || saving}
                        >
                            Cancel
                        </button>
                    </div>
                </>
            )}
        </div>
    )
}

export default StatusTransitionManager

