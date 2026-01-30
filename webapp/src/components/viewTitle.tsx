// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {BlockIcons} from '../blockIcons'
import {Board} from '../blocks/board'
import mutator from '../mutator'
import Button from '../widgets/buttons/button'
import Editable from '../widgets/editable'
import CompassIcon from '../widgets/icons/compassIcon'
import {Permission} from '../constants'
import {useHasCurrentBoardPermissions} from '../hooks/permissions'

import BoardIconSelector from './boardIconSelector'
import {MarkdownEditor} from './markdownEditor'
import './viewTitle.scss'

type Props = {
    board: Board
    readonly: boolean
}

const ViewTitle = (props: Props) => {
    const {board} = props

    const [title, setTitle] = useState(board.title)
    const onEditTitleSave = useCallback(() => mutator.changeBoardTitle(board.id, board.title, title), [board.id, board.title, title])
    const onEditTitleCancel = useCallback(() => setTitle(board.title), [board.title])
    const onDescriptionBlur = useCallback((text) => mutator.changeBoardDescription(board.id, board.id, board.description, text), [board.id, board.description])
    const onAddRandomIcon = useCallback(() => {
        const newIcon = BlockIcons.shared.randomIcon()
        mutator.changeBoardIcon(board.id, board.icon, newIcon)
    }, [board.id, board.icon])
    const onShowDescription = useCallback(() => mutator.showBoardDescription(board.id, Boolean(board.showDescription), true), [board.id, board.showDescription])
    const onHideDescription = useCallback(() => mutator.showBoardDescription(board.id, Boolean(board.showDescription), false), [board.id, board.showDescription])
    const canEditBoardProperties = useHasCurrentBoardPermissions([Permission.ManageBoardProperties])

    // Issue 2: Force readonly to true - editing is now done in Settings only
    const readonly = true

    const intl = useIntl()

    // Issue 2: Hide description if empty
    const hasDescription = board.description && board.description.trim().length > 0

    return (
        <div className='ViewTitle'>
            {/* Issue 2: Remove add-buttons section - editing is now done in Settings only */}

            <div className='title'>
                {/* Issue 2: Icon is hidden if empty (BoardIconSelector returns null) */}
                <BoardIconSelector
                    board={board}
                    readonly={readonly}
                />
                <Editable
                    className='title'
                    value={title}
                    placeholderText={intl.formatMessage({id: 'ViewTitle.untitled-board', defaultMessage: 'Untitled board'})}
                    onChange={(newTitle) => setTitle(newTitle)}
                    saveOnEsc={true}
                    onSave={onEditTitleSave}
                    onCancel={onEditTitleCancel}
                    readonly={readonly}
                    spellCheck={true}
                />
            </div>

            {/* Issue 2: Only show description if it has content */}
            {hasDescription &&
                <div className='description'>
                    <MarkdownEditor
                        text={board.description}
                        placeholderText='Add a description...'
                        onBlur={onDescriptionBlur}
                        readonly={readonly}
                    />
                </div>
            }
        </div>
    )
}

export default React.memo(ViewTitle)
