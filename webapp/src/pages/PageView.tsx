// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect} from 'react'
import {useDispatch, useSelector} from 'react-redux'

import {fetchPage, getPage, setCurrentPage} from '../store/pages'
import type {AppDispatch} from '../store'

import PageBreadcrumb from './PageBreadcrumb'
import PageEditor from './PageEditor'
import PageChannelsBar from './PageChannelsBar'

// Pages feature — page detail view (Phase 1).
//
// Composition:
//   <PageBreadcrumb/>     parent trail
//   <PageHeader>          title + icon
//   <PageEditor/>         Tiptap instance + debounced save
//
// Title editing in Phase 1 is intentionally simple (contentEditable on the h1).
// Phase 2 adds: icon picker, share button, more menu, Yjs awareness display.

type Props = {
    pageId: string
}

export default function PageView({pageId}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const page = useSelector(getPage(pageId))

    useEffect(() => {
        dispatch(setCurrentPage(pageId))
        if (!page) {
            dispatch(fetchPage(pageId))
        }
    }, [pageId, dispatch, page])

    if (!page) {
        return <div className='PageView PageView--loading'>{'Loading page…'}</div>
    }

    return (
        <div className='PageView'>
            <PageBreadcrumb pageId={pageId}/>
            <header className='PageView__header' style={{marginBottom: 8}}>
                {page.icon && <span style={{fontSize: 32, marginRight: 8}}>{page.icon}</span>}
                <h1 style={{fontSize: 28, margin: 0}}>{page.title || '(Untitled)'}</h1>
            </header>
            <PageChannelsBar pageId={pageId} teamId={page.teamId}/>
            <PageEditor pageId={pageId}/>
        </div>
    )
}
