// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react'

import {Board} from '../../blocks/board'
import {BoardView} from '../../blocks/boardView'
import octoClient from '../../octoClient'
import {Utils} from '../../utils'

import './boardCodesManager.scss'

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

type BoardWithViews = {
    board: Board
    views: BoardView[]
    code: string
    isEditing: boolean
    error: string
}

const BoardCodesManager = (props: Props) => {
    const [boards, setBoards] = useState<BoardWithViews[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')

    useEffect(() => {
        loadBoards()
    }, [])

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
                    // Get template boards for this team
                    const teamTemplates = await octoClient.getTeamTemplates(team.id)
                    if (teamTemplates && teamTemplates.length > 0) {
                        allBoards.push(...teamTemplates)
                    }

                    // Get regular boards for this team using direct fetch
                    // octoClient.getBoards() doesn't accept teamId, so we use fetch directly
                    const response = await fetch(octoClient.getBaseURL() + `/api/v2/teams/${team.id}/boards`, {
                        headers: octoClient.headers()
                    })
                    if (response.status === 200) {
                        const regularBoards = await response.json() as Board[]
                        if (regularBoards && regularBoards.length > 0) {
                            allBoards.push(...regularBoards)
                        }
                    }
                } catch (teamErr) {
                    Utils.logError(`Failed to load boards for team ${team.id}: ${teamErr}`)
                    // Continue with other teams
                }
            }

            if (allBoards.length === 0) {
                setBoards([])
                return
            }

            const boardsWithViews: BoardWithViews[] = allBoards.map(board => ({
                board,
                views: [],
                code: board.code || '',
                isEditing: false,
                error: ''
            }))

            setBoards(boardsWithViews)
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
        // Validate in real-time
        const validationError = validateCode(newCode)

        setBoards(prev => prev.map(b =>
            b.board.id === boardId
                ? {...b, code: newCode, error: validationError}
                : b
        ))
        // Don't call setSaveNeeded - this component handles its own save via per-row Save button
    }

    const handleEditClick = (boardId: string) => {
        setBoards(prev => prev.map(b =>
            b.board.id === boardId
                ? {...b, isEditing: true, error: ''}
                : b
        ))
    }

    const handleSaveClick = async (boardId: string) => {
        // Use functional form to avoid stale closure
        setBoards(prev => {
            const boardData = prev.find(b => b.board.id === boardId)
            if (!boardData) return prev

            const validationError = validateCode(boardData.code)
            if (validationError) {
                return prev.map(b =>
                    b.board.id === boardId
                        ? {...b, error: validationError}
                        : b
                )
            }

            // Perform async save
            octoClient.patchBoard(boardId, {code: boardData.code})
                .then(response => {
                    // Check response status
                    if (!response.ok) {
                        setBoards(prev2 => prev2.map(b =>
                            b.board.id === boardId
                                ? {...b, error: `Failed to save code (HTTP ${response.status})`}
                                : b
                        ))
                        Utils.logError(`Failed to save board code: HTTP ${response.status}`)
                        return
                    }

                    // Success - close edit mode and update board
                    setBoards(prev2 => prev2.map(b =>
                        b.board.id === boardId
                            ? {...b, isEditing: false, error: '', board: {...b.board, code: boardData.code}}
                            : b
                    ))
                })
                .catch(err => {
                    setBoards(prev2 => prev2.map(b =>
                        b.board.id === boardId
                            ? {...b, error: 'Failed to save code'}
                            : b
                    ))
                    Utils.logError(`Failed to save board code: ${err}`)
                })

            // Return current state unchanged (async operation will update later)
            return prev
        })
    }

    const handleCancelClick = (boardId: string) => {
        setBoards(prev => prev.map(b =>
            b.board.id === boardId
                ? {...b, code: b.board.code || '', isEditing: false, error: ''}
                : b
        ))
    }

    if (loading) {
        return <div className='BoardCodesManager loading'>Loading boards...</div>
    }

    if (error) {
        return <div className='BoardCodesManager error'>{error}</div>
    }

    return (
        <div className='BoardCodesManager'>
            <div className='BoardCodesManager__header'>
                <p className='BoardCodesManager__help-text'>{props.helpText}</p>
            </div>

            {boards.length === 0 ? (
                <div className='BoardCodesManager__empty'>
                    No boards found
                </div>
            ) : (
                <div className='BoardCodesManager__list'>
                    <table className='BoardCodesManager__table'>
                        <thead>
                            <tr>
                                <th>Board Title</th>
                                <th>Team</th>
                                <th>Code</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {boards.map(({board, code, isEditing, error}) => (
                                <tr key={board.id} className='BoardCodesManager__row'>
                                    <td className='BoardCodesManager__title'>
                                        {board.icon ? <span className='BoardCodesManager__icon'>{board.icon}</span> : null}
                                        {board.title}
                                    </td>
                                    <td className='BoardCodesManager__team'>
                                        {board.teamId}
                                    </td>
                                    <td className='BoardCodesManager__code'>
                                        {isEditing ? (
                                            <div className='BoardCodesManager__code-edit'>
                                                <input
                                                    type='text'
                                                    value={code}
                                                    onChange={(e) => handleCodeChange(board.id, e.target.value.toUpperCase())}
                                                    maxLength={10}
                                                    className={error ? 'error' : ''}
                                                    disabled={props.disabled}
                                                    placeholder='Enter code'
                                                    aria-label='Board code'
                                                />
                                                {error && <span className='BoardCodesManager__error'>{error}</span>}
                                            </div>
                                        ) : (
                                            <span className='BoardCodesManager__code-display'>
                                                {code || <em>Not set</em>}
                                            </span>
                                        )}
                                    </td>
                                    <td className='BoardCodesManager__actions'>
                                        {isEditing ? (
                                            <>
                                                <button
                                                    type='button'
                                                    onClick={() => handleSaveClick(board.id)}
                                                    disabled={props.disabled}
                                                    className='btn btn-primary btn-sm'
                                                >
                                                    Save
                                                </button>
                                                <button
                                                    type='button'
                                                    onClick={() => handleCancelClick(board.id)}
                                                    disabled={props.disabled}
                                                    className='btn btn-link btn-sm'
                                                >
                                                    Cancel
                                                </button>
                                            </>
                                        ) : (
                                            <button
                                                type='button'
                                                onClick={() => handleEditClick(board.id)}
                                                disabled={props.disabled}
                                                className='btn btn-link btn-sm'
                                            >
                                                Edit
                                            </button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    )
}

export default BoardCodesManager

