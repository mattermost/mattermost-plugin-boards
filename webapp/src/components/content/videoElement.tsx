// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState, useCallback} from 'react'
import {IntlShape} from 'react-intl'

import {ContentBlock} from '../../blocks/contentBlock'
import {VideoBlock, createVideoBlock} from '../../blocks/videoBlock'
import octoClient from '../../octoClient'
import VideoIcon from '../../widgets/icons/video'
import {sendFlashMessage} from '../../components/flashMessages'
import CompassIcon from '../../widgets/icons/compassIcon'

import {FileInfo} from '../../blocks/block'
import VideoViewer from '../videoViewer/videoViewer'
import RootPortal from '../rootPortal'

import {contentRegistry} from './contentRegistry'
import './videoElement.scss'

type Props = {
    block: ContentBlock
}

type VideoSourceType = 'file' | 'youtube' | 'gdrive'

// URL detection patterns
const YOUTUBE_PATTERNS = [
    /(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})/,
    /youtube\.com\/embed\/([a-zA-Z0-9_-]{11})/,
]

const GDRIVE_PATTERN = /drive\.google\.com\/file\/d\/([a-zA-Z0-9_-]+)/

const detectVideoSource = (url: string): {sourceType: VideoSourceType; videoId: string} | null => {
    // Check YouTube patterns
    for (const pattern of YOUTUBE_PATTERNS) {
        const match = url.match(pattern)
        if (match) {
            return {sourceType: 'youtube', videoId: match[1]}
        }
    }

    // Check Google Drive pattern
    const gdriveMatch = url.match(GDRIVE_PATTERN)
    if (gdriveMatch) {
        return {sourceType: 'gdrive', videoId: gdriveMatch[1]}
    }

    return null
}

const VideoElement = (props: Props): JSX.Element|null => {
    const [videoDataUrl, setVideoDataUrl] = useState<string|null>(null)
    const [showViewer, setShowViewer] = useState(false)

    const {block} = props
    const sourceType: VideoSourceType = block.fields.sourceType || 'file'
    const videoId = block.fields.videoId || ''
    const videoUrl = block.fields.videoUrl || ''

    useEffect(() => {
        if (sourceType === 'file' && !videoDataUrl && block.fields.fileId) {
            const loadVideo = async () => {
                const fileURL = await octoClient.getFileAsDataUrl(block.boardId, block.fields.fileId)
                setVideoDataUrl(fileURL.url || '')
            }
            loadVideo()
        }
    }, [block, sourceType, videoDataUrl])

    const handleVideoClick = useCallback((e: React.MouseEvent) => {
        e.stopPropagation()
        setShowViewer(true)
    }, [])

    const handleVideoKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            e.stopPropagation()
            setShowViewer(true)
        }
    }, [])

    const handleCloseViewer = useCallback(() => {
        setShowViewer(false)
    }, [])

    // YouTube embed
    if (sourceType === 'youtube' && videoId) {
        const thumbnailUrl = `https://img.youtube.com/vi/${videoId}/hqdefault.jpg`
        return (
            <>
                <div className='VideoElement__container'>
                    <div className='VideoElement__wrapper'>
                        <img
                            className='VideoElement__thumbnail'
                            src={thumbnailUrl}
                            alt='Video thumbnail'
                        />
                        <div
                            className='VideoElement__overlay'
                            onClick={handleVideoClick}
                            onKeyDown={handleVideoKeyDown}
                            tabIndex={0}
                            role='button'
                            aria-label='Play video in full screen'
                        >
                            <div className='VideoElement__play-icon'>
                                <CompassIcon
                                    icon='play'
                                    className='PlayIcon'
                                />
                            </div>
                        </div>
                    </div>
                    <div className='VideoElement__metadata'>
                        <span className='VideoElement__source'>YouTube</span>
                    </div>
                </div>
                {showViewer && (
                    <RootPortal>
                        <VideoViewer
                            sourceType='youtube'
                            videoId={videoId}
                            onClose={handleCloseViewer}
                        />
                    </RootPortal>
                )}
            </>
        )
    }

    // Google Drive embed
    if (sourceType === 'gdrive' && videoId) {
        return (
            <>
                <div className='VideoElement__container'>
                    <div className='VideoElement__wrapper'>
                        <div className='VideoElement__gdrive-placeholder'>
                            <CompassIcon
                                icon='file-video-outline'
                                className='GDriveIcon'
                            />
                        </div>
                        <div
                            className='VideoElement__overlay'
                            onClick={handleVideoClick}
                            onKeyDown={handleVideoKeyDown}
                            tabIndex={0}
                            role='button'
                            aria-label='Play video in full screen'
                        >
                            <div className='VideoElement__play-icon'>
                                <CompassIcon
                                    icon='play'
                                    className='PlayIcon'
                                />
                            </div>
                        </div>
                    </div>
                    <div className='VideoElement__metadata'>
                        <span className='VideoElement__source'>Google Drive</span>
                    </div>
                </div>
                {showViewer && (
                    <RootPortal>
                        <VideoViewer
                            sourceType='gdrive'
                            videoId={videoId}
                            onClose={handleCloseViewer}
                        />
                    </RootPortal>
                )}
            </>
        )
    }

    // File upload
    if (sourceType === 'file' && videoDataUrl) {
        return (
            <>
                <div className='VideoElement__container'>
                    <div className='VideoElement__wrapper'>
                        <video
                            className='VideoElement__preview'
                        >
                            <source src={videoDataUrl}/>
                        </video>
                        <div
                            className='VideoElement__overlay'
                            onClick={handleVideoClick}
                            onKeyDown={handleVideoKeyDown}
                            tabIndex={0}
                            role='button'
                            aria-label='Play video in full screen'
                        >
                            <div className='VideoElement__play-icon'>
                                <CompassIcon
                                    icon='play'
                                    className='PlayIcon'
                                />
                            </div>
                        </div>
                    </div>
                    <div className='VideoElement__metadata'>
                        <span className='VideoElement__source'>Video</span>
                    </div>
                </div>
                {showViewer && (
                    <RootPortal>
                        <VideoViewer
                            sourceType='file'
                            videoUrl={videoDataUrl}
                            onClose={handleCloseViewer}
                        />
                    </RootPortal>
                )}
            </>
        )
    }

    return null
}

