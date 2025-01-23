// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import {useEffect, useMemo} from 'react'

import {Board} from '../blocks/board'

import octoClient from '../octoClient'

import {useAppDispatch, useAppSelector} from '../store/hooks'
import {fetchGlobalTemplates, getGlobalTemplates} from '../store/globalTemplates'
import {getTemplates} from '../store/boards'

import {Constants} from '../constants'

export const useGetAllTemplates = () => {
    const dispatch = useAppDispatch()
    const globalTemplates = useAppSelector<Board[]>(getGlobalTemplates) || []

    useEffect(() => {
        if (octoClient.teamId !== Constants.globalTeamId && globalTemplates.length === 0) {
            dispatch(fetchGlobalTemplates())
        }
    }, [octoClient.teamId])

    const unsortedTemplates = useAppSelector(getTemplates)
    const templates = useMemo(() => Object.values(unsortedTemplates).sort((a: Board, b: Board) => a.createAt - b.createAt), [unsortedTemplates])

    return useMemo(() => globalTemplates.concat(templates), [globalTemplates])
}
