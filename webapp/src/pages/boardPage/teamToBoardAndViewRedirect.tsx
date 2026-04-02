// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useEffect, useRef} from 'react'
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

    // Clear skip flag once the URL includes a boardId. The component stays mounted when
    // navigating from /team/:teamId to /team/:teamId/:boardId, so unmount cleanup would not run.
    useEffect(() => {
        if (match.params.boardId) {
            sessionStorage.removeItem(Constants.sessionStorageSkipBoardRedirectKey)
        }
    }, [match.params.boardId])

    useEffect(() => {
        let boardID = match.params.boardId

        // Check if we should skip all auto-redirects (e.g., after error page)
        const skipRedirect = sessionStorage.getItem(Constants.sessionStorageSkipBoardRedirectKey) === 'true'

        if (!match.params.boardId) {
            // Skip auto-redirect if flag is set
            if (skipRedirect) {
                return
            }

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
                const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, boardId: boardID, viewID: undefined})
                history.replace(newPath)

                // return from here because the loadBoardData() call
                // will fetch the data to be used below. We'll
                // use it in the next render cycle.
                return
            }
        }

        let viewID = match.params.viewId

        // when a view isn't open,
        // but the data is available, try opening a view
        if ((!viewID || viewID === '0') && boardId && boardId === match.params.boardId && boardViews && boardViews.length > 0) {
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
                const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, viewId: viewID})
                history.replace(newPath)
            }
        }
    }, [teamId, match.params.boardId, match.params.viewId, categories.length, boardViews.length, boardId])

    return null
}

export default TeamToBoardAndViewRedirect
