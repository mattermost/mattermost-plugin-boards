// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useEffect} from 'react'
import {useParams, useHistory} from 'react-router-dom'
import {generatePath} from 'react-router'

import octoClient from '../octoClient'
import {Utils} from '../utils'

const TaskRedirect = (): null => {
    const {code} = useParams<{code: string}>()
    const history = useHistory()

    useEffect(() => {
        const fetchCard = async () => {
            try {
                const response = await octoClient.getCardByCode(code)
                const {teamId, boardId, viewId, cardId} = response

                const newPath = generatePath('/team/:teamId/:boardId/:viewId/:cardId', {
                    teamId,
                    boardId,
                    viewId,
                    cardId,
                })

                history.replace(newPath)
            } catch (error) {
                Utils.logError(`Failed to load card by code ${code}: ${error}`)
                history.replace('/error?id=not-found')
            }
        }

        fetchCard()
    }, [code, history])

    return null
}

export default TaskRedirect

