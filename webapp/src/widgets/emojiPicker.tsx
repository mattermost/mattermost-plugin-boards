// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {FC} from 'react'

import 'emoji-mart/css/emoji-mart.css'
import {Picker, BaseEmoji} from 'emoji-mart'

import {Utils} from '../utils'
import './emojiPicker.scss'

import emojiSpirit from '../../static/emoji_spirit.png'

type Props = {
    onSelect: (emoji: string) => void
}

const EmojiPicker: FC<Props> = (props: Props): JSX.Element => (
    <div
        className='EmojiPicker'
        onClick={(e) => e.stopPropagation()}
    >
        <Picker
            onSelect={(emoji: BaseEmoji) => props.onSelect(emoji.native)}
            backgroundImageFn={() => Utils.buildURL(emojiSpirit, true)}
        />
    </div>
)

export default EmojiPicker
