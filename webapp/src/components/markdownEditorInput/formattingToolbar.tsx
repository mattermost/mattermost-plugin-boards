// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {useIntl} from 'react-intl'

import IconButton from '../../widgets/buttons/iconButton'
import CompassIcon from '../../widgets/icons/compassIcon'

import './formattingToolbar.scss'

type Props = {
    onFormat: (format: string) => void
}

const FormattingToolbar = (props: Props): JSX.Element => {
    const {onFormat} = props
    const intl = useIntl()

    const boldText = intl.formatMessage({id: 'FormattingToolbar.bold', defaultMessage: 'Bold'})
    const italicText = intl.formatMessage({id: 'FormattingToolbar.italic', defaultMessage: 'Italic'})
    const strikethroughText = intl.formatMessage({id: 'FormattingToolbar.strikethrough', defaultMessage: 'Strikethrough'})
    const codeText = intl.formatMessage({id: 'FormattingToolbar.code', defaultMessage: 'Code'})
    const linkText = intl.formatMessage({id: 'FormattingToolbar.link', defaultMessage: 'Link'})
    const bulletListText = intl.formatMessage({id: 'FormattingToolbar.bulletList', defaultMessage: 'Bullet list'})
    const numberListText = intl.formatMessage({id: 'FormattingToolbar.numberList', defaultMessage: 'Numbered list'})
    const quoteText = intl.formatMessage({id: 'FormattingToolbar.quote', defaultMessage: 'Quote'})

    return (
        <div className='FormattingToolbar'>
            <IconButton
                onClick={() => onFormat('bold')}
                icon={<CompassIcon icon='format-bold'/>}
                title={boldText}
                size='small'
            />
            <IconButton
                onClick={() => onFormat('italic')}
                icon={<CompassIcon icon='format-italic'/>}
                title={italicText}
                size='small'
            />
            <IconButton
                onClick={() => onFormat('strikethrough')}
                icon={<CompassIcon icon='format-strikethrough'/>}
                title={strikethroughText}
                size='small'
            />
            <IconButton
                onClick={() => onFormat('code')}
                icon={<CompassIcon icon='code-tags'/>}
                title={codeText}
                size='small'
            />
            <div className='FormattingToolbar__separator'/>
            <IconButton
                onClick={() => onFormat('link')}
                icon={<CompassIcon icon='link-variant'/>}
                title={linkText}
                size='small'
            />
            <div className='FormattingToolbar__separator'/>
            <IconButton
                onClick={() => onFormat('bulletList')}
                icon={<CompassIcon icon='format-list-bulleted'/>}
                title={bulletListText}
                size='small'
            />
            <IconButton
                onClick={() => onFormat('numberList')}
                icon={<CompassIcon icon='format-list-numbered'/>}
                title={numberListText}
                size='small'
            />
            <IconButton
                onClick={() => onFormat('quote')}
                icon={<CompassIcon icon='format-quote-close'/>}
                title={quoteText}
                size='small'
            />
        </div>
    )
}

export default FormattingToolbar

