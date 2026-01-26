// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useRef} from 'react'
import {useIntl} from 'react-intl'

import {MarkdownBlock, createMarkdownBlock} from '../../blocks/markdownBlock'
import mutator from '../../mutator'
import TextIcon from '../../widgets/icons/text'
import {MarkdownEditor} from '../markdownEditor'

import {contentRegistry} from './contentRegistry'

type Props = {
    block: MarkdownBlock
    readonly: boolean
}

const MarkdownElement = (props: Props): JSX.Element => {
    const {block, readonly} = props
    const intl = useIntl()
    const containerRef = useRef<HTMLDivElement>(null)

    return (
        <div ref={containerRef}>
            <MarkdownEditor
                text={block.title}
                placeholderText={intl.formatMessage({id: 'ContentBlock.editMarkdown', defaultMessage: 'Edit markdown...'})}
                onBlur={(text: string) => {
                    if (text !== block.title) {
                        mutator.changeBlockTitle(block.boardId, block.id, block.title, text, intl.formatMessage({id: 'ContentBlock.editCardMarkdown', defaultMessage: 'edit card markdown'}))
                    }
                }}
                readonly={readonly}
            />
        </div>
    )
}

contentRegistry.registerContentType({
    type: 'markdown',
    getDisplayText: (intl) => intl.formatMessage({id: 'ContentBlock.markdown', defaultMessage: 'markdown'}),
    getIcon: () => <TextIcon/>,
    createBlock: async () => {
        return createMarkdownBlock()
    },
    createComponent: (block, readonly) => {
        return (
            <MarkdownElement
                block={block as MarkdownBlock}
                readonly={readonly}
            />
        )
    },
})

export default React.memo(MarkdownElement)

