// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useMemo, useState} from 'react'
import {
    Router,
    Redirect,
    Switch,
    useRouteMatch,
    useHistory,
    generatePath,
    useLocation,
} from 'react-router-dom'
import {createBrowserHistory, History} from 'history'

import {IAppWindow} from './types'
import BoardPage from './pages/boardPage/boardPage'
import WelcomePage from './pages/welcome/welcomePage'
import ErrorPage from './pages/errorPage'
import AccessDeniedPage from './pages/accessDeniedPage'
import {Utils} from './utils'
import octoClient from './octoClient'
import {setGlobalError, getGlobalError} from './store/globalError'
import {useAppSelector, useAppDispatch} from './store/hooks'
import {ErrorId} from './errors'
import {getFirstTeam, fetchTeams, Team} from './store/teams'
import {getSidebarCategories, CategoryBoards} from './store/sidebar'
import {getMySortedBoards} from './store/boards'
import {UserSettings} from './userSettings'
import FBRoute from './route'

declare let window: IAppWindow

function HomeToCurrentTeam(props: {path: string, exact: boolean}) {
    return (
        <FBRoute
            path={props.path}
            exact={props.exact}
            loginRequired={true}
            component={() => {
                const firstTeam = useAppSelector<Team|null>(getFirstTeam)
                const dispatch = useAppDispatch()
                const [teamsFetched, setTeamsFetched] = useState(false)
                const [teamsFetchError, setTeamsFetchError] = useState(false)
                const categories = useAppSelector<CategoryBoards[]>(getSidebarCategories)
                const myBoards = useAppSelector(getMySortedBoards)
                const history = useHistory()

                useEffect(() => {
                    const loadTeams = async () => {
                        try {
                            const result = await dispatch(fetchTeams())
                            if (fetchTeams.rejected.match(result)) {
                                setTeamsFetchError(true)
                            } else if (fetchTeams.fulfilled.match(result)) {
                                const teams = result.payload as Team[]
                                const windowTeamID = (window.getCurrentTeamId && window.getCurrentTeamId()) || ''
                                const lastTeamID = UserSettings.lastTeamId
                                
                                if (teams.length === 0 && !windowTeamID && !lastTeamID) {
                                    setTeamsFetchError(true)
                                }
                            }
                            setTeamsFetched(true)
                        } catch (error) {
                            setTeamsFetchError(true)
                            setTeamsFetched(true)
                        }
                    }
                    loadTeams()
                }, [dispatch])

                let teamID = (window.getCurrentTeamId && window.getCurrentTeamId()) || ''
                const lastTeamID = UserSettings.lastTeamId
                const hasNoTeamId = !teamID && !firstTeam && !lastTeamID
                
                if (teamsFetchError && hasNoTeamId) {
                    history.replace('/error?id=unknown')
                    return null
                }
                
                if (hasNoTeamId && !teamsFetched) {
                    return <></>
                }
                
                teamID = teamID || lastTeamID || firstTeam?.id || ''
                
                if (!teamID && teamsFetched) {
                    history.replace('/error?id=unknown')
                    return null
                }

                const validBoardIds = new Set(myBoards.filter((b) => !b.deleteAt).map((b) => b.id))

                if (UserSettings.lastBoardId) {
                    const lastBoardID = UserSettings.lastBoardId[teamID]
                    const lastViewID = UserSettings.lastViewId[lastBoardID]

                    if (lastBoardID) {
                        if (!validBoardIds.has(lastBoardID)) {
                            let fallbackBoardId: string | null = null
                            for (const category of categories) {
                                const visible = category.boardMetadata.find((m) => !m.hidden && validBoardIds.has(m.boardID))
                                if (visible) {
                                    fallbackBoardId = visible.boardID
                                    break
                                }
                            }

                            if (fallbackBoardId) {
                                UserSettings.setLastBoardID(teamID, fallbackBoardId)
                                return <Redirect to={`/team/${teamID}/${fallbackBoardId}`}/>
                            }

                            UserSettings.setLastBoardID(teamID, null)
                            return <Redirect to={`/team/${teamID}`}/>
                        }

                        if (lastBoardID && lastViewID) {
                            return <Redirect to={`/team/${teamID}/${lastBoardID}/${lastViewID}`}/>
                        }
                        if (lastBoardID) {
                            return <Redirect to={`/team/${teamID}/${lastBoardID}`}/>
                        }
                    }
                }

                if (validBoardIds.size === 0) {
                    return <Redirect to={`/team/${teamID}`}/>
                }

                let firstBoardId: string | null = null
                for (const category of categories) {
                    const visible = category.boardMetadata.find((m) => !m.hidden && validBoardIds.has(m.boardID))
                    if (visible) {
                        firstBoardId = visible.boardID
                        break
                    }
                }

                if (firstBoardId) {
                    return <Redirect to={`/team/${teamID}/${firstBoardId}`}/>
                }

                return <Redirect to={`/team/${teamID}`}/>
            }}
        />
    )
}

