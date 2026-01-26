// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import './markdown.scss'

export default function MarkdownIcon(): JSX.Element {
    return (
        <svg
            xmlns='http://www.w3.org/2000/svg'
            className='MarkdownIcon Icon'
            viewBox='0 0 24 24'
            fill='currentColor'
        >
            <path
                d='M3 3h18v18H3V3zm2 2v14h14V5H5zm2 2h2v2h2V7h2v10h-2v-6H9v6H7V7zm10 0h2l2 3v7h-2v-5l-1 1.5L17 14v5h-2V7z'
            />
        </svg>
    )
}

