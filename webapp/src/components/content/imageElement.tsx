// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState, useCallback} from 'react'
import {IntlShape} from 'react-intl'

import {ContentBlock} from '../../blocks/contentBlock'
import {ImageBlock, createImageBlock} from '../../blocks/imageBlock'
import octoClient from '../../octoClient'
import {Utils} from '../../utils'
import ImageIcon from '../../widgets/icons/image'
import {sendFlashMessage} from '../../components/flashMessages'
import CompassIcon from '../../widgets/icons/compassIcon'

import {FileInfo} from '../../blocks/block'
import ImageViewer from '../imageViewer/imageViewer'
import RootPortal from '../rootPortal'

import {contentRegistry} from './contentRegistry'
import ArchivedFile from './archivedFile/archivedFile'

import './imageElement.scss'

type Props = {
    block: ContentBlock
}

type ImageDimensions = {
    width: number
    height: number
}

const ImageElement = (props: Props): JSX.Element|null => {
    const [imageDataUrl, setImageDataUrl] = useState<string|null>(null)
    const [fileInfo, setFileInfo] = useState<FileInfo>({})
    const [imageDimensions, setImageDimensions] = useState<ImageDimensions|null>(null)
    const [showViewer, setShowViewer] = useState(false)

    const {block} = props

    useEffect(() => {
        if (!imageDataUrl) {
            const loadImage = async () => {
                const fileURL = await octoClient.getFileAsDataUrl(block.boardId, props.block.fields.fileId)
                setImageDataUrl(fileURL.url || '')

                const fullFileInfo = await octoClient.getFileInfo(block.boardId, props.block.fields.fileId)
                setFileInfo(fullFileInfo)
            }
            loadImage()
        }
    }, [])

    const handleImageLoad = useCallback((e: React.SyntheticEvent<HTMLImageElement>) => {
        const img = e.currentTarget
        setImageDimensions({
            width: img.naturalWidth,
            height: img.naturalHeight,
        })
    }, [])

    const handleImageClick = useCallback((e: React.MouseEvent) => {
        e.stopPropagation()
        setShowViewer(true)
    }, [])

    const handleImageKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            e.stopPropagation()
            setShowViewer(true)
        }
    }, [])

    const handleCloseViewer = useCallback(() => {
        setShowViewer(false)
    }, [])

    if (fileInfo && fileInfo.archived) {
        return (
            <ArchivedFile fileInfo={fileInfo}/>
        )
    }

    if (!imageDataUrl) {
        return null
    }

    return (
        <>
            <div className='ImageElement__container'>
                <div className='ImageElement__wrapper'>
                    <img
                        className='ImageElement'
                        src={imageDataUrl}
                        alt=''
                        aria-label={block.title || 'View image in full screen'}
                        onLoad={handleImageLoad}
                    />
                    <div
                        className='ImageElement__overlay'
                        onClick={handleImageClick}
                        onKeyDown={handleImageKeyDown}
                        tabIndex={0}
                        role='button'
                        aria-label='View image in full screen'
                    >
                        <div className='ImageElement__magnify-icon'>
                            <CompassIcon
                                icon='magnify'
                                className='MagnifyIcon'
                            />
                        </div>
                    </div>
                </div>
                {(imageDimensions || (fileInfo && fileInfo.size) || imageDataUrl) && (
                    <div className='ImageElement__metadata'>
                        {imageDimensions && (
                            <span className='ImageElement__dimensions'>
                                {imageDimensions.width}×{imageDimensions.height}
                            </span>
                        )}
                        {fileInfo && fileInfo.size && (
                            <span className='ImageElement__size'>
                                {Utils.humanFileSize(fileInfo.size)}
                            </span>
                        )}
                        {imageDataUrl && (
                            <a
                                href={imageDataUrl}
                                target='_blank'
                                rel='noopener noreferrer'
                                className='ImageElement__download'
                                onClick={(e) => e.stopPropagation()}
                            >
                                Download
                            </a>
                        )}
                    </div>
                )}
            </div>
            {showViewer && (
                <RootPortal>
                    <ImageViewer
                        imageUrl={imageDataUrl}
                        onClose={handleCloseViewer}
                    />
                </RootPortal>
            )}
        </>
    )
}

contentRegistry.registerContentType({
    type: 'image',
    getDisplayText: (intl: IntlShape) => intl.formatMessage({id: 'ContentBlock.image', defaultMessage: 'image'}),
    getIcon: () => <ImageIcon/>,
    createBlock: async (boardId: string, intl: IntlShape) => {
        return new Promise<ImageBlock>(
            (resolve) => {
                Utils.selectLocalFile(async (file) => {
                    const fileId = await octoClient.uploadFile(boardId, file)

                    if (fileId) {
                        const block = createImageBlock()
                        block.fields.fileId = fileId || ''
                        resolve(block)
                    } else {
                        sendFlashMessage({content: intl.formatMessage({id: 'createImageBlock.failed', defaultMessage: 'Unable to upload the file. File size limit reached.'}), severity: 'normal'})
                    }
                },
                '.jpg,.jpeg,.png,.gif')
            },
        )

        // return new ImageBlock()
    },
    createComponent: (block) => <ImageElement block={block}/>,
})

export default React.memo(ImageElement)
