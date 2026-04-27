// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {useSelector} from 'react-redux'

import {getAncestorTrail} from '../store/pages'

// Pages feature — parent trail.

type Props = {
    pageId: string
}

export default function PageBreadcrumb({pageId}: Props): JSX.Element {
    const trail = useSelector(getAncestorTrail(pageId))

    if (trail.length <= 1) {
        return <></>
    }

    return (
        <nav
            className='PageBreadcrumb'
            aria-label='breadcrumb'
            style={{fontSize: 13, color: '#6b7280', marginBottom: 8}}
        >
            {trail.slice(0, -1).map((p, i) => (
                <span key={p.id}>
                    {i > 0 && ' / '}
                    {p.title || '(Untitled)'}
                </span>
            ))}
            <span style={{margin: '0 4px'}}>{' / '}</span>
        </nav>
    )
}