contentRegistry.registerContentType({
    type: 'video',
    getDisplayText: (intl: IntlShape) => intl.formatMessage({id: 'ContentBlock.video', defaultMessage: 'video'}),
    getIcon: () => <VideoIcon/>,
    createBlock: async (boardId: string, intl: IntlShape) => {
        return new Promise<VideoBlock>((resolve) => {
            // Prompt for URL or file
            const url = prompt('Enter YouTube or Google Drive URL (or leave empty to upload a file):')
            if (url && url.trim()) {
                const detected = detectVideoSource(url.trim())
                if (detected) {
                    const block = createVideoBlock()
                    block.fields.sourceType = detected.sourceType
                    block.fields.videoId = detected.videoId
                    block.fields.videoUrl = url.trim()
                    resolve(block)
                } else {
                    sendFlashMessage({
                        content: intl.formatMessage({
                            id: 'createVideoBlock.invalidUrl',
                            defaultMessage: 'Unable to detect video source. Please enter a valid YouTube or Google Drive URL.'
                        }),
                        severity: 'normal'
                    })
                    resolve({} as VideoBlock)
                }
            } else {
                // File upload
                const input = document.createElement('input')
                input.type = 'file'
                input.accept = 'video/*'
                input.onchange = async (e) => {
                    const file = (e.target as HTMLInputElement).files?.[0]
                    if (file) {
                        const fileId = await octoClient.uploadFile(boardId, file)
                        if (fileId) {
                            const block = createVideoBlock()
                            block.fields.sourceType = 'file'
                            block.fields.fileId = fileId
                            resolve(block)
                        } else {
                            sendFlashMessage({
                                content: intl.formatMessage({
                                    id: 'createVideoBlock.failed',
                                    defaultMessage: 'Unable to upload the file. File size limit reached.'
                                }),
                                severity: 'normal'
                            })
                            resolve({} as VideoBlock)
                        }
                    } else {
                        resolve({} as VideoBlock)
                    }
                }
                input.click()
            }
        })
    },
    createComponent: (block) => <VideoElement block={block}/>,
})

export default React.memo(VideoElement)
