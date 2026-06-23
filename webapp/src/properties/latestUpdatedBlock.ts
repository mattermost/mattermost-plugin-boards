// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Block} from '../blocks/block'

export function definedBlocksForLatestUpdate(card: Block, lastContent?: Block, lastComment?: Block): Block[] {
    return [card, lastContent, lastComment].filter((block): block is Block => Boolean(block))
}

export function getLatestUpdatedBlock(card: Block, lastContent?: Block, lastComment?: Block): Block {
    const allBlocks = definedBlocksForLatestUpdate(card, lastContent, lastComment)
    return allBlocks.sort((a, b) => b.updateAt - a.updateAt)[0] ?? card
}
