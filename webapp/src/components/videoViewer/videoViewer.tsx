// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {useHotkeys} from 'react-hotkeys-hook'
import {useIntl} from 'react-intl'

import IconButton from '../../widgets/buttons/iconButton'
import CloseIcon from '../../widgets/icons/close'

import './videoViewer.scss'

type VideoSourceType = 'file' | 'youtube' | 'gdrive'

type Props = {
    sourceType: VideoSourceType
    videoUrl?: string
    videoId?: string
    onClose: () => void
}

const VideoViewer = (props: Props): JSX.Element => {
    const {sourceType, videoUrl, videoId, onClose} = props
    const intl = useIntl()

    useHotkeys('esc', (e) => {
        e.preventDefault()
        e.stopPropagation()
        onClose()
    }, [onClose])

    const handleBackdropClick = useCallback((e: React.MouseEvent) => {
        const target = e.target
        if (target instanceof HTMLElement &&
            (target.classList.contains('VideoViewer') || target.classList.contains('VideoViewer__backdrop'))) {
            onClose()
        }
    }, [onClose])

    const closeText = intl.formatMessage({
        id: 'VideoViewer.close',
        defaultMessage: 'Close video',
    })

    const renderVideo = () => {
        if (sourceType === 'youtube' && videoId) {
            return (
                <iframe
                    className='VideoViewer__iframe'
                    src={`https://www.youtube.com/embed/${videoId}?autoplay=1`}
                    title='YouTube video player'
                    frameBorder='0'
                    allow='accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture'
                    allowFullScreen={true}
                />
            )
        }

        if (sourceType === 'gdrive' && videoId) {
            return (
                <iframe
                    className='VideoViewer__iframe'
                    src={`https://drive.google.com/file/d/${videoId}/preview`}
                    title='Google Drive video player'
                    frameBorder='0'
                    allow='autoplay'
                    allowFullScreen={true}
                />
            )
        }

        if (sourceType === 'file' && videoUrl) {
            return (
                <video
                    className='VideoViewer__video'
                    controls={true}
                    autoPlay={true}
                    onClick={(e) => e.stopPropagation()}
                >
                    <source src={videoUrl}/>
                </video>
            )
        }

        return null
    }

    return (
        <div
            className='VideoViewer'
            onClick={handleBackdropClick}
        >
            <div className='VideoViewer__backdrop'/>
            <div className='VideoViewer__controls VideoViewer__controls--top'>
                <IconButton
                    onClick={onClose}
                    icon={<CloseIcon/>}
                    title={closeText}
                    size='medium'
                />
            </div>
            <div className='VideoViewer__content'>
                {renderVideo()}
            </div>
        </div>
    )
}

export default React.memo(VideoViewer)

