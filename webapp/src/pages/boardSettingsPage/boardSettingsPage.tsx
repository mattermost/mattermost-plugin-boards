// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react'
import {FormattedMessage} from 'react-intl'
import {useHistory, useRouteMatch} from 'react-router-dom'

import {useAppDispatch, useAppSelector} from '../../store/hooks'
import {getCurrentBoardId, getCurrentBoard, setCurrent as setCurrentBoard, fetchBoardMembers, updateBoards} from '../../store/boards'
import {getCurrentTeam, setTeam} from '../../store/teams'
import {Permission} from '../../constants'
import {useHasPermissions} from '../../hooks/permissions'
import {initialLoad, loadBoardData} from '../../store/initialLoad'
import {getMe} from '../../store/users'
import {getCategoryOfBoard, getHiddenBoardIDs, updateBoardCategories} from '../../store/sidebar'
import octoClient from '../../octoClient'
import {UserSettings} from '../../userSettings'
import mutator from '../../mutator'
import {Board} from '../../blocks/board'
import Sidebar from '../../components/sidebar/sidebar'
import BoardTemplateSelector from '../../components/boardTemplateSelector/boardTemplateSelector'

import GeneralSection from './generalSection'
import BoardSettingsFooter from './boardSettingsFooter'

import './boardSettingsPage.scss'

