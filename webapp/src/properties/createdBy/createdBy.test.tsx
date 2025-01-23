// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'

import {render} from '@testing-library/react'
import configureStore from 'redux-mock-store'

import {IUser} from '../../user'
import {createCard} from '../../blocks/card'
import {Board, IPropertyTemplate} from '../../blocks/board'

import {wrapIntl} from '../../testUtils'

import CreatedByProperty from './property'
import CreatedBy from './createdBy'

describe('properties/createdBy', () => {
    test('should match snapshot', () => {
        const card = createCard()
        card.createdBy = 'user-id-1'

        const mockStore = configureStore([])
        const store = mockStore({
            users: {
                boardUsers: {
                    'user-id-1': {username: 'username_1'} as IUser,
                },
            },
            clientConfig: {
                value: {
                    teammateNameDisplay: 'username',
                },
            },
        })

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreatedBy
                    property={new CreatedByProperty()}
                    board={{} as Board}
                    card={card}
                    readOnly={false}
                    propertyTemplate={{} as IPropertyTemplate}
                    propertyValue={''}
                    showEmptyPlaceholder={false}
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    test('should match snapshot as guest', () => {
        const card = createCard()
        card.createdBy = 'user-id-1'

        const mockStore = configureStore([])
        const store = mockStore({
            users: {
                boardUsers: {
                    'user-id-1': {username: 'username_1', is_guest: true} as IUser,
                },
            },
            clientConfig: {
                value: {
                    teammateNameDisplay: 'username',
                },
            },
        })

        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreatedBy
                    property={new CreatedByProperty()}
                    board={{} as Board}
                    card={card}
                    readOnly={false}
                    propertyTemplate={{} as IPropertyTemplate}
                    propertyValue={''}
                    showEmptyPlaceholder={false}
                />
            </ReduxProvider>,
        )

        const {container} = render(wrapIntl(component))
        expect(container).toMatchSnapshot()
    })
})
