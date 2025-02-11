// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'

import {FileInfo} from '../../../blocks/block'
import BrokenFile from '../../../widgets/icons/brokenFile'
import {Utils} from '../../../utils'

import './archivedFile.scss'

type Props = {
    fileInfo: FileInfo
}

const ArchivedFile = (props: Props): JSX.Element => {
    const fileName = useCallback(() => props.fileInfo.name || 'untitled file', [props.fileInfo.name])

    const fileExtension = useCallback(() => {
        let extension = props.fileInfo.extension
        extension = extension?.startsWith('.') ? extension?.substring(1) : extension
        return extension?.toUpperCase()
    }, [props.fileInfo.extension])

    const fileSize = useCallback(() => Utils.humanFileSize(props.fileInfo.size || 0), [props.fileInfo.size])

    return (
        <div className='ArchivedFile'>
            <BrokenFile/>
            <div className='fileMetadata'>
                <p className='filename'>{fileName()}</p>
                <p>{fileExtension()} {fileSize()}</p>
            </div>
        </div>
    )
}

export default ArchivedFile
