// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback, useEffect} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Board} from '../../blocks/board'
import BoardIconSelector from '../../components/boardIconSelector'
import Editable from '../../widgets/editable'

import './generalSection.scss'

type Props = {
    board: Board
    onBoardChange: (board: Board) => Promise<void> | void
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
            Promise.resolve(props.onBoardChange({...board, title})).catch((err) => {
                // eslint-disable-next-line no-console
                console.error('Failed to update board title:', err)
                setTitle(board.title)
            })
        }
    }, [title, board, props])

    const handleCodeChange = useCallback((newCode: string) => {
        // Enforce max 10 characters on frontend (Issue 1)
        // Always allow edits (including reducing over-length existing codes); truncate to 10
        setCode(newCode.slice(0, 10))
    }, [])

    const handleCodeSave = useCallback(() => {
        if (code !== (board.code || '')) {
            Promise.resolve(props.onBoardChange({...board, code})).catch((err) => {
                // eslint-disable-next-line no-console
                console.error('Failed to update board code:', err)
                setCode(board.code || '')
            })
        }
    }, [code, board, props])

    const handleDescriptionChange = useCallback((newDescription: string) => {
        setDescription(newDescription)
    }, [])

    const handleDescriptionSave = useCallback(() => {
        if (description !== (board.description || '')) {
            Promise.resolve(props.onBoardChange({...board, description})).catch((err) => {
                // eslint-disable-next-line no-console
                console.error('Failed to update board description:', err)
                setDescription(board.description || '')
            })
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
            {/* Board Name and Code - Compact Layout (Issue 1) */}
            <div className='GeneralSection__field'>
                <label className='GeneralSection__label'>
                    <FormattedMessage
                        id='BoardSettings.general.name'
                        defaultMessage='Board Name'
                    />
                </label>
                <div className='GeneralSection__name-code-wrapper'>
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
                    <label className='GeneralSection__code-label'>
                        <FormattedMessage
                            id='BoardSettings.general.code-label'
                            defaultMessage='Code'
                        />
                    </label>
                    <Editable
                        className='GeneralSection__input GeneralSection__input--code'
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
                </div>
            </div>

            {/* Board Description */}
            <div className='GeneralSection__field'>
                <label className='GeneralSection__label'>
                    <FormattedMessage
                        id='BoardSettings.general.description'
                        defaultMessage='Description'
                    />
                </label>
                <Editable
                    className='GeneralSection__input'
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

