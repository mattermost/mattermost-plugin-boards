// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getEmojiDataFromNative, BaseEmoji} from 'emoji-mart'
import data from 'emoji-mart/data/all.json'

const EMOJI_SET = 'apple' as const // Single source of truth

export function getValidEmojiData(native: string): BaseEmoji | null {
    try {
        return getEmojiDataFromNative(native, EMOJI_SET, data)
    } catch (err) {
        return null
    }
}
