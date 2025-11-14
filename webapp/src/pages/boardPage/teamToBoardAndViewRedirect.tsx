// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useEffect, useMemo} from 'react'
import {generatePath, useHistory, useRouteMatch} from 'react-router-dom'

import {getBoards, getCurrentBoardId} from '../../store/boards'
import {setCurrent as setCurrentView, getCurrentBoardViews} from '../../store/views'
import {useAppSelector, useAppDispatch} from '../../store/hooks'
import {UserSettings} from '../../userSettings'
import {Utils} from '../../utils'
import {getSidebarCategories} from '../../store/sidebar'
import {Constants} from '../../constants'

const TeamToBoardAndViewRedirect = (): null => {
    const boardId = useAppSelector(getCurrentBoardId)
    const boardViews = useAppSelector(getCurrentBoardViews)
    const dispatch = useAppDispatch()
    const history = useHistory()
    const match = useRouteMatch<{boardId: string, viewId: string, cardId?: string, teamId?: string}>()
    const categories = useAppSelector(getSidebarCategories)
    const boards = useAppSelector(getBoards)
    const teamId = match.params.teamId || UserSettings.lastTeamId || Constants.globalTeamId

    const boardCount = useMemo(() => Object.keys(boards).length, [boards])
    const categoryCount = useMemo(() => categories.length, [categories])

    useEffect(() => {
        if (match.params.boardId) {
            return
        }

        if (boardCount === 0 && categoryCount === 0) {
            return
        }

        let boardID: string | undefined = undefined
        
        const lastBoardId = UserSettings.lastBoardId[teamId]
        if (lastBoardId) {
            const board = boards[lastBoardId]
            if (board && (board.teamId === teamId || board.teamId === Constants.globalTeamId)) {
                boardID = lastBoardId
            } else {
                UserSettings.setLastBoardID(teamId, null)
            }
        }

        if (!boardID && categoryCount > 0) {
            for (const category of categories) {
                for (const boardMetadata of category.boardMetadata) {
                    const board = boards[boardMetadata.boardID]
                    // Pick the first category board that exists, is not hidden, and belongs to this team
                    if (!boardMetadata.hidden && board && (board.teamId === teamId || board.teamId === Constants.globalTeamId)) {
                        boardID = boardMetadata.boardID
                        break
                    }
                }
                if (boardID) {
                    break
                }
            }
        }

        if (boardID) {
            const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, boardId: boardID, viewID: undefined})
            history.replace(newPath)
            return
        }

    }, [teamId, match.params.boardId, boardCount, categoryCount, boards, categories, history, match.path, match.params])

    const viewCount = useMemo(() => boardViews.length, [boardViews])

    useEffect(() => {
        const viewID = match.params.viewId

        // when a view isn't open,
        // but the data is available, try opening a view
        if ((!viewID || viewID === '0') && boardId && boardId === match.params.boardId && viewCount > 0) {
            // most recent view gets the first preference
            let selectedViewID = UserSettings.lastViewId[boardId]
            if (selectedViewID) {
                UserSettings.setLastViewId(boardId, selectedViewID)
                dispatch(setCurrentView(selectedViewID))
            } else if (boardViews.length > 0) {
                // if most recent view is unavailable, pick the first view
                selectedViewID = boardViews[0].id
                UserSettings.setLastViewId(boardId, selectedViewID)
                dispatch(setCurrentView(selectedViewID))
            }

            if (selectedViewID) {
                const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, viewId: selectedViewID})
                history.replace(newPath)
            }
        }
    }, [match.params.boardId, match.params.viewId, viewCount, boardId, boardViews, dispatch, history, match.path, match.params])

    return null
}

export default TeamToBoardAndViewRedirect
