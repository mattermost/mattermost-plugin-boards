// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {useHotkeys} from 'react-hotkeys-hook'
import {useIntl} from 'react-intl'

import IconButton from '../../widgets/buttons/iconButton'
import CloseIcon from '../../widgets/icons/close'
import ZoomInIcon from '../../widgets/icons/zoomIn'
import ZoomOutIcon from '../../widgets/icons/zoomOut'

import './imageViewer.scss'

type Props = {
    imageUrl: string
    onClose: () => void
}

const MIN_ZOOM = 0.5
const MAX_ZOOM = 5
const ZOOM_STEP = 0.5

const ImageViewer = (props: Props): JSX.Element => {
    const {imageUrl, onClose} = props
    const intl = useIntl()
    const [zoom, setZoom] = useState(1)
    const [isMobile, setIsMobile] = useState(false)

    useEffect(() => {
        const checkMobile = () => {
            setIsMobile(window.innerWidth <= 768)
        }
        checkMobile()
        window.addEventListener('resize', checkMobile)
        return () => window.removeEventListener('resize', checkMobile)
    }, [])

    useHotkeys('esc', (e) => {
        e.preventDefault()
        e.stopPropagation()
        onClose()
    })

    const handleZoomIn = useCallback(() => {
        setZoom((prevZoom) => Math.min(prevZoom + ZOOM_STEP, MAX_ZOOM))
    }, [])

    const handleZoomOut = useCallback(() => {
        setZoom((prevZoom) => Math.max(prevZoom - ZOOM_STEP, MIN_ZOOM))
    }, [])

    const handleBackdropClick = useCallback((e: React.MouseEvent) => {
        const target = e.target as HTMLElement
        if (target.classList.contains('ImageViewer') || target.classList.contains('ImageViewer__backdrop')) {
            onClose()
        }
    }, [onClose])

    const handleOpenInNewTab = useCallback(() => {
        window.open(imageUrl, '_blank')
    }, [imageUrl])

    const closeText = intl.formatMessage({
        id: 'ImageViewer.close',
        defaultMessage: 'Close image',
    })

    const zoomInText = intl.formatMessage({
        id: 'ImageViewer.zoomIn',
        defaultMessage: 'Zoom in',
    })

    const zoomOutText = intl.formatMessage({
        id: 'ImageViewer.zoomOut',
        defaultMessage: 'Zoom out',
    })

    const openInNewTabText = intl.formatMessage({
        id: 'ImageViewer.openInNewTab',
        defaultMessage: 'Open in new tab',
    })

    return (
        <div
            className='ImageViewer'
            onClick={handleBackdropClick}
        >
            <div className='ImageViewer__backdrop'/>
            <div className='ImageViewer__controls ImageViewer__controls--top'>
                <IconButton
                    onClick={onClose}
                    icon={<CloseIcon/>}
                    title={closeText}
                    size='medium'
                />
            </div>
            <div className='ImageViewer__content'>
                <img
                    className={`ImageViewer__image ${zoom > 1 ? 'ImageViewer__image--zoomed' : ''}`}
                    src={imageUrl}
                    alt=''
                    data-zoom={zoom}
                    onClick={(e) => e.stopPropagation()}
                />
            </div>
            {!isMobile && (
                <div className='ImageViewer__controls ImageViewer__controls--bottom'>
                    <IconButton
                        onClick={handleZoomOut}
                        icon={<ZoomOutIcon/>}
                        title={zoomOutText}
                        size='medium'
                    />
                    <span className='ImageViewer__zoom-level'>{Math.round(zoom * 100)}%</span>
                    <IconButton
                        onClick={handleZoomIn}
                        icon={<ZoomInIcon/>}
                        title={zoomInText}
                        size='medium'
                    />
                </div>
            )}
            {isMobile && (
                <div className='ImageViewer__controls ImageViewer__controls--bottom'>
                    <button
                        type='button'
                        className='ImageViewer__mobile-button'
                        onClick={handleOpenInNewTab}
                    >
                        {openInNewTabText}
                    </button>
                </div>
            )}
        </div>
    )
}

export default React.memo(ImageViewer)

