// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {KeyboardEvent, useState} from 'react'
import {useDispatch} from 'react-redux'

import Dialog from '../components/dialog'
import Button from '../widgets/buttons/button'

import {createPageCategory, renamePageCategory} from '../store/pageCategories'
import type {AppDispatch} from '../store'

// Pages — modal for creating or renaming a page category. Mirrors
// components/createCategory/createCategory.tsx so the look matches Boards.
//
// Using an in-tree modal (rather than window.prompt) avoids the focus race
// where prompt was silently dismissed when the dropdown menu closed.

type Props = {
    teamId: string
    initialValue?: string
    categoryId?: string  // present → rename mode
    onClose: () => void
}

export default function CreatePageCategoryModal({teamId, initialValue, categoryId, onClose}: Props): JSX.Element {
    const dispatch = useDispatch<AppDispatch>()
    const [name, setName] = useState<string>(initialValue || '')
    const isRename = Boolean(categoryId)

    const onSubmit = async () => {
        const trimmed = name.trim()
        if (!trimmed) {
            return
        }
        if (isRename && categoryId) {
            await dispatch(renamePageCategory({teamId, categoryId, name: trimmed}))
        } else {
            await dispatch(createPageCategory({teamId, name: trimmed}))
        }
        onClose()
    }

    const onKeyUp = (e: KeyboardEvent) => {
        if (e.key === 'Enter') {
            void onSubmit()
        }
    }

    return (
        <Dialog
            size='small'
            className='CreateCategoryModal'
            onClose={onClose}
        >
            <div className='box-area'>
                <h3 className='text-heading5'>{isRename ? '카테고리 이름 변경' : '새 카테고리 만들기'}</h3>
                <div style={{margin: '12px 0'}}>
                    <input
                        type='text'
                        autoFocus={true}
                        maxLength={100}
                        placeholder='카테고리 이름'
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        onKeyUp={onKeyUp}
                        style={{
                            width: '100%',
                            padding: '8px 10px',
                            border: '1px solid rgba(var(--center-channel-color-rgb), 0.16)',
                            borderRadius: 4,
                            background: 'rgba(var(--center-channel-bg-rgb), 1)',
                            color: 'inherit',
                            fontSize: 14,
                        }}
                    />
                </div>
                <div className='action-buttons' style={{display: 'flex', gap: 8, justifyContent: 'flex-end'}}>
                    <Button onClick={onClose} emphasis='secondary' size='medium'>
                        {'취소'}
                    </Button>
                    <Button
                        onClick={onSubmit}
                        emphasis='primary'
                        size='medium'
                        filled={Boolean(name.trim())}
                    >
                        {isRename ? '변경' : '만들기'}
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}
