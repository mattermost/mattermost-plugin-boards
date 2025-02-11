// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'

import {render} from '@testing-library/react'
import configureStore from 'redux-mock-store'

import {createCard} from '../../blocks/card'
import {IUser} from '../../user'
import {wrapIntl} from '../../testUtils'

import {createBoard, IPropertyTemplate} from '../../blocks/board'

import {createCommentBlock} from '../../blocks/commentBlock'

import UpdatedByProperty from './property'
import UpdatedBy from './updatedBy'

describe('properties/updatedBy', () => {
    test('should match snapshot', () => {
        const card = createCard()
        card.id = 'card-id-1'
        card.modifiedBy = 'user-id-1'

        const board = createBoard()
        const comment = createCommentBlock()
        comment.modifiedBy = 'user-id-1'
        comment.parentId = 'card-id-1'

        const mockStore = configureStore([])
        const store = mockStore({
            users: {
                boardUsers: {
                    'user-id-1': {username: 'username_1'} as IUser,
                },
            },
            comments: {
                comments: {
                    [comment.id]: comment,
                },
                commentsByCard: {
                    [card.id]: [comment],
                },
            },
            clientConfig: {
                value: {
                    teammateNameDisplay: 'username',
                },
            },
        })

        const component = (
            <ReduxProvider store={store}>
                <UpdatedBy
                    property={new UpdatedByProperty()}
                    card={card}
                    board={board}
                    propertyTemplate={{} as IPropertyTemplate}
                    propertyValue={''}
                    readOnly={false}
                    showEmptyPlaceholder={false}
                />
            </ReduxProvider>
        )

        const {container} = render(wrapIntl(component))
        expect(container).toMatchSnapshot()
    })
})
