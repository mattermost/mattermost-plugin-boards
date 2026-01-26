// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useRef, useEffect, useState, useCallback} from 'react'

import {BlockInputProps, ContentType} from '../types'
import octoClient from '../../../../octoClient'
import ImageViewer from '../../../imageViewer/imageViewer'
import RootPortal from '../../../rootPortal'
import CompassIcon from '../../../../widgets/icons/compassIcon'
import {Utils} from '../../../../utils'

import './image.scss'

type FileInfo = {
    file: string|File
    width?: number
    align?: 'left'|'center'|'right'
}

type FullFileInfo = {
    url?: string
    archived?: boolean
    extension?: string
    name?: string
    size?: number
}

type ImageDimensions = {
    width: number
    height: number
}

const Image: ContentType<FileInfo> = {
    name: 'image',
    displayName: 'Image',
    slashCommand: '/image',
    prefix: '',
    runSlashCommand: (): void => {},
    editable: false,
    Display: (props: BlockInputProps<FileInfo>) => {
        const [imageDataUrl, setImageDataUrl] = useState<string|null>(null)
        const [fullFileInfo, setFullFileInfo] = useState<FullFileInfo>({})
        const [imageDimensions, setImageDimensions] = useState<ImageDimensions|null>(null)
        const [showViewer, setShowViewer] = useState(false)

        useEffect(() => {
            if (!imageDataUrl) {
                const loadImage = async () => {
                    if (props.value && props.value.file && typeof props.value.file === 'string') {
                        const fileURL = await octoClient.getFileAsDataUrl(props.currentBoardId || '', props.value.file)
                        setImageDataUrl(fileURL.url || '')

                        const fileInfo = await octoClient.getFileInfo(props.currentBoardId || '', props.value.file)
                        setFullFileInfo(fileInfo)
                    }
                }
                loadImage()
            }
        }, [props.value, props.value.file, props.currentBoardId])

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

        if (imageDataUrl) {
            return (
                <>
                    <div className='ImageView__container'>
                        <div className='ImageView__wrapper'>
                            <img
                                data-testid='image'
                                className='ImageView'
                                src={imageDataUrl}
                                alt=''
                                aria-label='View image in full screen'
                                onLoad={handleImageLoad}
                            />
                            <div
                                className='ImageView__overlay'
                                onClick={handleImageClick}
                                onKeyDown={handleImageKeyDown}
                                tabIndex={0}
                                role='button'
                                aria-label='View image in full screen'
                            >
                                <div className='ImageView__magnify-icon'>
                                    <CompassIcon
                                        icon='magnify'
                                        className='MagnifyIcon'
                                    />
                                </div>
                            </div>
                        </div>
                        {(imageDimensions || (fullFileInfo && fullFileInfo.size) || imageDataUrl) && (
                            <div className='ImageView__metadata'>
                                {imageDimensions && (
                                    <span className='ImageView__dimensions'>
                                        {imageDimensions.width}×{imageDimensions.height}
                                    </span>
                                )}
                                {fullFileInfo && fullFileInfo.size && (
                                    <span className='ImageView__size'>
                                        {Utils.humanFileSize(fullFileInfo.size)}
                                    </span>
                                )}
                                {imageDataUrl && (
                                    <a
                                        href={imageDataUrl}
                                        target='_blank'
                                        rel='noopener noreferrer'
                                        className='ImageView__download'
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
        return null
    },
    Input: (props: BlockInputProps<FileInfo>) => {
        const ref = useRef<HTMLInputElement|null>(null)
        useEffect(() => {
            ref.current?.click()
        }, [])

        return (
            <div>
                {props.value.file && (typeof props.value.file === 'string') && (
                    <img
                        className='ImageView'
                        src={props.value.file}
                        onClick={() => ref.current?.click()}
                    />
                )}
                <input
                    ref={ref}
                    className='Image'
                    data-testid='image-input'
                    type='file'
                    accept='image/*'
                    onChange={(e) => {
                        const file = (e.currentTarget?.files || [])[0]
                        props.onSave({file})
                    }}
                />
            </div>
        )
    },
}

Image.runSlashCommand = (changeType: (contentType: ContentType<FileInfo>) => void, changeValue: (value: FileInfo) => void): void => {
    changeType(Image)
    changeValue({file: ''})
}

export default Image
