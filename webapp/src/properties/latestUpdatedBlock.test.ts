// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createCard} from '../blocks/card'
import {createCommentBlock} from '../blocks/commentBlock'

import {definedBlocksForLatestUpdate, getLatestUpdatedBlock} from './latestUpdatedBlock'

describe('properties/latestUpdatedBlock', () => {
    test('excludes undefined content and comment blocks before sorting', () => {
        const card = createCard()
        card.updateAt = Date.parse('10 Jun 2021 16:22:00')

        expect(definedBlocksForLatestUpdate(card, undefined, undefined)).toEqual([card])
    })

    test('returns the block with the most recent updateAt', () => {
        const card = createCard()
        card.updateAt = Date.parse('10 Jun 2021 16:22:00')

        const comment = createCommentBlock()
        comment.updateAt = Date.parse('15 Jun 2021 16:22:00')

        expect(getLatestUpdatedBlock(card, undefined, comment)).toBe(comment)
    })
})