function WorkspaceToTeamRedirect() {
    const match = useRouteMatch<{boardId: string, viewId: string, cardId?: string, workspaceId?: string}>()
    const queryParams = new URLSearchParams(useLocation().search)
    const history = useHistory()
    useEffect(() => {
        octoClient.getBoard(match.params.boardId).then((board) => {
            if (board) {
                let newPath = generatePath(match.path.replace('/workspace/:workspaceId', '/team/:teamId'), {
                    teamId: board?.teamId,
                    boardId: board?.id,
                    viewId: match.params.viewId,
                    cardId: match.params.cardId,
                })
                if (queryParams) {
                    newPath += '?' + queryParams
                }
                history.replace(newPath)
            }
        })
    }, [])
    return null
}

function GlobalErrorRedirect() {
    const globalError = useAppSelector<string>(getGlobalError)
    const dispatch = useAppDispatch()
    const history = useHistory()
    const location = useLocation()

    useEffect(() => {
        if (globalError) {
            // Don't redirect if we're already on an error page
            if (location.pathname === '/error' || location.pathname === '/access-denied') {
                dispatch(setGlobalError(''))
                return
            }

            dispatch(setGlobalError(''))
            // Redirect to access denied page for access denied errors
            if (globalError === ErrorId.AccessDenied || globalError === ErrorId.InvalidReadOnlyBoard) {
                const currentPath = location.pathname + location.search
                history.replace(`/access-denied?r=${encodeURIComponent(currentPath)}`)
            } else {
                history.replace(`/error?id=${globalError}`)
            }
        }
    }, [globalError, history, location, dispatch])

    return null
}

type Props = {
    history?: History<unknown>
}

const FocalboardRouter = (props: Props): JSX.Element => {

    let browserHistory: History<unknown>
    if (props.history) {
        browserHistory = props.history
    } else {
        browserHistory = useMemo(() => {
            return createBrowserHistory({basename: Utils.getFrontendBaseURL()})
        }, [])
    }

    useEffect(() => {
        if (window.frontendBaseURL) {
            browserHistory.replace(window.location.pathname.replace(window.frontendBaseURL, ''))
        }
    }, [])

    return (
        <Router history={browserHistory}>
            <GlobalErrorRedirect/>
            <Switch>
                <HomeToCurrentTeam
                    path='/'
                    exact={true}
                />
                <FBRoute
                    exact={true}
                    path='/welcome'
                >
                    <WelcomePage/>
                </FBRoute>

                <FBRoute path='/error'>
                    <ErrorPage/>
                </FBRoute>
                <FBRoute path='/access-denied'>
                    <AccessDeniedPage/>
                </FBRoute>
                <FBRoute path={['/team/:teamId/new/:channelId']}>
                    <BoardPage new={true}/>
                </FBRoute>

                <FBRoute path={['/team/:teamId/shared/:boardId?/:viewId?/:cardId?', '/shared/:boardId?/:viewId?/:cardId?']}>
                    <BoardPage readonly={true}/>
                </FBRoute>

                <FBRoute
                    loginRequired={true}
                    path='/board/:boardId?/:viewId?/:cardId?'
                    getOriginalPath={({params: {boardId, viewId, cardId}}) => {
                        return `/board/${Utils.buildOriginalPath('', boardId, viewId, cardId)}`
                    }}
                >
                    <BoardPage/>
                </FBRoute>
                <FBRoute path={['/workspace/:workspaceId/shared/:boardId?/:viewId?/:cardId?', '/workspace/:workspaceId/:boardId?/:viewId?/:cardId?']}>
                    <WorkspaceToTeamRedirect/>
                </FBRoute>
                <FBRoute
                    loginRequired={true}
                    path='/team/'
                    exact={true}
                    component={() => {
                        return <Redirect to='/'/>
                    }}
                />
                <FBRoute
                    loginRequired={true}
                    path='/team/:teamId/:boardId?/:viewId?/:cardId?'
                    getOriginalPath={({params: {teamId, boardId, viewId, cardId}}) => {
                        return `/team/${Utils.buildOriginalPath(teamId, boardId, viewId, cardId)}`
                    }}
                >
                    <BoardPage/>
                </FBRoute>
            </Switch>
        </Router>
    )
}

export default React.memo(FocalboardRouter)
