// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Block, createBlock} from './block'
import {ContentBlock} from './contentBlock'

type VideoSourceType = 'file' | 'youtube' | 'gdrive'

type VideoBlockFields = {
    sourceType: VideoSourceType
    fileId?: string
    videoUrl?: string
    videoId?: string
    thumbnailUrl?: string
}

type VideoBlock = ContentBlock & {
    type: 'video'
    fields: VideoBlockFields
}

function createVideoBlock(block?: Block): VideoBlock {
    return {
        ...createBlock(block),
        type: 'video',
        fields: {
            sourceType: block?.fields.sourceType || 'file',
            fileId: block?.fields.fileId || '',
            videoUrl: block?.fields.videoUrl || '',
            videoId: block?.fields.videoId || '',
            thumbnailUrl: block?.fields.thumbnailUrl || '',
        },
    }
}

export {VideoBlock, createVideoBlock}
