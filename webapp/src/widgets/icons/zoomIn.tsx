// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import './zoomIn.scss'

export default function ZoomInIcon(): JSX.Element {
    return (
        <svg
            xmlns='http://www.w3.org/2000/svg'
            className='ZoomInIcon Icon'
            viewBox='0 0 24 24'
            fill='currentColor'
        >
            <path d='M12 5v14M5 12h14' stroke='currentColor' strokeWidth='2' strokeLinecap='round'/>
        </svg>
    )
}

