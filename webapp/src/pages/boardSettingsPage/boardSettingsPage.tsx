// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react'
import {FormattedMessage} from 'react-intl'
import {useHistory, useRouteMatch} from 'react-router-dom'

import {useAppDispatch, useAppSelector} from '../../store/hooks'
import {getCurrentBoardId, setCurrent as setCurrentBoard, fetchBoardMembers} from '../../store/boards'
import {getCurrentTeam, setTeam} from '../../store/teams'
import {Permission} from '../../constants'
import {useHasPermissions} from '../../hooks/permissions'
import {initialLoad, loadBoardData} from '../../store/initialLoad'
import {getMe} from '../../store/users'
import octoClient from '../../octoClient'
import {UserSettings} from '../../userSettings'
import Button from '../../widgets/buttons/button'
import Sidebar from '../../components/sidebar/sidebar'
import BoardTemplateSelector from '../../components/boardTemplateSelector/boardTemplateSelector'

import './boardSettingsPage.scss'

const BoardSettingsPage = (): JSX.Element => {
    const history = useHistory()
    const match = useRouteMatch<{teamId: string, boardId: string}>()
    const dispatch = useAppDispatch()
    const currentBoardId = useAppSelector(getCurrentBoardId)
    const currentTeam = useAppSelector(getCurrentTeam)
    const me = useAppSelector(getMe)
    const [boardTemplateSelectorOpen, setBoardTemplateSelectorOpen] = useState(false)

    const teamId = match.params.teamId || currentTeam?.id || ''
    const boardId = match.params.boardId || currentBoardId || ''

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
            dispatch(initialLoad(boardId))
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
        // TODO: Implement save logic in subtasks
        // For now, just navigate back
        handleCancel()
    }, [handleCancel])

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
                            <p className='BoardSettingsPage__placeholder'>
                                <FormattedMessage
                                    id='BoardSettings.coming-soon'
                                    defaultMessage='Coming soon: Board name, icon, code, and description settings'
                                />
                            </p>
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

                    {/* Fixed footer with Save/Cancel buttons */}
                    <div className='BoardSettingsPage__footer'>
                        <Button
                            emphasis='tertiary'
                            size='medium'
                            onClick={handleCancel}
                        >
                            <FormattedMessage
                                id='BoardSettings.cancel'
                                defaultMessage='Cancel'
                            />
                        </Button>
                        <Button
                            filled={true}
                            size='medium'
                            onClick={handleSave}
                        >
                            <FormattedMessage
                                id='BoardSettings.save'
                                defaultMessage='Save'
                            />
                        </Button>
                    </div>
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
