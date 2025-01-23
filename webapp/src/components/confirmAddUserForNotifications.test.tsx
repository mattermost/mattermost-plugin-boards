// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import '@testing-library/jest-dom'
import {render} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'

import {MemberRole} from '../blocks/board'

import {wrapDNDIntl} from '../testUtils'
import {IUser} from '../user'

import ConfirmAddUserForNotifications from './confirmAddUserForNotifications'

describe('/components/confirmAddUserForNotifications', () => {
    it('should match snapshot', async () => {
        const result = render(
            wrapDNDIntl(
                <ConfirmAddUserForNotifications
                    allowManageBoardRoles={true}
                    minimumRole={MemberRole.Editor}
                    user={{id: 'fake-user-id', username: 'fake-username'} as IUser}
                    onConfirm={jest.fn()}
                    onClose={jest.fn()}
                />,
            ),
        )
        expect(result.container).toMatchSnapshot()
    })

    it('confirm button click, run onConfirm Function once', () => {
        const onConfirm = jest.fn()

        const result = render(
            wrapDNDIntl(
                <ConfirmAddUserForNotifications
                    allowManageBoardRoles={true}
                    minimumRole={MemberRole.Editor}
                    user={{id: 'fake-user-id', username: 'fake-username'} as IUser}
                    onConfirm={onConfirm}
                    onClose={jest.fn()}
                />,
            ),
        )
        userEvent.click(result.getByTitle('Add to board'))
        expect(onConfirm).toBeCalledTimes(1)
    })

    it('cancel button click runs onClose function', () => {
        const onClose = jest.fn()

        const result = render(
            wrapDNDIntl(
                <ConfirmAddUserForNotifications
                    allowManageBoardRoles={true}
                    minimumRole={MemberRole.Editor}
                    user={{id: 'fake-user-id', username: 'fake-username'} as IUser}
                    onConfirm={jest.fn()}
                    onClose={onClose}
                />,
            ),
        )
        userEvent.click(result.getByTitle('Cancel'))
        expect(onClose).toBeCalledTimes(1)
    })
})
