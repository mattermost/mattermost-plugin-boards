// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {render} from '@testing-library/react'
import '@testing-library/jest-dom'

import {wrapIntl} from '../testUtils'

import GuestBadge from './guestBadge'

describe('widgets/guestBadge', () => {
    test('should match the snapshot on show', () => {
        const {container} = render(wrapIntl(<GuestBadge show={true}/>))
        expect(container).toMatchSnapshot()
    })

    test('should match the snapshot on hide', () => {
        const {container} = render(wrapIntl(<GuestBadge show={false}/>))
        expect(container).toMatchInlineSnapshot('<div />')
    })
})
