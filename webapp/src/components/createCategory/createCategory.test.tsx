// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import {render} from '@testing-library/react'

import userEvent from '@testing-library/user-event'

import thunk from 'redux-thunk'

import {Provider as ReduxProvider} from 'react-redux'

import {mocked} from 'jest-mock'

import {mockStateStore, wrapIntl} from '../../testUtils'

import {IUser} from '../../user'

import mutator from '../../mutator'

import CreateCategory from './createCategory'

jest.mock('../../mutator')
const mockedMutator = mocked(mutator, true)

describe('components/createCategory/CreateCategory', () => {
    const me: IUser = {
        id: 'user-id-1',
        username: 'username_1',
        email: '',
        nickname: '',
        firstname: '',
        lastname: '',
        props: {},
        create_at: 0,
        update_at: 0,
        is_bot: false,
        roles: 'system_user',
        is_guest: false,
    }

    const state = {
        teams: {
            current: {id: 'team-id', title: 'Test Team'},
        },
        users: {
            me,
        },
    }
    const store = mockStateStore([thunk], state)

    it('base case should match snapshot', () => {
        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreateCategory
                    onClose={jest.fn()}
                    title={
                        <span>{'title'}</span>
                    }
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        expect(container).toMatchSnapshot()
    })

    it('should call onClose on being closed', () => {
        const onCloseHandler = jest.fn()
        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreateCategory
                    onClose={onCloseHandler}
                    title={
                        <span>{'title'}</span>
                    }
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        const cancelBtn = container.querySelector('.createCategoryActions > .Button.danger')
        expect(cancelBtn).toBeTruthy()
        userEvent.click(cancelBtn as Element)
        expect(onCloseHandler).toBeCalledTimes(1)

        const closeBtn = container.querySelector('.toolbar .dialog__close')
        expect(closeBtn).toBeTruthy()
        userEvent.click(closeBtn as Element)
        expect(onCloseHandler).toBeCalledTimes(2)
    })

    it('should call onCreate on pressing enter', () => {
        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreateCategory
                    onClose={jest.fn()}
                    title={
                        <span>{'title'}</span>
                    }
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        const inputField = container.querySelector('.categoryNameInput')
        expect(inputField).toBeTruthy()
        userEvent.type(inputField as Element, 'category name{enter}')
        expect(mockedMutator.createCategory).toBeCalledWith({
            name: 'category name',
            teamID: 'team-id',
            userID: 'user-id-1',
        })
    })

    it('should show initial value', () => {
        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreateCategory
                    initialValue='Dwight prank ideas'
                    onClose={jest.fn()}
                    title={
                        <span>{'title'}</span>
                    }
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        const inputField = container.querySelector('.categoryNameInput')
        expect(inputField).toBeTruthy()
        expect((inputField as HTMLInputElement).value).toBe('Dwight prank ideas')
    })

    it('should clear input field on clicking clear icon', () => {
        const component = wrapIntl(
            <ReduxProvider store={store}>
                <CreateCategory
                    initialValue='Dunder Mifflin'
                    onClose={jest.fn()}
                    title={
                        <span>{'title'}</span>
                    }
                />
            </ReduxProvider>,
        )

        const {container} = render(component)
        const inputField = container.querySelector('.categoryNameInput')
        expect(inputField).toBeTruthy()
        expect((inputField as HTMLInputElement).value).toBe('Dunder Mifflin')

        const clearBtn = container.querySelector('.clearBtn')
        expect(clearBtn).toBeTruthy()
        userEvent.click(clearBtn as Element)
        expect((inputField as HTMLInputElement).value).toBe('')
    })
})
