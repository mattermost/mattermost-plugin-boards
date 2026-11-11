// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useEffect, useRef} from 'react'
import {generatePath, useHistory, useRouteMatch} from 'react-router-dom'

import {getBoards, getCurrentBoardId, isLoadingBoard} from '../../store/boards'
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
    const isLoading = useAppSelector(isLoadingBoard)
    const checkedViewBoardIdRef = useRef('')
    const lastViewIdOnOpenRef = useRef<string | undefined>(undefined)
    const teamId = match.params.teamId || UserSettings.lastTeamId || Constants.globalTeamId

    // BoardPage stores the URL's view as the last viewed one, even when that view was deleted.
    // Child effects run first, so this reads the value from before the board was opened.
    useEffect(() => {
        lastViewIdOnOpenRef.current = match.params.boardId ? UserSettings.lastViewId[match.params.boardId] : undefined
    }, [match.params.boardId])

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

        if (!boardId || boardId !== match.params.boardId || boardViews.length === 0 || isLoading) {
            return
        }

        const isBoardView = (id?: string): id is string => Boolean(id) && boardViews.some((view) => view.id === id)
        let viewID = match.params.viewId

        // A view from the URL or local storage can point to a deleted view, which renders an empty board.
        // Check only once per board: a newly created view reaches the URL before it reaches the store.
        const isStaleView = checkedViewBoardIdRef.current !== boardID && viewID !== '0' && !isBoardView(viewID)
        checkedViewBoardIdRef.current = boardID

        if (!viewID || viewID === '0' || isStaleView) {
            const lastViewID = isStaleView ? lastViewIdOnOpenRef.current : UserSettings.lastViewId[boardID]
            viewID = isBoardView(lastViewID) ? lastViewID : boardViews[0].id
            UserSettings.setLastViewId(boardID, viewID)
            dispatch(setCurrentView(viewID))

            const newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, viewId: viewID})
            history.replace({pathname: newPath, search: history.location.search})
        }
    }, [teamId, match.params.boardId, match.params.viewId, categories.length, boardViews.length, boardId, isLoading])

    return null
}

export default TeamToBoardAndViewRedirect
