// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback, useEffect} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Board} from '../../blocks/board'
import BoardIconSelector from '../../components/boardIconSelector'
import Editable from '../../widgets/editable'
import EditableArea from '../../widgets/editableArea'

import './generalSection.scss'

type Props = {
    board: Board
    onBoardChange: (board: Board) => void
}

const GeneralSection = (props: Props): JSX.Element => {
    const {board} = props
    const intl = useIntl()

    const [title, setTitle] = useState(board.title)
    const [code, setCode] = useState(board.code || '')
    const [description, setDescription] = useState(board.description || '')

    // Update local state when board changes
    useEffect(() => {
        setTitle(board.title)
        setCode(board.code || '')
        setDescription(board.description || '')
    }, [board.id, board.title, board.code, board.description])

    const handleTitleChange = useCallback((newTitle: string) => {
        setTitle(newTitle)
    }, [])

    const handleTitleSave = useCallback(() => {
        if (title !== board.title) {
            props.onBoardChange({...board, title})
        }
    }, [title, board, props])

    const handleCodeChange = useCallback((newCode: string) => {
        setCode(newCode)
    }, [])

    const handleCodeSave = useCallback(() => {
        if (code !== (board.code || '')) {
            props.onBoardChange({...board, code})
        }
    }, [code, board, props])

    const handleDescriptionChange = useCallback((newDescription: string) => {
        setDescription(newDescription)
    }, [])

    const handleDescriptionSave = useCallback(() => {
        if (description !== (board.description || '')) {
            props.onBoardChange({...board, description})
        }
    }, [description, board, props])

    const validateCode = useCallback((value: string): boolean => {
        // Code validation: 1-10 alphanumeric characters, must start with a letter
        if (!value) {
            return true // Empty is valid
        }
        const codeRegex = /^[a-zA-Z][a-zA-Z0-9]{0,9}$/
        return codeRegex.test(value)
    }, [])

    return (
        <div className='GeneralSection'>
            {/* Board Name */}
            <div className='GeneralSection__field'>
                <label className='GeneralSection__label'>
                    <FormattedMessage
                        id='BoardSettings.general.name'
                        defaultMessage='Board Name'
                    />
                </label>
                <div className='GeneralSection__input-wrapper'>
                    <BoardIconSelector
                        board={board}
                        size='m'
                    />
                    <Editable
                        className='GeneralSection__input GeneralSection__input--title'
                        value={title}
                        placeholderText={intl.formatMessage({
                            id: 'BoardSettings.general.name-placeholder',
                            defaultMessage: 'Untitled board',
                        })}
                        onChange={handleTitleChange}
                        onSave={handleTitleSave}
                        saveOnEsc={true}
                        spellCheck={true}
                    />
                </div>
            </div>

            {/* Board Code */}
            <div className='GeneralSection__field'>
                <label className='GeneralSection__label'>
                    <FormattedMessage
                        id='BoardSettings.general.code'
                        defaultMessage='Board Code'
                    />
                </label>
                <Editable
                    className='GeneralSection__input'
                    value={code}
                    placeholderText={intl.formatMessage({
                        id: 'BoardSettings.general.code-placeholder',
                        defaultMessage: 'e.g., PROJ1',
                    })}
                    onChange={handleCodeChange}
                    onSave={handleCodeSave}
                    saveOnEsc={true}
                    validator={validateCode}
                    spellCheck={false}
                />
                <p className='GeneralSection__help-text'>
                    <FormattedMessage
                        id='BoardSettings.general.code-help'
                        defaultMessage='1-10 alphanumeric characters, must start with a letter'
                    />
                </p>
            </div>

            {/* Board Description */}
            <div className='GeneralSection__field'>
                <label className='GeneralSection__label'>
                    <FormattedMessage
                        id='BoardSettings.general.description'
                        defaultMessage='Description'
                    />
                </label>
                <EditableArea
                    className='GeneralSection__textarea'
                    value={description}
                    placeholderText={intl.formatMessage({
                        id: 'BoardSettings.general.description-placeholder',
                        defaultMessage: 'Add a description...',
                    })}
                    onChange={handleDescriptionChange}
                    onSave={handleDescriptionSave}
                    saveOnEsc={true}
                    spellCheck={true}
                />
            </div>
        </div>
    )
}

export default React.memo(GeneralSection)

