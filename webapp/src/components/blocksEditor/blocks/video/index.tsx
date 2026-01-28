// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useRef, useEffect, useState, useCallback} from 'react'

import {BlockInputProps, ContentType} from '../types'
import octoClient from '../../../../octoClient'
import CompassIcon from '../../../../widgets/icons/compassIcon'
import VideoViewer from '../../../videoViewer/videoViewer'
import RootPortal from '../../../rootPortal'

import './video.scss'

type VideoSourceType = 'file' | 'youtube' | 'gdrive'

type FileInfo = {
    file?: string|File
    filename?: string
    width?: number
    align?: 'left'|'center'|'right'
    sourceType?: VideoSourceType
    videoUrl?: string
    videoId?: string
}

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

const Video: ContentType<FileInfo> = {
    name: 'video',
    displayName: 'Video',
    slashCommand: '/video',
    prefix: '',
    runSlashCommand: (): void => {},
    editable: false,
    Display: (props: BlockInputProps<FileInfo>) => {
        const [videoDataUrl, setVideoDataUrl] = useState<string|null>(null)
        const [showViewer, setShowViewer] = useState(false)
        const sourceType = props.value?.sourceType || 'file'
        const videoId = props.value?.videoId || ''

        useEffect(() => {
            if (sourceType === 'file' && !videoDataUrl) {
                const loadVideo = async () => {
                    if (props.value && props.value.file && typeof props.value.file === 'string') {
                        const fileURL = await octoClient.getFileAsDataUrl(props.currentBoardId || '', props.value.file)
                        setVideoDataUrl(fileURL.url || '')
                    }
                }
                loadVideo()
            }
        }, [props.value, props.value?.file, props.currentBoardId, sourceType])

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
                                data-testid='video-thumbnail'
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
        if (videoDataUrl) {
            return (
                <>
                    <div className='VideoElement__container'>
                        <div className='VideoElement__wrapper'>
                            <video
                                className='VideoElement__preview'
                                data-testid='video'
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
                            <span className='VideoElement__source'>{props.value?.filename || 'Video'}</span>
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
    },
    Input: (props: BlockInputProps<FileInfo>) => {
        const ref = useRef<HTMLInputElement|null>(null)
        const [urlInput, setUrlInput] = useState('')
        const [showFileInput, setShowFileInput] = useState(false)

        useEffect(() => {
            // Auto-detect if value contains a URL
            if (props.value && typeof props.value === 'string') {
                const detected = detectVideoSource(props.value)
                if (detected) {
                    props.onSave({
                        sourceType: detected.sourceType,
                        videoId: detected.videoId,
                        videoUrl: props.value,
                    })
                    return
                }
            }
        }, [])

        const handleUrlSubmit = useCallback(() => {
            if (!urlInput.trim()) {
                props.onCancel()
                return
            }

            const detected = detectVideoSource(urlInput)
            if (detected) {
                props.onSave({
                    sourceType: detected.sourceType,
                    videoId: detected.videoId,
                    videoUrl: urlInput,
                })
            } else {
                // Invalid URL, show error or cancel
                props.onCancel()
            }
        }, [urlInput, props])

        const handleFileSelect = useCallback(() => {
            setShowFileInput(true)
            setTimeout(() => ref.current?.click(), 0)
        }, [])

        return (
            <div className='VideoInput'>
                {!showFileInput ? (
                    <div className='VideoInput__url'>
                        <input
                            type='text'
                            placeholder='Paste YouTube or Google Drive URL...'
                            value={urlInput}
                            onChange={(e) => setUrlInput(e.target.value)}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter') {
                                    handleUrlSubmit()
                                } else if (e.key === 'Escape') {
                                    props.onCancel()
                                }
                            }}
                            autoFocus={true}
                        />
                        <button onClick={handleUrlSubmit}>{'Add'}</button>
                        <button onClick={handleFileSelect}>{'Upload File'}</button>
                    </div>
                ) : (
                    <>
                        <input
                            ref={ref}
                            className='Video'
                            data-testid='video-input'
                            type='file'
                            accept='video/*'
                            onChange={(e) => {
                                const file = (e.currentTarget?.files || [])[0]
                                if (file) {
                                    props.onSave({file, filename: file.name, sourceType: 'file'})
                                } else {
                                    props.onCancel()
                                }
                            }}
                        />
                    </>
                )}
            </div>
        )
    },
}

Video.runSlashCommand = (changeType: (contentType: ContentType<FileInfo>) => void, changeValue: (value: FileInfo) => void): void => {
    changeType(Video)
    changeValue({} as any)
}

export default Video