const BoardSettingsPage = (): JSX.Element => {
    const history = useHistory()
    const match = useRouteMatch<{teamId: string, boardId: string}>()
    const dispatch = useAppDispatch()
    const currentBoardId = useAppSelector(getCurrentBoardId)
    const currentTeam = useAppSelector(getCurrentTeam)
    const board = useAppSelector(getCurrentBoard)
    const me = useAppSelector(getMe)
    const hiddenBoardIDs = useAppSelector(getHiddenBoardIDs)

    const teamId = match.params.teamId || currentTeam?.id || ''
    const boardId = match.params.boardId || currentBoardId || ''
    const isHidden = hiddenBoardIDs.includes(boardId)

    // Use route boardId (not currentBoardId) for consistent category resolution
    const category = useAppSelector(getCategoryOfBoard(boardId))
    const [boardTemplateSelectorOpen, setBoardTemplateSelectorOpen] = useState(false)

    // Initialize team and board data (same as BoardPage)
    useEffect(() => {
        if (teamId) {
            UserSettings.lastTeamId = teamId
            octoClient.teamId = teamId
            dispatch(setTeam(teamId))
        }
    }, [teamId])

    useEffect(() => {
        if (boardId) {
            dispatch(initialLoad())
            dispatch(setCurrentBoard(boardId))
        }
    }, [teamId, boardId, me?.id])

    useEffect(() => {
        if (boardId && me) {
            dispatch(loadBoardData(boardId))
            dispatch(fetchBoardMembers({teamId, boardId}))
        }
    }, [teamId, boardId, me?.id])

    // Check if user has admin permissions
    const hasAdminPermission = useHasPermissions(teamId, boardId, [Permission.ManageBoardType])

    const handleCancel = useCallback(() => {
        // Navigate back to the board
        if (teamId && boardId) {
            history.push(`/team/${teamId}/${boardId}`)
        }
    }, [history, teamId, boardId])

    const handleSave = useCallback(() => {
        // Navigate back after save
        handleCancel()
    }, [handleCancel])

    const handleBoardChange = useCallback(async (updatedBoard: Board) => {
        // Update board using mutator
        if (!board) {
            return
        }
        await mutator.updateBoard(updatedBoard, board, 'update board settings')
        dispatch(updateBoards([updatedBoard]))
    }, [board, dispatch])

    const handleHideBoard = useCallback(async () => {
        if (!category || !me) {
            return
        }
        await octoClient.hideBoard(category.id, boardId)
        dispatch(updateBoardCategories([
            {
                boardID: boardId,
                categoryID: category.id,
                hidden: true,
            },
        ]))
        // Navigate back after hiding
        handleCancel()
    }, [category, boardId, me, dispatch, handleCancel])

    const handleShowBoard = useCallback(async () => {
        if (!category || !me) {
            return
        }
        await octoClient.unhideBoard(category.id, boardId)
        dispatch(updateBoardCategories([
            {
                boardID: boardId,
                categoryID: category.id,
                hidden: false,
            },
        ]))
    }, [category, boardId, me, dispatch])

    const handleDeleteBoard = useCallback(async () => {
        if (!board) {
            return
        }
        await mutator.deleteBoard(board, 'delete board')
        // Navigate to team page after deletion
        if (teamId) {
            history.push(`/team/${teamId}`)
        }
    }, [board, teamId, history])

    // Redirect if user doesn't have permission
    if (!hasAdminPermission) {
        return (
            <div className='BoardSettingsPage error'>
                <FormattedMessage
                    id='BoardSettings.no-permission'
                    defaultMessage='You do not have permission to access board settings.'
                />
            </div>
        )
    }

    // Show loading state if board is not loaded yet
    if (!board) {
        return (
            <div className='BoardSettingsPage'>
                <div className='BoardSettingsPage__wrapper'>
                    <Sidebar
                        onBoardTemplateSelectorOpen={() => setBoardTemplateSelectorOpen(true)}
                        onBoardTemplateSelectorClose={() => setBoardTemplateSelectorOpen(false)}
                        activeBoardId={boardId}
                    />
                    <div className='BoardSettingsPage__content'>
                        <div className='BoardSettingsPage__header'>
                            <h1>
                                <FormattedMessage
                                    id='BoardSettings.title'
                                    defaultMessage='Board Settings'
                                />
                            </h1>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className='BoardSettingsPage'>
            <div className='BoardSettingsPage__wrapper'>
                {/* Left sidebar with boards/views list */}
                <Sidebar
                    onBoardTemplateSelectorOpen={() => setBoardTemplateSelectorOpen(true)}
                    onBoardTemplateSelectorClose={() => setBoardTemplateSelectorOpen(false)}
                    activeBoardId={boardId}
                />

                {/* Main settings content */}
                <div className='BoardSettingsPage__content'>
                    <div className='BoardSettingsPage__header'>
                        <h1>
                            <FormattedMessage
                                id='BoardSettings.title'
                                defaultMessage='Board Settings'
                            />
                        </h1>
                        <div className='BoardSettingsPage__breadcrumb'>
                            <button
                                className='BoardSettingsPage__back-button'
                                onClick={handleCancel}
                            >
                                <FormattedMessage
                                    id='BoardSettings.back-to-board'
                                    defaultMessage='← Back to board'
                                />
                            </button>
                        </div>
                    </div>

                    <div className='BoardSettingsPage__sections'>
                        {/* Section 1: General (IT-369) */}
                        <div className='BoardSettingsPage__section'>
                            <h2>
                                <FormattedMessage
                                    id='BoardSettings.general-section'
                                    defaultMessage='General'
                                />
                            </h2>
                            <GeneralSection
                                board={board}
                                onBoardChange={handleBoardChange}
                            />
                        </div>

                        {/* Section 2: Views Management (IT-370) */}
                        <div className='BoardSettingsPage__section'>
                            <h2>
                                <FormattedMessage
                                    id='BoardSettings.views-section'
                                    defaultMessage='Views Management'
                                />
                            </h2>
                            <p className='BoardSettingsPage__placeholder'>
                                <FormattedMessage
                                    id='BoardSettings.views-coming-soon'
                                    defaultMessage='Coming soon: Table of views with management options'
                                />
                            </p>
                        </div>

                        {/* Section 3: Card Properties and Options (IT-371) */}
                        <div className='BoardSettingsPage__section'>
                            <h2>
                                <FormattedMessage
                                    id='BoardSettings.properties-section'
                                    defaultMessage='Card Properties and Options'
                                />
                            </h2>
                            <p className='BoardSettingsPage__placeholder'>
                                <FormattedMessage
                                    id='BoardSettings.properties-coming-soon'
                                    defaultMessage='Coming soon: Card properties management'
                                />
                            </p>
                        </div>
                    </div>

                    {/* Fixed footer with actions */}
                    <BoardSettingsFooter
                        board={board}
                        isHidden={isHidden}
                        onHideBoard={handleHideBoard}
                        onShowBoard={handleShowBoard}
                        onDeleteBoard={handleDeleteBoard}
                        onCancel={handleCancel}
                        onSave={handleSave}
                    />
                </div>
            </div>

            {/* Board template selector modal (triggered from sidebar "+ Add board") */}
            {boardTemplateSelectorOpen &&
                <BoardTemplateSelector
                    title={
                        <FormattedMessage
                            id='BoardTemplateSelector.plugin.no-content-title'
                            defaultMessage='Create a board'
                        />
                    }
                    description={
                        <FormattedMessage
                            id='BoardTemplateSelector.plugin.no-content-description'
                            defaultMessage='Add a board to the sidebar to start managing your project tasks, meetings, and more.'
                        />
                    }
                    onClose={() => setBoardTemplateSelectorOpen(false)}
                />
            }
        </div>
    )
}

export default React.memo(BoardSettingsPage)
