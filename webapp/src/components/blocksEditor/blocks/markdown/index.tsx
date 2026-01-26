// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {MarkdownEditor} from '../../../markdownEditor'
import {Utils} from '../../../../utils'

import {BlockInputProps, ContentType} from '../types'

import './markdown.scss'

const MarkdownContent: ContentType = {
    name: 'markdown',
    displayName: 'Markdown',
    slashCommand: '/markdown',
    prefix: '',
    runSlashCommand: (): void => {},
    editable: true,
    Display: (props: BlockInputProps) => {
        const html: string = Utils.htmlFromMarkdown(props.value || '')
        return (
            <div
                dangerouslySetInnerHTML={{__html: html}}
                className={props.value ? 'octo-editor-preview' : 'octo-editor-preview octo-placeholder'}
            />
        )
    },
    Input: (props: BlockInputProps) => {
        return (
            <div
                className='MarkdownContent'
                data-testid='markdown'
            >
                <MarkdownEditor
                    autofocus={true}
                    onBlur={(val: string) => {
                        props.onSave(val)
                    }}
                    text={props.value}
                    saveOnEnter={true}
                    onEditorCancel={() => {
                        props.onCancel()
                    }}
                    showToolbar={true}
                    keepEditing={true}
                />
            </div>
        )
    },
}

MarkdownContent.runSlashCommand = (changeType: (contentType: ContentType) => void, changeValue: (value: string) => void, ...args: string[]): void => {
    changeType(MarkdownContent)
    changeValue(args.join(' '))
}

export default MarkdownContent

