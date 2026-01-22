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
            const fetchedBoards = await octoClient.getBoards()
            
            const boardsWithViews: BoardWithViews[] = fetchedBoards.map(board => ({
                board,
                views: [],
                code: board.code || '',
                isEditing: false,
                error: ''
            }))
            
            setBoards(boardsWithViews)
        } catch (err) {
            setError('Failed to load boards')
            Utils.logError('Failed to load boards:', err)
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
        setBoards(boards.map(b => 
            b.board.id === boardId 
                ? {...b, code: newCode, error: ''}
                : b
        ))
        props.setSaveNeeded()
    }

    const handleEditClick = (boardId: string) => {
        setBoards(boards.map(b => 
            b.board.id === boardId 
                ? {...b, isEditing: true}
                : b
        ))
    }

    const handleSaveClick = async (boardId: string) => {
        const boardData = boards.find(b => b.board.id === boardId)
        if (!boardData) return

        const validationError = validateCode(boardData.code)
        if (validationError) {
            setBoards(boards.map(b => 
                b.board.id === boardId 
                    ? {...b, error: validationError}
                    : b
            ))
            return
        }

        try {
            await octoClient.patchBoard(boardId, {code: boardData.code})
            setBoards(boards.map(b => 
                b.board.id === boardId 
                    ? {...b, isEditing: false, error: '', board: {...b.board, code: boardData.code}}
                    : b
            ))
        } catch (err) {
            setBoards(boards.map(b => 
                b.board.id === boardId 
                    ? {...b, error: 'Failed to save code'}
                    : b
            ))
            Utils.logError('Failed to save board code:', err)
        }
    }

    const handleCancelClick = (boardId: string) => {
        setBoards(boards.map(b => 
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

