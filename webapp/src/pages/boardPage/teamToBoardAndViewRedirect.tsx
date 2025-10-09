// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useEffect} from 'react'
import {generatePath, useHistory, useRouteMatch} from 'react-router-dom'

import {getBoards, getCurrentBoardId, setCurrent as setCurrentBoard} from '../../store/boards'
import {setCurrent as setCurrentView, getCurrentBoardViews} from '../../store/views'
import {useAppSelector, useAppDispatch} from '../../store/hooks'
import {UserSettings} from '../../userSettings'
import {Utils} from '../../utils'
import {getSidebarCategories, fetchSidebarCategories} from '../../store/sidebar'
import {Constants} from '../../constants'
import {getCurrentTeam} from '../../store/teams'
import {loadBoardData} from '../../store/initialLoad'

const TeamToBoardAndViewRedirect = (): null => {
    const boardId = useAppSelector(getCurrentBoardId)
    const boardViews = useAppSelector(getCurrentBoardViews)
    const dispatch = useAppDispatch()
    const history = useHistory()
    const match = useRouteMatch<{boardId: string, viewId: string, cardId?: string, teamId?: string}>()
    const categories = useAppSelector(getSidebarCategories)
    const boards = useAppSelector(getBoards)
    const team = useAppSelector(getCurrentTeam)
    const teamId = match.params.teamId || UserSettings.lastTeamId || Constants.globalTeamId

    useEffect(() => {
        // Check if we're showing template selector (avoid redirect loop)
        const urlParams = new URLSearchParams(window.location.search)
        if (urlParams.get('template') === 'true') {
            return
        }
        
        // Load categories if team is available but categories are empty
        if (team && categories.length === 0) {
            dispatch(fetchSidebarCategories(team.id))
            return
        }
        
        // Wait for team and categories to be loaded
        if (!team || categories.length === 0) {
            return
        }
        
        let boardID = match.params.boardId
        
        if (!boardID) {
            // first preference is for last visited board
            boardID = UserSettings.lastBoardId[teamId]

            // if last visited board is unavailable, use the first board in categories list
            if (!boardID && categories.length > 0) {
                let goToBoardID: string | null = null

                for (const category of categories) {
                    for (const boardMetadata of category.boardMetadata) {
                        // pick the first category board that exists and is not hidden
                        if (!boardMetadata.hidden && boards[boardMetadata.boardID]) {
                            goToBoardID = boardMetadata.boardID
                            break
                        }
                    }
                }

                // there may even be no boards at all
                if (goToBoardID) {
                    boardID = goToBoardID
                }
            }

            if (boardID) {
                // Search for view when board is found
                let viewID = match.params.viewId

                // Load board data if boardViews is empty
                if (!boardViews || boardViews.length === 0) {
                    console.log("Loading board data for board:", boardID)
                    // Set current board first so getCurrentBoardViews can work
                    dispatch(setCurrentBoard(boardID))
                    dispatch(loadBoardData(boardID))
                    return
                }

                // when a view isn't open,
                // but the data is available, try opening a view
                if ((!viewID || viewID === '0') && boardViews && boardViews.length > 0) {
                    // most recent view gets the first preference
                    viewID = UserSettings.lastViewId[boardID]
                    
                    if (viewID) {
                        UserSettings.setLastViewId(boardID, viewID)
                        dispatch(setCurrentView(viewID))
                    } else if (boardViews.length > 0) {
                        // if most recent view is unavailable, pick the first view
                        viewID = boardViews[0].id
                        UserSettings.setLastViewId(boardID, viewID)
                        dispatch(setCurrentView(viewID))
                    }

                    if (viewID) {
                        const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, boardId: boardID, viewId: viewID})
                        history.replace(newPath)
                        return
                    }
                }

                // If no viewID in localStorage and no view found above, load the first view
                if ((!viewID || viewID === '0') && boardViews && boardViews.length > 0) {
                    viewID = boardViews[0].id
                    UserSettings.setLastViewId(boardID, viewID)
                    dispatch(setCurrentView(viewID))
                    
                    const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, boardId: boardID, viewId: viewID})
                    history.replace(newPath)
                    return
                }

                // If no view found or view logic didn't redirect, redirect to board without view
                const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, boardId: boardID, viewID: undefined})
                history.replace(newPath)
                return
            } else {
                // No boardID found, redirect to template selector
                const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, boardId: undefined, viewID: undefined}) + '?template=true'
                history.replace(newPath)
                return
            }
        }
    }, [teamId, match.params.boardId, match.params.viewId, categories.length, boardViews.length, boardId, team])

    return null
}

export default TeamToBoardAndViewRedirect
