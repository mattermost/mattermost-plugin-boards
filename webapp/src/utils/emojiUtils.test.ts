// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getValidEmojiData} from './emojiUtils'

describe('getValidEmojiData', () => {
    it('should return valid emoji data for a known emoji', () => {
        const emoji = '😄' // smiling face
        const result = getValidEmojiData(emoji)

        expect(result).not.toBeNull()
        expect(result?.native).toBe(emoji)
        expect(result?.id).toBe('smile')
    })

    it('should return null for an invalid emoji', () => {
        const invalidEmoji = 'not-an-emoji'
        const result = getValidEmojiData(invalidEmoji)

        expect(result).toBeNull()
    })

    it('should return correct metadata for a complex emoji (e.g. skin tone)', () => {
        const emoji = '👍🏽'
        const result = getValidEmojiData(emoji)

        expect(result).not.toBeNull()
        expect(result?.native).toBe(emoji)
        expect(result?.id).toBe('+1')
        expect(result?.skin).toBe(4) 
    })
})
