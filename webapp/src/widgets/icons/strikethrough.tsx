// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import './strikethrough.scss'

export default function StrikethroughIcon(): JSX.Element {
    return (
        <svg
            xmlns='http://www.w3.org/2000/svg'
            className='StrikethroughIcon Icon'
            viewBox='0 0 24 24'
            fill='currentColor'
        >
            <path d='M3 12h18v2H3v-2zm6.5-6c-1.1 0-2 .9-2 2h-2c0-2.2 1.8-4 4-4h5c2.2 0 4 1.8 4 4 0 1.5-.8 2.8-2 3.4v-2.2c.6-.3 1-.9 1-1.7 0-1.1-.9-2-2-2h-5c-.6 0-1 .4-1 1s.4 1 1 1h3v2h-4zm5 8c.6 0 1-.4 1-1h2c0 2.2-1.8 4-4 4h-5c-2.2 0-4-1.8-4-4 0-1.5.8-2.8 2-3.4v2.2c-.6.3-1 .9-1 1.7 0 1.1.9 2 2 2h5c.6 0 1-.4 1-1s-.4-1-1-1h-3v-2h4z'/>
        </svg>
    )
}

