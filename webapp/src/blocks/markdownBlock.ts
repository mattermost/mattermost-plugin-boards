// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ContentBlock} from './contentBlock'
import {Block, createBlock} from './block'

type MarkdownBlock = ContentBlock & {
    type: 'markdown'
}

function createMarkdownBlock(block?: Block): MarkdownBlock {
    return {
        ...createBlock(block),
        type: 'markdown',
    }
}

export {MarkdownBlock, createMarkdownBlock}

